package queueset

import (
	"github.com/tkashem/apf/pkg/fairqueuing"
)

type Config struct {
	TotalSeats    uint32
	QueuingConfig *QueuingConfig
	QueueSelector fairqueuing.QueueSelector
	Events        Events
}

type QueuingConfig struct {
	NQueues        int
	QueueMaxLength int
}
