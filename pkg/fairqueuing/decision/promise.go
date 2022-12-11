package decision

import (
	"context"
	"sync"

	"github.com/tkashem/apf/pkg/scheduler"
)

func New(queueTimeoutCtx context.Context, cancel context.CancelFunc) *promise {
	return &promise{
		setCh:           make(chan struct{}),
		queueTimeoutCtx: queueTimeoutCtx,
		cancel:          cancel,
	}
}

type promise struct {
	setCh chan struct{}
	once  sync.Once
	value scheduler.DecisionType

	queueTimeoutCtx context.Context
	cancel          context.CancelFunc
}

func (p *promise) WaitForDecision() scheduler.DecisionType {
	defer p.cancel()
	select {
	case <-p.setCh:
	case <-p.queueTimeoutCtx.Done():
		p.SetDecision(scheduler.DecisionTimeout)
	}
	return p.value
}

func (p *promise) SetDecision(t scheduler.DecisionType) bool {
	var ans bool
	p.once.Do(func() {
		defer close(p.setCh)
		p.value = t
		ans = true
	})
	return ans
}
