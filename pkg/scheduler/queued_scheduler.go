package scheduler

import (
	"fmt"
	"net/http"

	"github.com/tkashem/apf/pkg/core"
	apfdecision "github.com/tkashem/apf/pkg/decision"
	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/shufflesharding"
)

func NewQueuedSchedulerFactory(totalSeats int32, queueMaxLength int, nQueues int, events core.SchedulingEvents) (*core.SimpleSchedulerFactory, error) {
	if totalSeats < 1 {
		return nil, fmt.Errorf("seats must be positive")
	}
	return &core.SimpleSchedulerFactory{
		Scheduler: &queuedScheduler{
			queueset:      fairqueuing.NewQueueset(totalSeats, queueMaxLength, nQueues, events),
			queueAssigner: shufflesharding.NewRoundRobinQueueAssigner(),
		},
	}, nil
}

type queuedScheduler struct {
	queueset      core.FairQueueSet
	queueAssigner core.QueueAssigner
}

func (s *queuedScheduler) Schedule(r *http.Request) (core.Finisher, error) {
	scoped, err := core.RequestScopedFrom(r.Context())
	if err != nil {
		return nil, err
	}

	// 1) shuffle sharding, pick a queue
	// 2) Reject old requests that have been waiting too long
	// 3) Reject current request if there is not enough concurrency shares and
	// we are at max queue length
	// 4) If not rejected, create a request and enqueue
	callbacks, err := s.queueset.Enqueue(r, s.queueAssigner)
	if err != nil {
		// log the error
		return rejectedRequest{}, nil
	}

	decision := apfdecision.New(r.Context().Done())
	scoped.Decision = decision

	s.queueset.Dispatch()

	return &queuedRequest{
		decision:  decision,
		callbacks: callbacks,
	}, nil
}
