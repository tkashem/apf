package queueassigner

import (
	"fmt"

	"github.com/tkashem/apf/pkg/fairqueuing"
)

func NewRoundRobinQueueSelector() *robinQueueSelector {
	return &robinQueueSelector{}
}

var _ fairqueuing.QueueSelector = &robinQueueSelector{}

type robinQueueSelector struct {
	robinIdx int
}

func (p *robinQueueSelector) SelectQueue(queues fairqueuing.FairQueueAccessor, _ fairqueuing.FlowIDType) (fairqueuing.FairQueue, error) {
	total := queues.TotalQueues()
	if total <= 0 {
		return nil, fmt.Errorf("number of queues in the queueset cannot be zero")
	}

	p.robinIdx = (p.robinIdx + 1) % total
	return queues.GetFairQueue(p.robinIdx), nil
}
