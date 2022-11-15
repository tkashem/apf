package handler

import (
	"net/http"

	"github.com/tkashem/apf/pkg/core"
	"k8s.io/utils/clock"
)

func NewStarter(executor http.Handler, errorHandler core.ErrorHandler, exemption core.Exemption,
	callback core.ResponseCallback, events core.SchedulingEvents) http.Handler {
	starter := &starter{
		clock:        clock.RealClock{},
		executor:     executor,
		errorHandler: errorHandler,
		exemption:    exemption,
		callback:     callback,
		events:       events,
	}
	return starter.Build()
}

type starter struct {
	clock        clock.Clock
	exemption    core.Exemption
	errorHandler core.ErrorHandler
	callback     core.ResponseCallback
	executor     http.Handler
	events       core.SchedulingEvents
}

func (s *starter) Build() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exempt, err := s.exemption.IsExempt(r)
		if err != nil {
			s.errorHandler.HandleError(w, r, err)
			return
		}
		if exempt {
			s.executor.ServeHTTP(w, r)
			s.callback.OnExempt(w, r)
			return
		}

		scoped := &core.RequestScoped{}
		scoped.TotalLatency = core.NewLatencyTracker(s.clock)
		r = r.WithContext(core.WithRequestScopedContext(r.Context(), scoped))

		s.events.Arrived(r)

		defer func() {
			if !scoped.Served {
				s.callback.OnRejected(w, r)
				return
			}
			s.callback.OnServed(w, r)
		}()
		s.executor.ServeHTTP(w, r)
	})
}
