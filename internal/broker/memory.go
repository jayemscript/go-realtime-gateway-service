package broker

import "context"

// Memory is the local broker used until Redis is implemented.
type Memory struct{}

func NewMemory() *Memory {
	return &Memory{}
}

func (m *Memory) Ready(context.Context) error {
	return nil
}

func (m *Memory) Close() error {
	return nil
}
