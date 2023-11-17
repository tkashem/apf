package queueset

import (
	"github.com/tkashem/apf/pkg/fairqueuing"

	"k8s.io/utils/clock"
)

type Config struct {
	TotalSeats    uint32
	QueuingConfig *QueuingConfig
	QueueSelector fairqueuing.QueueSelector
	Clock         clock.Clock
	Events        Events
}

type QueuingConfig struct {
	NQueues        int
	QueueMaxLength int
}
