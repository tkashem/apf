package nowait

import (
	"fmt"
	"net/http"
	"sync"

	apfcontext "github.com/tkashem/apf/pkg/context"
	"github.com/tkashem/apf/pkg/scheduler"
)

func NewNoWaitSchedulerFactory(seats uint32, events scheduler.Events) (*scheduler.SimpleSchedulerFactory, error) {
	if seats < 1 {
		return nil, fmt.Errorf("seats must be positive")
	}
	return &scheduler.SimpleSchedulerFactory{
		Scheduler: &noWaitScheduler{
			totalSeats: seats,
			events:     events,
		},
	}, nil
}

type noWaitScheduler struct {
	lock                    sync.Mutex
	totalSeats              uint32
	seatsInUse              uint32
	currentRequestsInFlight uint32
	events                  scheduler.Events
}

func (s *noWaitScheduler) Schedule(r *http.Request) (scheduler.Finisher, error) {
	scoped, err := apfcontext.RequestScopedFrom(r.Context())
	if err != nil {
		return nil, err
	}
	seatsRequested := scoped.Estimate.GetSeats()

	s.lock.Lock()
	defer s.lock.Unlock()

	if seatsRequested+s.seatsInUse > s.totalSeats {
		return scheduler.RejectedRequest{}, nil
	}
	s.seatsInUse += seatsRequested
	s.currentRequestsInFlight += 1

	return immediateRequest{
		disposerFn: func() {
			defer s.events.Disposed(r)
			func() {
				s.lock.Lock()
				defer s.lock.Unlock()
				s.seatsInUse -= seatsRequested
				s.currentRequestsInFlight -= 1
			}()
		},
	}, nil
}
