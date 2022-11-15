package decision

import (
	"sync"

	"github.com/tkashem/apf/pkg/core"
)

func New(timeoutCh <-chan struct{}) *promise {
	return &promise{
		setCh:     make(chan struct{}),
		timeoutCh: timeoutCh,
	}
}

type promise struct {
	setCh     chan struct{}
	timeoutCh <-chan struct{}
	once      sync.Once
	value     core.DecisionType
}

func (p *promise) Get() core.DecisionType {
	select {
	case <-p.setCh:
	case <-p.timeoutCh:
		p.Set(core.DecisionTimeout)
	}
	return p.value
}

func (p *promise) Set(t core.DecisionType) bool {
	var ans bool
	p.once.Do(func() {
		defer close(p.setCh)
		p.value = t
		ans = true
	})
	return ans
}
