package context

import (
	"context"
	"fmt"
	"time"

	"github.com/tkashem/apf/pkg/fairqueuing/estimator"
	"github.com/tkashem/apf/pkg/fairqueuing/flow"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
	"github.com/tkashem/apf/pkg/scheduler"
)

type RequestScopedContextKeyType int

const RequestScopedContextKey RequestScopedContextKeyType = iota

var (
	reqScopedDataMissingErr = fmt.Errorf("no request scoped data found in the request context")
)

func RequestScopedFrom(ctx context.Context) (*RequestScoped, error) {
	if states, ok := ctx.Value(RequestScopedContextKey).(*RequestScoped); ok {
		return states, nil
	}
	return nil, reqScopedDataMissingErr
}

func WithRequestScopedContext(parent context.Context, scoped *RequestScoped) context.Context {
	return context.WithValue(parent, RequestScopedContextKey, scoped)
}

type RequestScoped struct {
	Served             bool
	UserHandlerLatency LatencyTracker
	TotalLatency       LatencyTracker

	Flow                      flow.RequestFlow
	Decision                  scheduler.DecisionSetter
	QueueWaitLatency          LatencyTracker
	PostDecisionExecutionWait LatencyTracker
	Estimate                  estimator.CostEstimate
	RTracker                  virtual.RTracker
}

type LatencyTracker interface {
	Done()
	Get() (startedAt time.Time, duration time.Duration)
}
