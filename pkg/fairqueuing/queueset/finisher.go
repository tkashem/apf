package queueset

import (
	"github.com/tkashem/apf/pkg/fairqueuing"
)

type queuedFinisher struct {
	waiter    fairqueuing.DecisionWaiter
	callbacks queueCleanupCallbacks

	finished bool
}

func (r *queuedFinisher) Finish(execute func()) {
	if r.finished {
		return
	}
	r.finished = true

	decision := r.waiter.WaitForDecision()
	switch decision {
	case fairqueuing.DecisionTimeout:
		// the request had timed out while in the queue
		r.callbacks.PostTimeout()

	case fairqueuing.DecisionExecute:
		// the request has been dequeued and scheduled for execution
		// execute the request handler
		func() {
			defer r.callbacks.PostExecution()
			execute()
		}()

	default:
		// impossible decision, log it
	}
}
