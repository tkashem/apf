package handler

import (
	"fmt"
	"net/http"

	apfcontext "github.com/tkashem/apf/pkg/context"
	"github.com/tkashem/apf/pkg/fairqueuing/estimator"
	"github.com/tkashem/apf/pkg/fairqueuing/flow"
	"github.com/tkashem/apf/pkg/scheduler"
	"k8s.io/utils/clock"
)

func NewExecutor(handler http.Handler, errorHandler ErrorHandler, estimator estimator.CostEstimator,
	factory scheduler.SchedulerFactory, events scheduler.Events,
	distinguisherFn flow.FlowDistinguisherFunc) http.Handler {
	executor := &executor{
		clock:           clock.RealClock{},
		estimator:       estimator,
		factory:         factory,
		errorHandler:    errorHandler,
		handler:         handler,
		events:          events,
		flow:            flow.ComputeFlow,
		distinguisherFn: distinguisherFn,
	}
	return executor.NewExecutorHandler()
}

type executor struct {
	clock           clock.Clock
	estimator       estimator.CostEstimator
	distinguisherFn flow.FlowDistinguisherFunc
	flow            flow.FlowComputerFunc
	factory         scheduler.SchedulerFactory
	errorHandler    ErrorHandler
	events          scheduler.Events
	// request handler
	handler http.Handler
}

func (e *executor) NewExecutorHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scoped, err := apfcontext.RequestScopedFrom(r.Context())
		if err != nil {
			// we should never be here
			e.errorHandler.HandleError(w, r, err)
			return
		}

		scoped.Flow = e.flow.ComputeFlow(r, e.distinguisherFn)

		vr, err := e.estimator.EstimateCost(r)
		if err != nil {
			e.errorHandler.HandleError(w, r, fmt.Errorf("error estimating request cost"))
			return
		}
		scoped.Estimate = vr

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
					scoped.PostDecisionExecutionWait.Done()
				}
				e.events.ExecutionStarting(r)
				defer func() {
					scoped.TotalLatency.Done()
					e.events.ExecutionEnded(r)
				}()

				e.handler.ServeHTTP(w, r)
			}()
		})
	})
}
