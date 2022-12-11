package test

import (
	"github.com/tkashem/apf/pkg/fairqueuing/queueassigner"
	"github.com/tkashem/apf/pkg/fairqueuing/queueset"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tkashem/apf/factory"
	apfcontext "github.com/tkashem/apf/pkg/context"
	"github.com/tkashem/apf/pkg/scheduler"
)

func TestSchedulerWithQueuing(t *testing.T) {
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

	handler, err := factory.NewBuilder().
		WithRequestHandler(requestHandler).
		WithUserProvidedCostEstimator(func(request *http.Request) (uint32, time.Duration) {
			return 1, 1 * time.Second
		}).
		WithFlowDistinguisher(func(r *http.Request) []string {
			return []string{r.URL.Path}
		}).
		WithServerConcurrency(1).
		WithSchedulingEvents(events{t: t}).
		WithQueuing(queueset.Config{NQueues: 16, QueueMaxLength: 128, QueueAssignerFactory: &queueassigner.RoundRobinQueueAssignerFactory{}}).
		Build()
	if err != nil {
		t.Fatalf("failed to build apf handler: %v", err)
	}

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

type events struct {
	t *testing.T
}

func (e events) Arrived(r *http.Request) {
	e.t.Logf("Arrived: %q", r.URL.Path)
}
func (e events) QueueAssigned(r *http.Request) {
	e.t.Logf("QueueAssigned: %q", r.URL.Path)
}
func (e events) Enqueued(r *http.Request) {
	scoped, _ := apfcontext.RequestScopedFrom(r.Context())
	e.t.Logf("Enqueued: %q, flow: %d", r.URL.Path, scoped.Flow)
}
func (e events) Dequeued(r *http.Request) {
	e.t.Logf("Dequeued: %q", r.URL.Path)
}
func (e events) DecisionChanged(r *http.Request, d scheduler.DecisionType) {
	e.t.Logf("DecisionChanged: %q decision: %d", r.URL.Path, d)
}
func (e events) ExecutionStarting(r *http.Request) {
	e.t.Logf("ExecutionStarting: %q ", r.URL.Path)
}
func (e events) ExecutionEnded(r *http.Request) {
	e.t.Logf("ExecutionEnded: %q ", r.URL.Path)
}
func (e events) Disposed(r *http.Request) {
	e.t.Logf("Disposed: %q ", r.URL.Path)
}
func (e events) Timeout(r *http.Request) {
	e.t.Logf("Timeout: %q ", r.URL.Path)
}
