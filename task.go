package repeater

import (
	"sync/atomic"
	"time"

	"github.com/Compogo/runner"
)

// Task represents a periodic task managed by the Repeater.
// It contains the task logic, execution schedule, and strategy for concurrent execution.
type Task struct {
	name    string
	process runner.Process
	delay   time.Duration

	strategy StrategyType

	nextExecuteTime atomic.Int64
	runNumbers      atomic.Uint64
}

// NewTaskWithLock creates a new periodic task with Lock strategy.
// Lock strategy ensures only one instance of this task runs at any given time.
// The task will execute every 'delay' duration.
func NewTaskWithLock(name string, process runner.Process, delay time.Duration, options ...Option) *Task {
	return NewTask(name, process, delay, Lock, options...)
}

// NewTaskWithUnlock creates a new periodic task with Unlock strategy.
// Unlock strategy allows multiple instances of this task to run concurrently.
// The task will execute every 'delay' duration.
func NewTaskWithUnlock(name string, process runner.Process, delay time.Duration, options ...Option) *Task {
	return NewTask(name, process, delay, Unlock, options...)
}

// NewTask creates a new periodic task with the specified strategy.
// This is the base constructor that all other constructors use.
func NewTask(name string, process runner.Process, delay time.Duration, strategy StrategyType, options ...Option) *Task {
	task := &Task{name: name, process: process, delay: delay, strategy: strategy}

	for _, option := range options {
		task = option(task)
	}

	return task
}

// String returns the task's name, implementing fmt.Stringer.
func (t *Task) String() string {
	return t.Name()
}

// Name returns the task's identifier.
func (t *Task) Name() string {
	return t.name
}

// Strategy returns the task's execution strategy (Lock/Unlock).
func (t *Task) Strategy() StrategyType {
	return t.strategy
}

// Delay returns the interval between task executions.
func (t *Task) Delay() time.Duration {
	return t.delay
}

// RunNumbers returns the number of times this task has been executed.
func (t *Task) RunNumbers() uint64 {
	return t.runNumbers.Load()
}
