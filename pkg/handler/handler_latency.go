package handler

import (
	"net/http"

	apfcontext "github.com/tkashem/apf/pkg/context"
	"k8s.io/utils/clock"
)

func NewHandlerLatencyTracker(handler http.Handler) http.Handler {
	tracker := &tracker{
		clock:   clock.RealClock{},
		handler: handler,
	}
	return tracker.Build()
}

type tracker struct {
	clock   clock.Clock
	handler http.Handler
}

func (t *tracker) Build() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scoped, err := apfcontext.RequestScopedFrom(r.Context())
		if err != nil {
			t.handler.ServeHTTP(w, r)
			return
		}

		scoped.UserHandlerLatency = apfcontext.NewLatencyTracker(t.clock)
		defer scoped.UserHandlerLatency.Done()
		t.handler.ServeHTTP(w, r)
	})
}
