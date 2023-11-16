package queueset

import (
	"testing"

	"github.com/tkashem/apf/pkg/fairqueuing/queueassigner"
)

func Test(t *testing.T) {
	config := &CompletedConfig{
		Config: Config{
			NQueues:              3,
			QueueMaxLength:       128,
			QueueAssignerFactory: &queueassigner.RoundRobinQueueAssignerFactory{},
		},
		TotalSeats: 1,
		Events:     nil,
	}

	_, err := NewQueueSet(config)
	if err != nil {
		t.Fatalf("failed to create queueset")
	}
}
