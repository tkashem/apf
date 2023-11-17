package queueset

import (
	"github.com/tkashem/apf/pkg/fairqueuing"
)

type queuedFinisher struct {
	waiter                     fairqueuing.DecisionWaiter
	postTimeout, postExecution disposer

	finished bool
}

func (r *queuedFinisher) Finish(fn func()) {
	if r.finished {
		return
	}
	r.finished = true

	decision := r.waiter.WaitForDecision()
	switch decision {
	case fairqueuing.DecisionTimeout:
		// the request had timed out while in the queue
		r.postTimeout.Dispose()

	case fairqueuing.DecisionExecute:
		// the request has been dequeued and scheduled for execution
		// execute the request handler
		func() {
			defer r.postExecution.Dispose()
			fn()
		}()

	default:
		// impossible decision, log it
	}
}
