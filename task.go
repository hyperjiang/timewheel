package timewheel

import "time"

// Task is the structure of a task in the time wheel.
type Task struct {
	delay time.Duration // the delay time of the task
	cycle int           // the cycles to wait before executing the task
	key   string        // the unique key of the task, used for removing the task
	data  any           // the data of the task
}
