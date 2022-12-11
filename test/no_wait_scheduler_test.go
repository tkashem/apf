package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tkashem/apf/factory"
)

func TestNoWaitScheduler(t *testing.T) {
	var secondExecuted, thirdExecuted bool
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
		case r.URL.Path == "/third":
			thirdExecuted = true
		}
	})

	handler, err := factory.NewBuilder().
		WithRequestHandler(requestHandler).
		WithServerConcurrency(1).
		WithSchedulingEvents(events{t: t}).
		WithFlowDistinguisher(func(r *http.Request) []string {
			return []string{r.URL.Path}
		}).
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

	// fire a third request, it should succeed
	func() {
		thirdURL := server.URL + "/third"
		t.Logf("sending third request: %q", thirdURL)
		resp, err := client.Get(thirdURL)
		if err := check(resp, err, http.StatusOK); err != nil {
			t.Errorf("[third]: %s", err.Error())
		}
	}()
	if !thirdExecuted {
		t.Errorf("expected third request to be executed")
	}
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
