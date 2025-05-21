package timewheel

import (
	"container/list"
	"sync"
	"time"
)

// TimeWheel is the main structure of the time wheel.
type TimeWheel struct {
	Options      // inherited options
	ticker       *time.Ticker
	slots        []*list.List // the slots of the time wheel, each slot is a linked list
	taskList     map[any]int  // key: task unique key, value: slot position
	currentPos   int          // the current position of the time wheel (the current slot)
	addTaskCh    chan Task    // channel for adding tasks
	removeTaskCh chan any     // channel for removing tasks
	stopChannel  chan bool    // channel for stopping the time wheel
	mu           sync.Mutex
}

func New(opts ...Option) *TimeWheel {
	tw := &TimeWheel{
		Options:      NewOptions(opts...),
		currentPos:   0,
		taskList:     make(map[any]int),
		addTaskCh:    make(chan Task),
		removeTaskCh: make(chan any),
		stopChannel:  make(chan bool),
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
	go tw.start()
}

// Stop stops the time wheel.
func (tw *TimeWheel) Stop() {
	tw.stopChannel <- true
}

// AddTask adds a task to the time wheel.
// The task will be executed after the specified delay.
// The key is used to identify the task and can be used to remove it later.
// The data is the payload of the task that will be passed to the handler.
// If the delay is negative, the task will not be added.
func (tw *TimeWheel) AddTask(delay time.Duration, key any, data any) {
	if delay < 0 || key == nil {
		tw.Logger.Printf("task %v add failed, delay: %v\n", key, delay)
		return
	}
	tw.addTaskCh <- Task{delay: delay, key: key, data: data}
}

// RemoveTask removes a task from the time wheel.
func (tw *TimeWheel) RemoveTask(key any) {
	if key == nil {
		return
	}
	tw.removeTaskCh <- key
}

func (tw *TimeWheel) start() {
	for {
		select {
		case <-tw.ticker.C:
			tw.tickHandler()
		case task := <-tw.addTaskCh:
			tw.addTask(&task)
		case key := <-tw.removeTaskCh:
			tw.removeTask(key)
		case <-tw.stopChannel:
			tw.ticker.Stop()
			return
		}
	}
}

// tickHandler handles the tick event of the time wheel.
// It scans the current slot and executes the tasks that are due.
// After executing the tasks, it moves to the next slot.
// If it reaches the last slot, it resets to the first slot.
// The tick event is triggered by the ticker.
func (tw *TimeWheel) tickHandler() {
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
		if task.circle > 0 {
			task.circle--
			e = e.Next()
			continue
		}

		go tw.Handler.Handle(task.data)

		next := e.Next()
		l.Remove(e)
		delete(tw.taskList, task.key)
		e = next
	}
}

func (tw *TimeWheel) addTask(task *Task) {
	if task.key != nil {
		tw.mu.Lock()
		if _, exists := tw.taskList[task.key]; exists {
			tw.mu.Unlock()
			tw.Logger.Printf("task key %v already exists, skip\n", task.key)
			return
		}
		tw.mu.Unlock()
	}

	pos, circle := tw.getPositionAndCircle(task.delay)
	task.circle = circle

	tw.Logger.Printf("[%d] task %v add to position %d, circle %d\n", tw.currentPos, task.key, pos, circle)

	// if the task is already in the current position and circle is 0, execute it immediately
	// this is to avoid the task missing execution due to the lock
	if pos == tw.currentPos && circle == 0 {
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
func (tw *TimeWheel) getPositionAndCircle(d time.Duration) (pos int, circle int) {
	delayNs := d.Nanoseconds()
	intervalNs := tw.TickDuration.Nanoseconds()
	steps := int(delayNs / intervalNs)
	circle = steps / tw.SlotNum
	pos = (tw.currentPos + steps) % tw.SlotNum
	return
}

func (tw *TimeWheel) removeTask(key any) {
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
		}

		e = e.Next()
	}
}
