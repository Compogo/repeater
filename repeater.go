package repeater

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Compogo/compogo/logger"
	"github.com/Compogo/runner"
	"github.com/Compogo/types/linker"
	"github.com/Compogo/types/mapper"
	"github.com/Compogo/types/set"
)

// Repeater defines the interface for a periodic task scheduler.
// It manages a collection of tasks, executes them on schedule,
// and ensures proper cleanup on shutdown.
type Repeater interface {
	// Closer stops all tasks and cleans up resources.
	io.Closer

	// Process implements runner.Process, allowing Repeater to be run as a task.
	// It periodically checks for tasks to execute and launches them via Runner.
	runner.Process

	// AddTasks adds multiple periodic tasks to the scheduler.
	AddTasks(tasks ...*Task) error

	// AddTask adds a single periodic task to the scheduler.
	AddTask(task *Task) error

	// RemoveTask removes a periodic task and stops all its running instances.
	RemoveTask(task *Task) error

	// RemoveTaskByName removes a periodic task by name and stops all its instances.
	RemoveTaskByName(name string) error

	// HasTaskByName checks if a periodic task with the given name exists.
	HasTaskByName(name string) bool

	// HasTask checks if the specific task instance exists in the scheduler.
	HasTask(task *Task) bool

	// Use registers middlewares that wrap all periodic task executions.
	Use(middlewares ...Middleware)
}

type repeater struct {
	tasks      *mapper.Mapper[*Task]
	taskNames  *linker.Linker[*Task, set.Set[string]]
	middleware []Middleware

	ticker     *time.Ticker
	cancelFunc context.CancelFunc
	rwMutex    sync.RWMutex

	logger logger.Logger
	runner runner.Runner
}

// NewRepeater creates a new Repeater instance.
// It requires:
//   - config: ticker interval configuration
//   - logger: for logging repeater events
//   - runner: for executing task instances
func NewRepeater(config *Config, logger logger.Logger, runner runner.Runner) Repeater {
	return &repeater{
		tasks:     mapper.NewMapper[*Task](),
		taskNames: linker.NewLinker[*Task, set.Set[string]](),
		ticker:    time.NewTicker(config.Delay),
		logger:    logger.GetLogger("repeater"),
		runner:    runner,
	}
}

func (r *repeater) Process(ctx context.Context) (err error) {
	ctx, r.cancelFunc = context.WithCancel(ctx)
	defer r.cancelFunc()
	defer func() {
		err = r.removeAll()
	}()

	for {
		if err := r.process(); err != nil {
			r.logger.Errorf("processing failed: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-r.ticker.C:
			continue
		}
	}
}

func (r *repeater) process() (err error) {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()

	now := time.Now()
	timestamp := now.UnixNano()

	for _, task := range r.tasks.All() {
		if task.nextExecuteTime.Load() <= timestamp {
			strategy, strategyErr := strategies.Get(task.strategy)
			if strategyErr != nil {
				err = fmt.Errorf("%s: run failed %w", task.name, strategyErr)
				continue
			}

			if !strategy.IsTaskRun(task, r.runner) {
				continue
			}

			name := strategy.GenerateName(task)
			process := r.taskProcess(name, task)

			task.runNumbers.Add(1)
			task.nextExecuteTime.Store(now.Add(task.Delay()).UnixNano())

			taskNames, _ := r.taskNames.Get(task)
			taskNames.Add(name)

			for _, middleware := range r.middleware {
				process = middleware.Middleware(task, process)
			}

			if runnerErr := r.runner.RunTask(runner.NewTask(name, process)); runnerErr != nil {
				err = fmt.Errorf("%s: run failed %w", task.name, runnerErr)
			}
		}
	}

	return err
}

func (r *repeater) Close() (err error) {
	if r.cancelFunc != nil {
		r.cancelFunc()
	}

	return nil
}

func (r *repeater) removeAll() (err error) {
	r.rwMutex.RLock()
	tasks := r.tasks
	r.rwMutex.RUnlock()

	for _, task := range tasks.All() {
		if err = r.RemoveTask(task); err != nil {
			return err
		}
	}

	return nil
}

func (r *repeater) taskProcess(name string, task *Task) runner.Process {
	return runner.ProcessFunc(func(ctx context.Context) (err error) {
		defer func() {
			r.rwMutex.Lock()
			defer r.rwMutex.Unlock()

			taskNames, _ := r.taskNames.Get(task)
			taskNames.Remove(name)
		}()

		return task.process.Process(ctx)
	})
}

func (r *repeater) AddTasks(tasks ...*Task) (err error) {
	for _, task := range tasks {
		if err = r.AddTask(task); err != nil {
			return err
		}
	}

	return nil
}

func (r *repeater) AddTask(task *Task) error {
	if r.HasTask(task) {
		return fmt.Errorf("[repeater] task '%s' %w", task.Name(), runner.TaskAlreadyExistsError)
	}

	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()

	r.tasks.Add(task)
	r.taskNames.Add(task, set.NewSet[string]())

	return nil
}

func (r *repeater) RemoveTaskByName(name string) error {
	if !r.HasTaskByName(name) {
		return fmt.Errorf("[repeater] task '%s' %w", name, runner.TaskUndefinedError)
	}

	r.rwMutex.RLock()
	task, _ := r.tasks.Get(name)
	r.rwMutex.RUnlock()

	return r.RemoveTask(task)
}

func (r *repeater) RemoveTask(task *Task) (err error) {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()

	r.tasks.RemoveByValue(task)
	taskNames, err := r.taskNames.Get(task)
	if err != nil {
		return fmt.Errorf("[repeater] task '%s' remove failed: %w", task.Name(), err)
	}

	for name := range taskNames {
		if rerr := r.runner.StopTaskByName(name); rerr != nil {
			err = fmt.Errorf("%w\n[repeater] task '%s' remove failed: %w", err, name, rerr)
		}
	}

	return err
}

func (r *repeater) HasTaskByName(name string) bool {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()

	return r.tasks.HasByKey(name)
}

func (r *repeater) HasTask(task *Task) bool {
	r.rwMutex.RLock()
	defer r.rwMutex.RUnlock()

	return r.tasks.HasByValue(task)
}

func (r *repeater) Use(middlewares ...Middleware) {
	r.rwMutex.Lock()
	defer r.rwMutex.Unlock()

	r.middleware = append(r.middleware, middlewares...)
}
