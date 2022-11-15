package scheduler

import (
	"github.com/tkashem/apf/pkg/core"
)

type immediateRequest struct {
	disposer core.Disposer
}

func (r immediateRequest) Finish(execute func()) {
	defer r.disposer.Dispose()
	execute()
}

type rejectedRequest struct{}

func (rejectedRequest) Finish(_ func()) {}

type queuedRequest struct {
	decision  core.DecisionGetter
	callbacks core.PostDequeueCallbacks
	finished  bool
}

func (r *queuedRequest) Finish(execute func()) {
	if r.finished {
		return
	}
	r.finished = true

	decision := r.decision.Get()
	switch decision {
	case core.DecisionTimeout:
		// the request had timed out while in the queue
		r.callbacks.PostTimeout.Dispose()
	case core.DecisionExecute:
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
