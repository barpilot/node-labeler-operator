package operator

import (
	"time"
)

// Config is the controller configuration.
type Config struct {
	// ResyncPeriod is the resync period of the operator.
	ResyncPeriod time.Duration
}

// NewOperatorConfig converts the command line flag arguments to operator configuration.
func NewOperatorConfig(t time.Duration) Config {
	return Config{
		ResyncPeriod: t,
	}
}
