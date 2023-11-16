package queueset

import "github.com/tkashem/apf/pkg/fairqueuing"

type QueueAssignerFactory interface {
	New() (fairqueuing.QueueSelector, error)
}

type Config struct {
	NQueues              int
	QueueMaxLength       int
	QueueAssignerFactory QueueAssignerFactory
}

type CompletedConfig struct {
	Config
	TotalSeats uint32
	Events     Events
}
