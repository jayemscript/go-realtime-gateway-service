package broker

import (
	"context"
	"fmt"
)

// Broker describes the lifecycle checks needed by the HTTP readiness endpoint.
// Publish and subscription operations are added when event routing is implemented.
type Broker interface {
	Ready(context.Context) error
	Close() error
}

func New(driver string) (Broker, error) {
	switch driver {
	case "memory":
		return NewMemory(), nil
	default:
		return nil, fmt.Errorf("unsupported broker driver %q", driver)
	}
}
