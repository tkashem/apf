package queueassigner

import (
	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/flow"
)

type Factory func()

// used by the shuffle sharding logic to pick a queue
type QueueSetAccessor interface {
	TotalQueues() int
	GetFairQueue(int) fairqueuing.FairQueue
}

type QueueAssigner interface {
	Assign(flow.RequestFlow) (fairqueuing.FairQueue, error)
}
