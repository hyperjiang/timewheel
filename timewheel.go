package timewheel

import (
	"container/list"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

const logPrefix = "[timewheel]"

var (
	ErrStopped = errors.New("timewheel stopped")
)

// TimeWheel is the main structure of the time wheel.
type TimeWheel struct {
	Options      // inherited options
	ticker       *time.Ticker
	slots        []*list.List         // the slots of the time wheel, each slot is a linked list
	taskIndex    map[string]taskEntry // key: task unique key, value: task entry, O(1) for removal
	currentPos   int                  // the current position of the time wheel (the current slot)
	addTaskCh    chan *Task           // channel for adding tasks
	removeTaskCh chan string          // channel for removing tasks
	stopCh       chan struct{}        // channel for stopping the time wheel
	stopped      bool
	mu           sync.Mutex
}

func New(opts ...Option) *TimeWheel {
	tw := &TimeWheel{
		Options:      NewOptions(opts...),
		taskIndex:    make(map[string]taskEntry),
		addTaskCh:    make(chan *Task),
		removeTaskCh: make(chan string),
		stopCh:       make(chan struct{}),
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
	tw.mu.Lock()
	if tw.stopped {
		tw.mu.Unlock()
		return
	}
	tw.mu.Unlock()
	tw.ticker = time.NewTicker(tw.TickDuration)
	go tw.watch()
}

// Stop stops the time wheel.
func (tw *TimeWheel) Stop() {
	tw.mu.Lock()
	if tw.stopped {
		tw.mu.Unlock()
		return
	}
	tw.stopped = true
	close(tw.stopCh)
	t := tw.ticker
	tw.mu.Unlock()
	if t != nil {
		t.Stop()
	}
}

// AddTask adds a task to the time wheel.
// The task will be executed after the specified delay.
// The data is the payload of the task that will be passed to the handler.
// If the delay is negative, the task will be executed immediately instead of being added to the time wheel.
// The function returns a unique key for the task, which can be used to remove the task later.
func (tw *TimeWheel) AddTask(delay time.Duration, data any) (string, error) {
	key := uuid.NewString()

	if delay < 0 {
		go tw.safeHandle(data)
		return key, nil
	}

	task := &Task{
		scheduledAt: time.Now().Add(delay),
		delay:       delay,
		key:         key,
		data:        data,
	}

	tw.mu.Lock()
	if tw.stopped {
		tw.mu.Unlock()
		return "", ErrStopped
	}
	tw.mu.Unlock()

	select {
	case tw.addTaskCh <- task:
		return key, nil
	case <-tw.stopCh:
		return "", ErrStopped
	}
}

// RemoveTask removes a task from the time wheel.
func (tw *TimeWheel) RemoveTask(key string) error {
	tw.mu.Lock()
	if tw.stopped {
		tw.mu.Unlock()
		return ErrStopped
	}
	tw.mu.Unlock()

	select {
	case tw.removeTaskCh <- key:
		return nil
	case <-tw.stopCh:
		return ErrStopped
	}
}

// GetTaskSize returns the number of tasks in the time wheel.
func (tw *TimeWheel) GetTaskSize() int {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	tw.Logger.Printf("%s[%d] current task size: %d\n", logPrefix, tw.currentPos, len(tw.taskIndex))

	return len(tw.taskIndex)
}

func (tw *TimeWheel) watch() {
	for {
		select {
		case <-tw.ticker.C:
			tw.runTask()
		case task := <-tw.addTaskCh:
			if task == nil {
				return
			}
			tw.addTask(task)
		case key := <-tw.removeTaskCh:
			tw.removeTask(key)
		case <-tw.stopCh:
			return
		}
	}
}

func (tw *TimeWheel) safeHandle(data any) {
	defer func() {
		if r := recover(); r != nil {
			tw.Logger.Printf("%s handler panic recovered: %v", logPrefix, r)
		}
	}()
	tw.Handler.Handle(data)
}

// Calculate the position of the task in the slot and the number of turns the time wheel needs to rotate before executing the task.
func calcPositionAndCycle(delay time.Duration, tick time.Duration, slotNum int, currentPos int) (pos int, cycle int) {
	if delay < 0 {
		return currentPos, 0
	}
	steps := int(delay.Nanoseconds() / tick.Nanoseconds())
	cycle = steps / slotNum
	pos = (currentPos + steps) % slotNum
	return
}

// runTask will be called every tick.
// It scans the current slot and executes the tasks that are due.
// After executing the tasks, it moves to the next slot.
// If it reaches the last slot, it resets to the first slot.
// The tick event is triggered by the ticker.
func (tw *TimeWheel) runTask() {
	var due []*Task

	// scan due tasks first and release the lock,
	// so that we can reduce the lock time
	tw.mu.Lock()
	l := tw.slots[tw.currentPos]
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.cycle > 0 {
			task.cycle--
			e = e.Next()
			continue
		}
		next := e.Next()
		l.Remove(e)
		delete(tw.taskIndex, task.key)
		e = next
		due = append(due, task)
	}

	currentPos := tw.currentPos

	if tw.currentPos == tw.SlotNum-1 { // already at the last slot, reset to 0
		tw.currentPos = 0
	} else {
		tw.currentPos++
	}
	tw.mu.Unlock()

	for _, task := range due {
		tw.Logger.Printf("%s[%d] run task %v, scheduled at %s\n",
			logPrefix, currentPos, task.key, task.scheduledAt.Format("15:04:05.000"))

		go tw.safeHandle(task.data)
	}
}

func (tw *TimeWheel) addTask(task *Task) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.stopped {
		return
	}

	pos, cycle := calcPositionAndCycle(task.delay, tw.TickDuration, tw.SlotNum, tw.currentPos)
	task.cycle = cycle

	// if the task is already in the current position and cycle is 0, execute it immediately
	// this is to avoid the task missing execution due to the lock
	if pos == tw.currentPos && cycle == 0 {
		tw.Logger.Printf("%s[%d] run task %v immediately\n", logPrefix, tw.currentPos, task.key)
		go tw.safeHandle(task.data)
		return
	}

	tw.Logger.Printf("%s[%d] add task %v to position %d, cycle %d, scheduled at %s\n",
		logPrefix, tw.currentPos, task.key, pos, cycle, task.scheduledAt.Format("15:04:05.000"))

	elem := tw.slots[pos].PushBack(task)
	tw.taskIndex[task.key] = taskEntry{slot: pos, elem: elem}
}

func (tw *TimeWheel) removeTask(key string) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	entry, ok := tw.taskIndex[key]
	if !ok {
		return
	}

	l := tw.slots[entry.slot]
	l.Remove(entry.elem)
	delete(tw.taskIndex, key)
}
