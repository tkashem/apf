package factory

import (
	"fmt"
	"net/http"

	"github.com/tkashem/apf/pkg/core"
	"github.com/tkashem/apf/pkg/handler"
	"github.com/tkashem/apf/pkg/scheduler"
)

func NewBuilder() *builder {
	return &builder{}
}

type builder struct {
	handler      http.Handler
	seats        int32
	factory      core.SchedulerFactory
	estimator    core.WorkEstimator
	errorHandler core.ErrorHandler
	exemption    core.Exemption
	callback     core.ResponseCallback
	events       core.SchedulingEvents

	err error
}

func (b *builder) WithRequestHandler(h http.Handler) *builder {
	b.handler = handler.NewHandlerLatencyTracker(h)
	return b
}

func (b *builder) WithServerConcurrency(seats int32) *builder {
	b.seats = seats
	return b
}

func (b *builder) WithQueuing(queueMaxLength int, nQueues int) *builder {
	if b.events == nil {
		b.events = scheduler.SchedulingEvents{}
	}
	b.factory, b.err = scheduler.NewQueuedSchedulerFactory(b.seats, queueMaxLength, nQueues, b.events)
	return b
}

func (b *builder) Build() (http.Handler, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.handler == nil {
		return nil, fmt.Errorf("must supply a request http Handler")
	}
	if b.errorHandler == nil {
		b.errorHandler = core.NewDefaultErrorHandler()
	}
	if b.callback == nil {
		b.callback = core.NewDefaultResponseCallback()
	}
	if b.exemption == nil {
		b.exemption = core.NewNoExemption()
	}
	if b.estimator == nil {
		b.estimator = core.NewUniformWorkEstimator()
	}
	if b.events == nil {
		b.events = scheduler.SchedulingEvents{}
	}
	if b.factory == nil {
		factory, err := scheduler.NewNoWaitSchedulerFactory(b.seats, b.events)
		if err != nil {
			return nil, err
		}
		b.factory = factory
	}

	h := handler.NewExecutor(b.handler, b.errorHandler, b.estimator, b.factory, b.events)
	h = handler.NewStarter(h, b.errorHandler, b.exemption, b.callback, b.events)
	return h, nil
}
