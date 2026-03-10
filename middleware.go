package repeater

import "github.com/Compogo/runner"

// Middleware defines the interface for task middleware.
// Middlewares can wrap a task's Process function to add cross-cutting concerns
// such as logging, metrics, or error handling specific to periodic tasks.
type Middleware interface {
	// Middleware wraps a periodic task's process function.
	// It receives the periodic task and the next Process in the chain,
	// and returns a new Process that will be executed instead.
	Middleware(task *Task, next runner.Process) runner.Process
}

// MiddlewareFunc is a function adapter that allows ordinary functions to be
// used as Middleware implementations for periodic tasks.
type MiddlewareFunc func(task *Task, next runner.Process) runner.Process

// Middleware implements the Middleware interface by calling the underlying function.
func (m MiddlewareFunc) Middleware(task *Task, next runner.Process) runner.Process {
	return m(task, next)
}
