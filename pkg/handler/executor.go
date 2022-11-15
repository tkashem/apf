package handler

import (
	"fmt"
	"net/http"

	"github.com/tkashem/apf/pkg/core"
	"k8s.io/utils/clock"
)

func NewExecutor(handler http.Handler, errorHandler core.ErrorHandler, estimator core.WorkEstimator,
	factory core.SchedulerFactory, events core.SchedulingEvents) http.Handler {
	executor := &executor{
		clock:        clock.RealClock{},
		estimator:    estimator,
		factory:      factory,
		errorHandler: errorHandler,
		handler:      handler,
		events:       events,
	}
	return executor.Build()
}

type executor struct {
	clock        clock.Clock
	estimator    core.WorkEstimator
	factory      core.SchedulerFactory
	errorHandler core.ErrorHandler
	events       core.SchedulingEvents
	// request handler
	handler http.Handler
}

func (e *executor) Build() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scoped, err := core.RequestScopedFrom(r.Context())
		if err != nil {
			// we should never be here
			e.errorHandler.HandleError(w, r, err)
			return
		}

		var estimate *core.WorkEstimate
		if e.estimator != nil {
			est, err := e.estimator.Estimate(r)
			if err != nil {
				e.errorHandler.HandleError(w, r, fmt.Errorf("error estimating request cost"))
				return
			}
			estimate = &est
		}
		if estimate == nil {
			estimate = &core.WorkEstimate{
				Seats: 1,
			}
		}
		scoped.Estimate = estimate

		scheduler := e.factory.GetScheduler(r)
		finisher, err := scheduler.Schedule(r)
		if err != nil {
			e.errorHandler.HandleError(w, r, fmt.Errorf("error scheduling request"))
			return
		}

		finisher.Finish(func() {
			scoped.Served = true
			func() {
				// track how much time the request had to wait between when a
				// decision had been made to execute the request, and the
				// user handler actually being executed.
				if scoped.PostDecisionExecutionWait != nil {
					scoped.PostDecisionExecutionWait.TookSince(e.clock)
				}
				e.events.ExecutionStarting(r)
				defer func() {
					scoped.TotalLatency.TookSince(e.clock)
					e.events.ExecutionEnded(r)
				}()

				e.handler.ServeHTTP(w, r)
			}()
		})
	})
}
