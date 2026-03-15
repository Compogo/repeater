package repeater

import (
	"fmt"

	"github.com/Compogo/runner"
	"github.com/Compogo/types/linker"
	"github.com/spf13/cast"
)

const (
	// Lock strategy ensures only one instance of the task runs at any time.
	// If a task is still running when its next execution is due, it is skipped.
	Lock StrategyType = iota

	// Unlock strategy allows multiple instances of the task to run concurrently.
	// Each execution creates a new task instance with a unique name.
	Unlock
)

var (
	strategies = linker.NewLinker[StrategyType, Strategy]()
)

func init() {
	StrategyRegister(Lock, &lockStrategy{})
	StrategyRegister(Unlock, &unlockStrategy{})
}

// StrategyRegister registers a new strategy implementation.
// This allows extending the repeater with custom strategies.
func StrategyRegister(st StrategyType, strategy Strategy) {
	strategies.Add(st, strategy)
}

// StrategyType defines how a periodic task handles concurrent executions.
type StrategyType uint8

func (s StrategyType) String() string {
	return cast.ToString(s)
}

// Strategy defines the interface for task execution strategies.
// Each strategy decides when a task should run and how to name its instances.
type Strategy interface {
	// IsTaskRun determines whether a new instance of the task should be started.
	IsTaskRun(task *Task, runner runner.Runner) bool

	// GenerateName creates a unique name for a task instance.
	// For Lock strategy, this returns the base name; for Unlock, it appends a counter.
	GenerateName(task *Task) string
}

// lockStrategy implements exclusive execution (only one instance at a time).
type lockStrategy struct{}

func (l *lockStrategy) IsTaskRun(task *Task, runner runner.Runner) bool {
	return !runner.HasTaskByName(task.Name())
}

func (l *lockStrategy) GenerateName(task *Task) string {
	return task.Name()
}

// unlockStrategy implements parallel execution (multiple instances allowed).
type unlockStrategy struct{}

func (u *unlockStrategy) IsTaskRun(_ *Task, _ runner.Runner) bool {
	return true
}

func (u *unlockStrategy) GenerateName(task *Task) string {
	return fmt.Sprintf("%s_%d", task.Name(), task.RunNumbers())
}
