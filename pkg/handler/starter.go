package handler

import (
	"net/http"

	apfcontext "github.com/tkashem/apf/pkg/context"
	"github.com/tkashem/apf/pkg/scheduler"
	"k8s.io/utils/clock"
)

func NewStarter(executor http.Handler, errorHandler ErrorHandler, exemption Exemption,
	callback ResponseCallback, events scheduler.Events,
	factory func() *apfcontext.RequestScoped) http.Handler {
	starter := &starter{
		clock:        clock.RealClock{},
		executor:     executor,
		errorHandler: errorHandler,
		exemption:    exemption,
		callback:     callback,
		events:       events,
		factory:      factory,
	}
	return starter.Build()
}

type starter struct {
	clock        clock.Clock
	exemption    Exemption
	errorHandler ErrorHandler
	callback     ResponseCallback
	executor     http.Handler
	events       scheduler.Events
	factory      func() *apfcontext.RequestScoped
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

		scoped := s.factory()
		scoped.TotalLatency = apfcontext.NewLatencyTracker(s.clock)
		r = r.WithContext(apfcontext.WithRequestScopedContext(r.Context(), scoped))

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
