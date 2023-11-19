package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/queueselector"
	"github.com/tkashem/apf/pkg/fairqueuing/queueset"
	"k8s.io/utils/clock"
)

func TestSchedulerWithQueuing(t *testing.T) {
	clock := clock.RealClock{}
	config := &queueset.Config{
		Clock: clock,
		QueuingConfig: &queueset.QueuingConfig{
			NQueues:        64,
			HandSize:       4,
			QueueMaxLength: 128,
		},
		TotalSeats:    1,
		Events:        queuingEvents{t: t},
		QueueSelector: queueselector.NewRoundRobinQueueSelector(),
	}

	qs, err := queueset.NewQueueSet(config)
	if err != nil {
		t.Fatalf("failed to create queueset")
	}

	converter := NewConverter(clock, func(r *http.Request) (context.Context, context.CancelFunc) {
		return context.WithTimeout(r.Context(), 3*time.Second)
	}, func(*http.Request) (fairqueuing.FlowIDType, error) {
		return 0, nil
	}, func(*http.Request) (seats uint32, duration time.Duration, err error) {
		return 1, time.Second, nil
	})

	var secondExecuted bool
	firstInProgressCh, firstDoneCh, firstBlockedCh := make(chan struct{}), make(chan struct{}), make(chan struct{})
	requestHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/first":
			defer close(firstDoneCh)
			close(firstInProgressCh)
			<-firstBlockedCh
		case r.URL.Path == "/second":
			secondExecuted = true
		}
	})

	handler := NewAPFHandler(requestHandler, qs, &Config{
		Exempt:       NewNoExemption(),
		ErrorHandler: NewDefaultErrorHandler(),
		Events:       NewDefaultEvents(),
		Clock:        clock,
		Converter:    converter,
	})
	server := httptest.NewUnstartedServer(handler)
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	client := server.Client()
	client.Timeout = 0

	type result struct {
		err      error
		response *http.Response
	}
	firstResultCh, secondResultCh := make(chan result, 1), make(chan result, 1)
	go func() {
		firstURL := server.URL + "/first"
		t.Logf("sending first request: %q", firstURL)
		resp, err := client.Get(firstURL)
		firstResultCh <- result{response: resp, err: err}
	}()
	go func() {
		<-firstInProgressCh
		secondURL := server.URL + "/second"
		t.Logf("sending second request: %q", secondURL)
		resp, err := client.Get(secondURL)
		secondResultCh <- result{response: resp, err: err}
	}()

	// the first request is in progress, we expect the second request to
	// be waiting in queue and then timeout
	secondResult := <-secondResultCh
	if err := check(secondResult.response, secondResult.err, http.StatusTooManyRequests); err != nil {
		t.Errorf("[second]: %s", err.Error())
	}
	if secondExecuted {
		t.Errorf("expected second request to be rejected")
	}

	close(firstBlockedCh)
	firstResult := <-firstResultCh
	if err := check(firstResult.response, firstResult.err, http.StatusOK); err != nil {
		t.Errorf("[first]: %s", err.Error())
	}

	// send the second request now, it should succeed
	secondURL := server.URL + "/second"
	t.Logf("retrying second request: %q", secondURL)
	resp, err := client.Get(secondURL)
	if err := check(resp, err, http.StatusOK); err != nil {
		t.Errorf("[second]: %s", err.Error())
	}
	if !secondExecuted {
		t.Errorf("expected second request to be served")
	}
}

type queuingEvents struct {
	t *testing.T
}

func (e queuingEvents) QueueSelected(q fairqueuing.FairQueue, r fairqueuing.Request) {
	e.t.Logf("selected queue: %q, for request: %q", q, r)
}
func (e queuingEvents) Enqueued(q fairqueuing.FairQueue, r fairqueuing.Request) {
	e.t.Logf("enqueued at: %q, request: %q", q, r)
}
func (e queuingEvents) Dequeued(q fairqueuing.FairQueue, r fairqueuing.Request) {
	e.t.Logf("dequeued from: %q, request: %q", q, r)
}
func (e queuingEvents) DecisionChanged(r fairqueuing.Request, d fairqueuing.DecisionType) {
	e.t.Logf("decision changed for request: %q, decision: %d", r, d)
}
func (e queuingEvents) Disposed(r fairqueuing.Request) {
	e.t.Logf("disposed: %q", r)
}
func (e queuingEvents) Timeout(r fairqueuing.Request) {
	e.t.Logf("timeout: %q", r)
}

func check(resp *http.Response, err error, want int) error {
	switch {
	case err != nil:
		return fmt.Errorf("expected no error, but got: %v", err)
	default:
		if resp.StatusCode != want {
			return fmt.Errorf("expected status code: %d, but got: %v", want, resp)
		}
	}
	return nil
}
