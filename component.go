package repeater

import (
	"github.com/Compogo/compogo/component"
	"github.com/Compogo/compogo/container"
	"github.com/Compogo/repeater/infrastructure/config"
	"github.com/Compogo/runner"
	"github.com/Compogo/runner/middleware/restore"
)

var (
	Component = &component.Component{
		Dependencies: component.Components{
			runner.Component,
			restore.ComponentNoRegistration,
			config.Component,
		},
		Init: component.StepFunc(func(container container.Container) error {
			return container.Provides(
				newRepeater,
				func(r *repeater) Repeater { return r },
			)
		}),
		Execute: component.StepFunc(func(container container.Container) error {
			return container.Invoke(func(r runner.Runner, restore *restore.Restore, repeater Repeater) error {
				return r.RunProcess(runner.NewTask(repeater.Name(), repeater.Process, restore))
			})
		}),
		Stop: component.StepFunc(func(container container.Container) error {
			return container.Invoke(func(repeater Repeater) error {
				return repeater.Close()
			})
		}),
	}

	ComponentNoRestore = &component.Component{
		Dependencies: component.Components{
			runner.Component,
			config.Component,
		},
		Init: component.StepFunc(func(container container.Container) error {
			return container.Provides(
				newRepeater,
				func(r *repeater) Repeater { return r },
			)
		}),
		Execute: component.StepFunc(func(container container.Container) error {
			return container.Invoke(func(r runner.Runner, repeater Repeater) error {
				return r.RunProcess(repeater)
			})
		}),
		Stop: component.StepFunc(func(container container.Container) error {
			return container.Invoke(func(repeater Repeater) error {
				return repeater.Close()
			})
		}),
	}
)
