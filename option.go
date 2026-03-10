package repeater

import "time"

// Option is a function that configures a Task during creation.
// Options are applied in the order they are provided.
type Option func(task *Task) *Task

// SkipFirstRun configures the task to skip its first scheduled execution.
// Useful when you want tasks to start after a delay rather than immediately.
func SkipFirstRun(task *Task) *Task {
	task.nextExecuteTime.Store(time.Now().Add(task.Delay()).UnixNano())

	return task
}
