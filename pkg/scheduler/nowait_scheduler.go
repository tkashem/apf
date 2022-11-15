package scheduler

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/tkashem/apf/pkg/core"
)

func NewNoWaitSchedulerFactory(seats int32, events core.SchedulingEvents) (*core.SimpleSchedulerFactory, error) {
	if seats < 1 {
		return nil, fmt.Errorf("seats must be positive")
	}
	return &core.SimpleSchedulerFactory{
		Scheduler: &noWaitScheduler{
			totalSeats: seats,
			events:     events,
		},
	}, nil
}

type noWaitScheduler struct {
	lock                    sync.Mutex
	totalSeats              int32
	seatsInUse              int32
	currentRequestsInFlight int32
	events                  core.SchedulingEvents
}

func (s *noWaitScheduler) Schedule(r *http.Request) (core.Finisher, error) {
	scoped, err := core.RequestScopedFrom(r.Context())
	if err != nil {
		return nil, err
	}
	seatsRequested := scoped.Estimate.Seats

	s.lock.Lock()
	defer s.lock.Unlock()

	if seatsRequested+s.seatsInUse > s.totalSeats {
		return rejectedRequest{}, nil
	}
	s.seatsInUse += seatsRequested
	s.currentRequestsInFlight += 1

	return immediateRequest{
		disposer: core.DisposerFunc(func() {
			defer s.events.Disposed(r)
			func() {
				s.lock.Lock()
				defer s.lock.Unlock()
				s.seatsInUse -= seatsRequested
				s.currentRequestsInFlight -= 1
			}()
		}),
	}, nil
}
