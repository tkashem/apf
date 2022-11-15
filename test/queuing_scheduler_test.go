package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tkashem/apf/pkg/factory"
)

func TestSchedulerWithQueuing(t *testing.T) {
	var secondExecuted bool
	firstInProgressCh, firstDoneCh, firstBlockedCh := make(chan struct{}), make(chan struct{}), make(chan struct{})
	requestHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("path=%q", r.URL.Path)
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
		WithServerConcurrency(1).
		WithQueuing(128, 16).
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

	secondResult := <-secondResultCh
	if err := check(secondResult.response, secondResult.err, http.StatusOK); err != nil {
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
}
