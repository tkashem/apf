package queueset

import (
	"testing"

	"github.com/tkashem/apf/pkg/fairqueuing/queueselector"
)

func Test(t *testing.T) {
	config := &Config{
		QueuingConfig: &QueuingConfig{
			NQueues:        3,
			QueueMaxLength: 128,
		},
		TotalSeats:    1,
		Events:        nil,
		QueueSelector: queueselector.NewRoundRobinQueueSelector(),
	}

	_, err := NewQueueSet(config)
	if err != nil {
		t.Fatalf("failed to create queueset")
	}
}
