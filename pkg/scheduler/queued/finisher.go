package queued

import (
	"context"
	"net/http"
	"time"

	apfcontext "github.com/tkashem/apf/pkg/context"
	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/scheduler"
)

type QueueTimeoutContextFunc func(*http.Request) (context.Context, context.CancelFunc)

type queuedRequest struct {
	decision  scheduler.DecisionWaiter
	callbacks fairqueuing.QueueCleanupCallbacks

	finished bool
}

func (r *queuedRequest) Finish(execute func()) {
	if r.finished {
		return
	}
	r.finished = true

	decision := r.decision.WaitForDecision()
	switch decision {
	case scheduler.DecisionTimeout:
		// the request had timed out while in the queue
		r.callbacks.PostTimeout.Dispose()
	case scheduler.DecisionExecute:
		// the request has been dequeued and scheduled for execution
		// execute the request handler
		func() {
			defer r.callbacks.PostExecution.Dispose()
			execute()
		}()

	default:
		// impossible decision, log it
	}
}

func queueTimeoutContext(r *http.Request) (context.Context, context.CancelFunc) {
	// TODO: for now we say that a request can wait in queue for about 8s
	// if it does not have any deadline
	f := func() (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		return ctx, cancel
	}

	deadlineAt, ok := r.Context().Deadline()
	if !ok {
		return f()
	}
	scoped, err := apfcontext.RequestScopedFrom(r.Context())
	if err != nil {
		return f()
	}
	arrivedAt, _ := scoped.TotalLatency.Get()
	if arrivedAt.Nanosecond() == 0 {
		return f()
	}

	// the request has a deadline, and we know when it arrived
	deadline := deadlineAt.Sub(arrivedAt)
	ctx, cancel := context.WithDeadline(context.Background(), arrivedAt.Add(time.Duration(deadline/4)))
	return ctx, cancel
}
