package queueassigner

import (
	"fmt"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
)

func NewShuffleShardingQueueSelector(deckSize, handSize int) (*shuffleShardingQueueSelector, error) {
	dealer, err := NewDealer(deckSize, handSize)
	if err != nil {
		return nil, err
	}
	return &shuffleShardingQueueSelector{
		dealer: dealer,
	}, nil
}

type shuffleShardingQueueSelector struct {
	dealer *Dealer
}

func (s *shuffleShardingQueueSelector) SelectQueue(queues fairqueuing.FairQueueAccessor, hash fairqueuing.FlowIDType) (fairqueuing.FairQueue, error) {
	if s.dealer.DeckSize() != queues.TotalQueues() {
		dealer, err := NewDealer(s.dealer.DeckSize(), s.dealer.HandSize())
		if err != nil {
			return nil, err
		}
		s.dealer = dealer
	}

	var backHand [8]int
	// Deal into a data structure, so that the order of visit below is not necessarily the order of the deal.
	// This removes bias in the case of flows with overlapping hands.
	hand := s.dealer.DealIntoHand(uint64(hash), backHand[:])

	// select the queue that has minimum work
	minimum := virtual.MaxSeatSeconds
	var minimumQueue fairqueuing.FairQueue
	for _, queueIndex := range hand {
		queue := queues.GetFairQueue(queueIndex)
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
