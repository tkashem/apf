package queued

import (
	"fmt"
	"net/http"

	apfcontext "github.com/tkashem/apf/pkg/context"
	"github.com/tkashem/apf/pkg/fairqueuing"
	apfdecision "github.com/tkashem/apf/pkg/fairqueuing/decision"
	"github.com/tkashem/apf/pkg/scheduler"
)

func NewQueuedSchedulerFactory(qs fairqueuing.FairQueueSet) (*scheduler.SimpleSchedulerFactory, error) {
	if qs == nil {
		return nil, fmt.Errorf("cannot be nil")
	}

	return &scheduler.SimpleSchedulerFactory{
		Scheduler: &queuedScheduler{
			queueset:       qs,
			queueTimeoutFn: queueTimeoutContext,
		},
	}, nil
}

type queuedScheduler struct {
	queueset       fairqueuing.FairQueueSet
	queueTimeoutFn QueueTimeoutContextFunc
}

func (s *queuedScheduler) Schedule(r *http.Request) (scheduler.Finisher, error) {
	scoped, err := apfcontext.RequestScopedFrom(r.Context())
	if err != nil {
		return nil, err
	}

	// 1) shuffle sharding, pick a queue
	// 2) Reject old requests that have been waiting too long
	// 3) Reject current request if there is not enough concurrency shares and
	// we are at max queue length
	// 4) If not rejected, create a request and enqueue
	callbacks, err := s.queueset.Enqueue(r)
	if err != nil {
		// log the error
		return scheduler.RejectedRequest{}, nil
	}

	queueTimeoutCtx, cancel := s.queueTimeoutFn(r)
	decision := apfdecision.New(queueTimeoutCtx, cancel)
	scoped.Decision = decision

	s.queueset.Dispatch()

	return &queuedRequest{
		decision:  decision,
		callbacks: callbacks,
	}, nil
}
