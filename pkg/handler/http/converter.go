package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/promise"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
	"github.com/tkashem/apf/pkg/latencytracker"
	"k8s.io/utils/clock"
)

type FlowGetterFunc func(*http.Request) (fairqueuing.FlowIDType, error)
type CostEstimatorFunc func(*http.Request) (seats uint32, duration time.Duration, err error)
type QueueWaitContextFunc func(*http.Request) (context.Context, context.CancelFunc)

type converter struct {
	clock            clock.PassiveClock
	flowGetter       FlowGetterFunc
	costEstimator    CostEstimatorFunc
	queueWaitContext QueueWaitContextFunc
}

func NewConverter(clock clock.PassiveClock, queueWaitContext QueueWaitContextFunc, flowGetter FlowGetterFunc, costEstimator CostEstimatorFunc) *converter {
	return &converter{clock: clock, queueWaitContext: queueWaitContext, flowGetter: flowGetter, costEstimator: costEstimator}
}

func (c converter) Convert(in *http.Request) (fairqueuing.Request, error) {
	flowID, err := c.flowGetter(in)
	if err != nil {
		return nil, err
	}

	seats, duration, err := c.costEstimator(in)
	if err != nil {
		return nil, err
	}

	r := &request{
		req:      in,
		flowID:   flowID,
		seats:    seats,
		duration: duration,
		RTracker: virtual.NewRTracker(),
		trackers: fairqueuing.LatencyTrackers{
			QueueWait:                 latencytracker.NewLatencyTracker(c.clock),
			PostDecisionExecutionWait: latencytracker.NewLatencyTracker(c.clock),
			ExecutionDuration:         latencytracker.NewLatencyTracker(c.clock),
			TotalDuration:             latencytracker.NewLatencyTracker(c.clock),
		},
	}

	ctx := in.Context()
	if c.queueWaitContext != nil {
		var cancel context.CancelFunc
		ctx, cancel = c.queueWaitContext(in)
		r.ctx = ctx
		r.cancel = cancel
	}
	r.DecisionWaiterSetter = promise.New(ctx)

	return r, nil
}

type request struct {
	fairqueuing.DecisionWaiterSetter
	virtual.RTracker

	ctx    context.Context
	cancel context.CancelFunc

	req *http.Request

	seats    uint32
	duration time.Duration
	flowID   fairqueuing.FlowIDType
	trackers fairqueuing.LatencyTrackers
}

func (r *request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return r.req.Context()
}
func (r *request) CancelFunc() context.CancelFunc    { return r.cancel }
func (r *request) GetFlowID() fairqueuing.FlowIDType { return r.flowID }
func (r *request) EstimateCost() (seats uint32, width virtual.SeatSeconds) {
	return r.seats, virtual.SeatsTimesDuration(float64(r.seats), r.duration)
}
func (r *request) LatencyTrackers() fairqueuing.LatencyTrackers { return r.trackers }
func (r *request) String() string                               { return fmt.Sprintf("%q", r.req.URL) }
