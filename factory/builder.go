package factory

import (
	"fmt"
	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/queueset"
	"github.com/tkashem/apf/pkg/scheduler/queued"
	"net/http"

	apfcontext "github.com/tkashem/apf/pkg/context"
	"github.com/tkashem/apf/pkg/defaults"
	fairqueuingestimator "github.com/tkashem/apf/pkg/fairqueuing/estimator"
	"github.com/tkashem/apf/pkg/fairqueuing/flow"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
	"github.com/tkashem/apf/pkg/handler"
	"github.com/tkashem/apf/pkg/scheduler"
	nowaitscheduler "github.com/tkashem/apf/pkg/scheduler/nowait"
)

func NewBuilder() *builder {
	return &builder{}
}

type builder struct {
	err             error
	handler         http.Handler
	seats           uint32
	factory         scheduler.SchedulerFactory
	estimator       fairqueuingestimator.CostEstimator
	errorHandler    handler.ErrorHandler
	exemption       handler.Exemption
	callback        handler.ResponseCallback
	events          scheduler.Events
	scopedFactory   func() *apfcontext.RequestScoped
	distinguisherFn flow.FlowDistinguisherFunc
	queueset        fairqueuing.FairQueueSet
}

func (b *builder) WithRequestHandler(h http.Handler) *builder {
	b.handler = handler.NewHandlerLatencyTracker(h)
	return b
}

func (b *builder) WithServerConcurrency(seats uint32) *builder {
	b.seats = seats
	return b
}

func (b *builder) WithQueuing(config queueset.Config) *builder {
	b.scopedFactory = func() *apfcontext.RequestScoped {
		return &apfcontext.RequestScoped{
			RTracker: virtual.NewRTracker(),
		}
	}

	completed := &queueset.CompletedConfig{
		Config:     config,
		TotalSeats: b.seats,
		Events:     b.events,
	}
	var qs fairqueuing.FairQueueSet
	qs, b.err = queueset.NewQueueSet(completed)
	if b.err != nil {
		return b
	}
	b.factory, b.err = queued.NewQueuedSchedulerFactory(qs)
	return b
}

func (b *builder) WithSchedulingEvents(events scheduler.Events) *builder {
	b.events = events
	return b
}

func (b *builder) WithUserProvidedCostEstimator(fn fairqueuingestimator.UserProvidedCostEstimatorFunc) *builder {
	var estimator fairqueuingestimator.CostEstimator
	estimator, b.err = fairqueuingestimator.NewCostEstimator(fn)

	b.estimator = estimator
	return b
}

func (b *builder) WithFlowDistinguisher(distinguisherFn flow.FlowDistinguisherFunc) *builder {
	b.distinguisherFn = distinguisherFn
	return b
}

func (b *builder) Build() (http.Handler, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.handler == nil {
		return nil, fmt.Errorf("must supply a request http Handler")
	}
	if b.distinguisherFn == nil {
		return nil, fmt.Errorf("request flow distinguisher must be specified")
	}

	if b.errorHandler == nil {
		b.errorHandler = defaults.NewDefaultErrorHandler()
	}
	if b.callback == nil {
		b.callback = defaults.NewDefaultResponseCallback()
	}
	if b.exemption == nil {
		b.exemption = defaults.NewNoExemption()
	}
	if b.events == nil {
		b.events = scheduler.NoEvents()
	}
	if b.scopedFactory == nil {
		b.scopedFactory = func() *apfcontext.RequestScoped {
			return &apfcontext.RequestScoped{}
		}
	}
	if b.factory == nil {
		factory, err := nowaitscheduler.NewNoWaitSchedulerFactory(b.seats, b.events)
		if err != nil {
			return nil, err
		}
		b.factory = factory
	}
	if b.estimator == nil {
		b.estimator = defaults.NewCostEstimator()
	}

	h := handler.NewExecutor(b.handler, b.errorHandler, b.estimator, b.factory, b.events, b.distinguisherFn)
	h = handler.NewStarter(h, b.errorHandler, b.exemption, b.callback, b.events, b.scopedFactory)
	return h, nil
}
