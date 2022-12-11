package queueassigner

import (
	"fmt"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/flow"
)

type RoundRobinQueueAssignerFactory struct {
	accessor QueueSetAccessor
}

func (f *RoundRobinQueueAssignerFactory) WithQueueSetAccessor(accessor QueueSetAccessor) {
	f.accessor = accessor
}

func (f *RoundRobinQueueAssignerFactory) New() (QueueAssigner, error) {
	return &robinQueueAssigner{
		accessor: f.accessor,
	}, nil
}

type robinQueueAssigner struct {
	accessor QueueSetAccessor
	robinIdx int
}

func (p *robinQueueAssigner) Assign(_ flow.RequestFlow) (fairqueuing.FairQueue, error) {
	total := p.accessor.TotalQueues()
	if total <= 0 {
		return nil, fmt.Errorf("number of queues in the queueset cannot be zero")
	}

	p.robinIdx = (p.robinIdx + 1) % total
	return p.accessor.GetFairQueue(p.robinIdx), nil
}
