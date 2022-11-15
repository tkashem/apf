package handler

import (
	"net/http"

	"github.com/tkashem/apf/pkg/core"
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
		scoped, err := core.RequestScopedFrom(r.Context())
		if err != nil {
			t.handler.ServeHTTP(w, r)
			return
		}

		scoped.UserHandlerLatency = core.NewLatencyTracker(t.clock)
		defer scoped.UserHandlerLatency.TookSince(t.clock)
		t.handler.ServeHTTP(w, r)
	})
}
