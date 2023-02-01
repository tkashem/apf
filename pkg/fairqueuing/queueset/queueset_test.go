package queueset

import (
	"context"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
	"testing"

	"github.com/tkashem/apf/pkg/fairqueuing/queueassigner"
)

func Test(t *testing.T) {
	config := &CompletedConfig{
		Config: Config{
			NQueues: 3,
			QueueMaxLength: 128,
			QueueAssignerFactory: &queueassigner.RoundRobinQueueAssignerFactory{},
		},
		TotalSeats: 1,
		Events: nil,
	}

	_, err := NewQueueSet(config)
	if err != nil {
		t.Fatalf("failed to create queueset")
	}
}

func Test2(t *testing.T) {
	input := []struct {
		arrivalRT virtual.SeatSeconds
		width
		queue int
		req

	} {
		{

		},
	}
}

type Interface interface {
	EventLoop(stopCtx context.Context)
	AddEvent(at virtual.SeatSeconds, f EventFunc)
}


type EventFunc func()
