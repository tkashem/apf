package queueset

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/queueselector"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"

	"k8s.io/utils/clock"
)

func Test(t *testing.T) {
	config := &Config{
		Clock: clock.RealClock{},
		QueuingConfig: &QueuingConfig{
			NQueues:        3,
			QueueMaxLength: 128,
		},
		TotalSeats:    1,
		Events:        events{t: t},
		QueueSelector: queueselector.NewRoundRobinQueueSelector(),
	}

	_, err := NewQueueSet(config)
	if err != nil {
		t.Fatalf("failed to create queueset")
	}
}

type request struct {
	id uint32
	virtual.RTracker
	seats    uint32
	duration time.Duration
	lt       fairqueuing.LatencyTracker
}

func (r *request) GetFlowID() fairqueuing.FlowIDType {
	return fairqueuing.FlowIDType(0)
}
func (r *request) GetContext() context.Context {
	return context.Background()
}
func (r *request) EstimateCost() (seats uint32, width virtual.SeatSeconds) {
	return r.seats, virtual.SeatsTimesDuration(float64(r.seats), r.duration)
}
func (r *request) QueueWaitLatencyTracker() fairqueuing.LatencyTracker                 { return r.lt }
func (r *request) PostDecisionExecutionWaitLatencyTracker() fairqueuing.LatencyTracker { return r.lt }
func (r *request) String() string                                                      { return fmt.Sprintf("%d", r.id) }

func newRequest() *request {
	return &request{
		RTracker: virtual.NewRTracker(),
	}
}

type events struct {
	t *testing.T
}

func (e events) QueueSelected(q fairqueuing.FairQueue, r fairqueuing.Request) {
	e.t.Logf("selected queue: %q, for request: %q", q, r)
}
func (e events) Enqueued(q fairqueuing.FairQueue, r fairqueuing.Request) {
	e.t.Logf("enqueued at: %q, request: %q", q, r)
}
func (e events) Dequeued(q fairqueuing.FairQueue, r fairqueuing.Request) {
	e.t.Logf("aequeued from: %q, request: %q", q, r)
}
func (e events) DecisionChanged(r fairqueuing.Request, d fairqueuing.DecisionType) {
	e.t.Logf("decision changed for request: %q, decision: %d", r, d)
}
func (e events) Disposed(r fairqueuing.Request) {
	e.t.Logf("disposed: %q", r)
}
func (e events) Timeout(r fairqueuing.Request) {
	e.t.Logf("timeout: %q", r)
}

type fakeLatencyTracker struct{}

func (f fakeLatencyTracker) Start()  {}
func (f fakeLatencyTracker) Finish() {}
func (f fakeLatencyTracker) GetDuration() (startedAt time.Time, duration time.Duration) {
	return time.Time{}, 0
}
