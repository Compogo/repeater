package repeater

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Compogo/compogo/logger"
	"github.com/Compogo/repeater/infrastructure/config"
	"github.com/Compogo/runner"
	"github.com/Compogo/types/linker"
)

type Repeater interface {
	runner.Process

	io.Closer
	AddProcess(Process) error
	AddProcesses(...Process) error
	StopProcess(Process) error
	StopProcessByName(string) error
	HasProcess(Process) bool
	HasProcessByName(string) bool
}

type repeater struct {
	ticker  *time.Ticker
	rwMutex sync.RWMutex

	processes     *linker.Linker[Process, int64]
	linkProcesses *linker.Linker[string, Process]

	runner runner.Runner
	logger logger.Logger
}

func newRepeater(runner runner.Runner, logger logger.Logger, config *config.Config) *repeater {
	return &repeater{
		ticker:        time.NewTicker(config.Delay),
		processes:     linker.NewLinker[Process, int64](),
		linkProcesses: linker.NewLinker[string, Process](linker.KeyStringNormalizer[Process]()),
		runner:        runner,
		logger:        logger,
	}
}

func (repeater *repeater) Close() (err error) {
	repeater.rwMutex.RLock()
	processes := repeater.processes
	repeater.rwMutex.RUnlock()

	for process := range processes.All() {
		if err = repeater.StopProcess(process); err != nil {
			return err
		}
	}

	return nil
}

func (repeater *repeater) AddProcesses(process ...Process) (err error) {
	for _, process := range process {
		if err = repeater.AddProcess(process); err != nil {
			return err
		}
	}

	return nil
}

func (repeater *repeater) AddProcess(process Process) error {
	if repeater.HasProcess(process) {
		return fmt.Errorf("[repeater] process '%s': %w", process.Name(), runner.TaskAlreadyExistsError)
	}

	repeater.rwMutex.Lock()
	defer repeater.rwMutex.Unlock()

	repeater.processes.Add(process, time.Now().UTC().UnixNano())
	repeater.linkProcesses.Add(process.Name(), process)

	return nil
}

func (repeater *repeater) StopProcessByName(name string) (err error) {
	var p Process
	repeater.rwMutex.RLock()
	p, err = repeater.linkProcesses.Get(name)
	repeater.rwMutex.RUnlock()

	if err != nil {
		return err
	}

	return repeater.StopProcess(p)
}

func (repeater *repeater) StopProcess(process Process) (err error) {
	repeater.rwMutex.Lock()
	defer repeater.rwMutex.Unlock()

	if err = process.Close(); err != nil {
		return err
	}

	repeater.processes.Remove(process)
	repeater.linkProcesses.Remove(process.Name())

	return nil
}

func (repeater *repeater) HasProcess(process Process) bool {
	repeater.rwMutex.RLock()
	defer repeater.rwMutex.RUnlock()

	return repeater.processes.Has(process)
}

func (repeater *repeater) HasProcessByName(name string) bool {
	repeater.rwMutex.RLock()
	defer repeater.rwMutex.RUnlock()

	return repeater.linkProcesses.Has(name)
}

func (repeater *repeater) Process(ctx context.Context) (err error) {
	for {
		if err = repeater.process(); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-repeater.ticker.C:
			continue
		}
	}
}

func (repeater *repeater) process() (err error) {
	repeater.rwMutex.Lock()
	defer repeater.rwMutex.Unlock()

	now := time.Now()
	timestamp := now.UnixNano()

	for process, nextTimestamp := range repeater.processes.All() {
		if nextTimestamp <= timestamp {
			nextTime := now.Add(process.Delay())

			err = repeater.runner.RunProcess(process)
			if err != nil && errors.Is(err, runner.TaskAlreadyExistsError) {
				repeater.logger.Debugf("process '%s' already execute, next time execute %s", process.Name(), nextTime.Format(time.RFC3339Nano))
				repeater.processes.Add(process, nextTime.UnixNano())
				continue
			}

			if err != nil {
				return err
			}

			repeater.logger.Infof("process '%s' next time execute %s", process.Name(), nextTime.Format(time.RFC3339Nano))
			repeater.processes.Add(process, nextTime.UnixNano())
		}
	}

	return nil
}

func (repeater *repeater) Name() string {
	return "repeater"
}
