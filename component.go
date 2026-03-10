package repeater

import (
	"github.com/Compogo/compogo/component"
	"github.com/Compogo/compogo/container"
	"github.com/Compogo/compogo/flag"
	"github.com/Compogo/runner"
)

// Component is a ready-to-use Compogo component that provides the Repeater.
// It automatically:
//   - Registers Config and Repeater in the DI container
//   - Adds command-line flags for ticker interval
//   - Starts the Repeater as a runner task during Run phase
//   - Stops the Repeater during Stop phase
//
// Usage:
//
//	compogo.WithComponents(
//	    runner.Component,
//	    repeater.Component,
//	)
var Component = &component.Component{
	Dependencies: component.Components{
		runner.Component,
	},
	Init: component.StepFunc(func(container container.Container) error {
		return container.Provides(
			NewConfig,
			NewRepeater,
		)
	}),
	BindFlags: component.BindFlags(func(flagSet flag.FlagSet, container container.Container) error {
		return container.Invoke(func(config *Config) {
			flagSet.DurationVar(&config.Delay, DelayFieldName, DelayDefault, "")
		})
	}),
	PreRun: component.StepFunc(func(container container.Container) error {
		return container.Invoke(Configuration)
	}),
	Run: component.StepFunc(func(container container.Container) error {
		return container.Invoke(func(r runner.Runner, repeater Repeater) error {
			return r.RunTask(runner.NewTask("repeater", repeater))
		})
	}),
	Stop: component.StepFunc(func(container container.Container) error {
		return container.Invoke(func(repeater Repeater) error {
			return repeater.Close()
		})
	}),
}
