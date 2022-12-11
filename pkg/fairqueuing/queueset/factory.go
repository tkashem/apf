package queueset

import (
	"github.com/tkashem/apf/pkg/fairqueuing/queueassigner"
	"github.com/tkashem/apf/pkg/scheduler"
)

type QueueAssignerFactory interface {
	WithQueueSetAccessor(queueassigner.QueueSetAccessor)
	New() (queueassigner.QueueAssigner, error)
}

type Config struct {
	NQueues              int
	QueueMaxLength       int
	QueueAssignerFactory QueueAssignerFactory
}

type CompletedConfig struct {
	Config
	TotalSeats uint32
	Events     scheduler.Events
}
