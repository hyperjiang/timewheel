package timewheel

import (
	"container/list"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TimeWheel is the main structure of the time wheel.
type TimeWheel struct {
	Options      // inherited options
	ticker       *time.Ticker
	slots        []*list.List   // the slots of the time wheel, each slot is a linked list
	taskList     map[string]int // key: task unique key, value: slot position
	currentPos   int            // the current position of the time wheel (the current slot)
	addTaskCh    chan Task      // channel for adding tasks
	removeTaskCh chan string    // channel for removing tasks
	stopCh       chan bool      // channel for stopping the time wheel
	mu           sync.Mutex
}

func New(opts ...Option) *TimeWheel {
	tw := &TimeWheel{
		Options:      NewOptions(opts...),
		currentPos:   0,
		taskList:     make(map[string]int),
		addTaskCh:    make(chan Task),
		removeTaskCh: make(chan string),
		stopCh:       make(chan bool),
	}
	tw.slots = make([]*list.List, tw.SlotNum)

	tw.initSlots()

	return tw
}

// initSlots initializes the slots of the time wheel.
// Each slot is a linked list that holds the tasks scheduled for that slot.
func (tw *TimeWheel) initSlots() {
	for i := 0; i < tw.SlotNum; i++ {
		tw.slots[i] = list.New()
	}
}

// Start starts the time wheel.
func (tw *TimeWheel) Start() {
	tw.ticker = time.NewTicker(tw.TickDuration)
	go tw.watch()
}

// Stop stops the time wheel.
func (tw *TimeWheel) Stop() {
	tw.stopCh <- true
}

// AddTask adds a task to the time wheel.
// The task will be executed after the specified delay.
// The data is the payload of the task that will be passed to the handler.
// If the delay is negative, the task will not be added.
// The function returns a unique key for the task, which can be used to remove the task later.
func (tw *TimeWheel) AddTask(delay time.Duration, data any) string {
	key := uuid.NewString()
	tw.addTaskCh <- Task{delay: delay, key: key, data: data}

	return key
}

// RemoveTask removes a task from the time wheel.
func (tw *TimeWheel) RemoveTask(key string) {
	tw.removeTaskCh <- key
}

// GetTaskSize returns the number of tasks in the time wheel.
func (tw *TimeWheel) GetTaskSize() int {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.Logger.Printf("[%d] current task size: %d\n", tw.currentPos, len(tw.taskList))

	return len(tw.taskList)
}

func (tw *TimeWheel) watch() {
	for {
		select {
		case <-tw.ticker.C:
			tw.runTask()
		case task := <-tw.addTaskCh:
			tw.addTask(&task)
		case key := <-tw.removeTaskCh:
			tw.removeTask(key)
		case <-tw.stopCh:
			tw.ticker.Stop()
			return
		}
	}
}

// runTask will be called every tick.
// It scans the current slot and executes the tasks that are due.
// After executing the tasks, it moves to the next slot.
// If it reaches the last slot, it resets to the first slot.
// The tick event is triggered by the ticker.
func (tw *TimeWheel) runTask() {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	l := tw.slots[tw.currentPos]
	tw.scanAndRunTask(l)
	if tw.currentPos == tw.SlotNum-1 { // already at the last slot, reset to 0
		tw.currentPos = 0
	} else {
		tw.currentPos++
	}
}

func (tw *TimeWheel) scanAndRunTask(l *list.List) {
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.cycle > 0 {
			task.cycle--
			e = e.Next()
			continue
		}

		task.delay -= tw.TickDuration
		go tw.Handler.Handle(task.data)

		next := e.Next()
		l.Remove(e)
		delete(tw.taskList, task.key)
		e = next
	}
}

func (tw *TimeWheel) addTask(task *Task) {
	pos, cycle := tw.getPositionAndCycle(task.delay)
	task.cycle = cycle

	tw.Logger.Printf("[%d] task %v add to position %d, cycle %d\n", tw.currentPos, task.key, pos, cycle)

	// if the task is already in the current position and cycle is 0, execute it immediately
	// this is to avoid the task missing execution due to the lock
	if pos == tw.currentPos && cycle == 0 {
		tw.Logger.Printf("[%d] task %v execute immediately\n", tw.currentPos, task.key)
		go tw.Handler.Handle(task.data)
		return
	}

	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.slots[pos].PushBack(task)
	tw.taskList[task.key] = pos
}

// Get the position of the task in the slot and the number of turns the time wheel needs to rotate before executing the task.
func (tw *TimeWheel) getPositionAndCycle(d time.Duration) (pos int, cycle int) {
	if d < 0 {
		cycle = 0
		pos = tw.currentPos
		return
	}

	delayNs := d.Nanoseconds()
	intervalNs := tw.TickDuration.Nanoseconds()
	steps := int(delayNs / intervalNs)

	cycle = steps / tw.SlotNum
	pos = (tw.currentPos + steps) % tw.SlotNum

	return
}

func (tw *TimeWheel) removeTask(key string) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	position, ok := tw.taskList[key]
	if !ok {
		return
	}

	l := tw.slots[position]
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.key == key {
			delete(tw.taskList, task.key)
			l.Remove(e)
			return
		}

		e = e.Next()
	}
}
