package queueassigner

import (
	"fmt"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/flow"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
)

type ShuffleShardedQueueAssignerFactory struct {
	accessor QueueSetAccessor
	handSize int
}

func (f ShuffleShardedQueueAssignerFactory) WithHandSize(handSize int) {
	f.handSize = handSize
}

func (f ShuffleShardedQueueAssignerFactory) WithQueueSetAccessor(accessor QueueSetAccessor) {
	f.accessor = accessor
}

func (f ShuffleShardedQueueAssignerFactory) New() (*shuffleShardingQueueAssigner, error) {
	if f.accessor == nil {
		return nil, fmt.Errorf("QueueSetAccessor can not be nil")
	}

	dealer, err := NewDealer(f.accessor.TotalQueues(), f.handSize)
	if err != nil {
		return nil, err
	}
	return &shuffleShardingQueueAssigner{
		dealer: dealer,
	}, nil
}

type shuffleShardingQueueAssigner struct {
	dealer   *Dealer
	queueset QueueSetAccessor
}

func (s *shuffleShardingQueueAssigner) Assign(rf flow.RequestFlow) (fairqueuing.FairQueue, error) {
	var backHand [8]int
	// Deal into a data structure, so that the order of visit below is not necessarily the order of the deal.
	// This removes bias in the case of flows with overlapping hands.
	hand := s.dealer.DealIntoHand(uint64(rf.Hash), backHand[:])

	// select the queue that has minimum work
	minimum := virtual.MaxSeatSeconds
	var minimumQueue fairqueuing.FairQueue
	for _, queueIndex := range hand {
		queue := s.queueset.GetFairQueue(queueIndex)
		if queue == nil {
			return nil, fmt.Errorf("queue returned by QueueSetAccessor is nil")
		}

		nextFinishR := queue.GetNextFinishR()
		if nextFinishR < minimum {
			minimum = nextFinishR
			minimumQueue = queue
		}
	}
	return minimumQueue, nil
}
