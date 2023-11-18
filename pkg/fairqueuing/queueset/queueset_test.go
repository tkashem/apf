package queueset

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/promise"
	"github.com/tkashem/apf/pkg/fairqueuing/queueselector"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"

	"k8s.io/utils/clock"
)

func Test(t *testing.T) {
	config := &Config{
		Clock: clock.RealClock{},
		QueuingConfig: &QueuingConfig{
			NQueues:        1,
			QueueMaxLength: 128,
		},
		TotalSeats:    1,
		Events:        events{t: t},
		QueueSelector: queueselector.NewRoundRobinQueueSelector(),
	}

	qs, err := NewQueueSet(config)
	if err != nil {
		t.Fatalf("failed to create queueset")
	}

	req := newRequest(1, 1, time.Second)
	finisher, _ := qs.Enqueue(req)
	if finisher == nil {
		t.Errorf("expected the request to be enqueued by the queueset")
		return
	}

	qs.Dispatch()

	var executed bool
	finisher.Finish(func() {
		executed = true
		t.Logf("request being executed")
	})
	if !executed {
		t.Errorf("expected the request to be executed")
	}
}

type request struct {
	id uint32
	virtual.RTracker
	seats    uint32
	duration time.Duration
	trackers fairqueuing.LatencyTrackers
	fairqueuing.DecisionWaiterSetter
}

func (r *request) GetFlowID() fairqueuing.FlowIDType {
	return fairqueuing.FlowIDType(0)
}
func (r *request) Context() context.Context {
	return context.Background()
}
func (r *request) EstimateCost() (seats uint32, width virtual.SeatSeconds) {
	return r.seats, virtual.SeatsTimesDuration(float64(r.seats), r.duration)
}
func (r *request) LatencyTrackers() fairqueuing.LatencyTrackers { return r.trackers }
func (r *request) String() string                               { return fmt.Sprintf("%d", r.id) }

func newRequest(id uint32, seats uint32, duration time.Duration) *request {
	return &request{
		id:                   id,
		seats:                seats,
		duration:             duration,
		RTracker:             virtual.NewRTracker(),
		DecisionWaiterSetter: promise.New(context.Background()),
		trackers: fairqueuing.LatencyTrackers{
			QueueWait:                 fakeLatencyTracker{},
			PostDecisionExecutionWait: fakeLatencyTracker{},
			ExecutionDuration:         fakeLatencyTracker{},
			TotalDuration:             fakeLatencyTracker{},
		},
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
	e.t.Logf("dequeued from: %q, request: %q", q, r)
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
