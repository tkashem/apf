package shufflesharding

import (
	"github.com/tkashem/apf/pkg/core"
)

func NewRoundRobinQueueAssigner() *robinQueueAssigner {
	return &robinQueueAssigner{}
}

type robinQueueAssigner struct {
	robinIdx int
}

func (p *robinQueueAssigner) Assign(accessor core.FairQueueAccessor) core.FairQueue {
	total := accessor.Total()
	if total <= 0 {
		return nil
	}

	p.robinIdx = (p.robinIdx + 1) % total
	return accessor.Get(p.robinIdx)
}
