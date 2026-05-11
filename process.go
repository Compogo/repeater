package repeater

import (
	"time"

	"github.com/Compogo/runner"
)

type Process interface {
	runner.Process
	Delay() time.Duration
	RunNumbers() uint64
}
