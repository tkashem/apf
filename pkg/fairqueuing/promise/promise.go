package promise

import (
	"context"
	"sync"

	"github.com/tkashem/apf/pkg/fairqueuing"
)

func New(ctx context.Context) *promise {
	return &promise{
		setCh:           make(chan struct{}),
		queueTimeoutCtx: ctx,
	}
}

type promise struct {
	setCh           chan struct{}
	once            sync.Once
	value           fairqueuing.DecisionType
	queueTimeoutCtx context.Context
}

func (p *promise) WaitForDecision() fairqueuing.DecisionType {
	select {
	case <-p.setCh:
	case <-p.queueTimeoutCtx.Done():
		p.SetDecision(fairqueuing.DecisionTimeout)
	}
	return p.value
}

func (p *promise) SetDecision(t fairqueuing.DecisionType) bool {
	var ans bool
	p.once.Do(func() {
		defer close(p.setCh)
		p.value = t
		ans = true
	})
	return ans
}
