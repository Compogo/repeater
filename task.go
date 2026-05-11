package repeater

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Compogo/runner"
)

type Task struct {
	m sync.Mutex

	name        string
	processFunc runner.ProcessFunc

	ctx        context.Context
	cancelFunc context.CancelFunc

	middlewares []runner.Middleware

	delay      time.Duration
	runNumbers atomic.Uint64
}

func NewTask(name string, delay time.Duration, processFunc runner.ProcessFunc, middlewares ...runner.Middleware) *Task {
	return &Task{
		name:        name,
		processFunc: processFunc,
		middlewares: middlewares,
		delay:       delay,
	}
}

func (task *Task) Close() error {
	task.m.Lock()
	defer task.m.Unlock()

	if task.cancelFunc != nil {
		task.cancelFunc()
	}

	return nil
}

func (task *Task) Process(ctx context.Context) error {
	task.m.Lock()
	defer task.m.Unlock()

	task.ctx, task.cancelFunc = context.WithCancel(ctx)
	defer task.cancelFunc()

	processFunc := task.processFunc
	for i := len(task.middlewares) - 1; i >= 0; i-- {
		processFunc = task.middlewares[i].Middleware(task, processFunc)
	}

	task.runNumbers.Add(1)
	return processFunc(task.ctx)
}

func (task *Task) Name() string {
	return task.name
}

func (task *Task) Delay() time.Duration {
	return task.delay
}

func (task *Task) RunNumbers() uint64 {
	return task.runNumbers.Load()
}
