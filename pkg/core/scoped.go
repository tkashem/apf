package core

import (
	"context"
	"fmt"
	"time"

	"k8s.io/utils/clock"
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
	Served   bool
	Decision DecisionSetter
	Estimate *WorkEstimate

	UserHandlerLatency        LatencyTracker
	TotalLatency              LatencyTracker
	QueueWaitLatency          LatencyTracker
	PostDecisionExecutionWait LatencyTracker
}

type LatencyTracker interface {
	TookSince(clock clock.PassiveClock)
	Get() (startedAt time.Time, duration time.Duration)
}

type latencyTracker struct {
	startedAt time.Time
	duration  time.Duration
}

func (t latencyTracker) TookSince(clock clock.PassiveClock) {
	t.duration = clock.Since(t.startedAt)
}

func (t latencyTracker) Get() (time.Time, time.Duration) {
	return t.startedAt, t.duration
}

func NewLatencyTracker(clock clock.PassiveClock) LatencyTracker {
	return latencyTracker{
		startedAt: clock.Now(),
	}
}
