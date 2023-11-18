package queueset

import (
	"github.com/tkashem/apf/pkg/fairqueuing"
)

type queuedFinisher struct {
	request                    fairqueuing.Request
	postTimeout, postExecution disposer
	finished                   bool
}

func (r *queuedFinisher) Finish(fn func()) {
	if r.finished {
		return
	}
	r.finished = true
	trackers := r.request.LatencyTrackers()

	decision := r.request.WaitForDecision()
	switch decision {
	case fairqueuing.DecisionTimeout:
		// the request had timed out while in the queue
		func() {
			defer trackers.TotalDuration.Finish()
			r.postTimeout.Dispose()
		}()

	case fairqueuing.DecisionExecute:
		// the request has been dequeued and scheduled for execution
		// execute the request handler
		func() {
			defer trackers.TotalDuration.Finish()

			trackers.PostDecisionExecutionWait.Finish()
			func() {
				defer r.postExecution.Dispose()
				func() {
					trackers.ExecutionDuration.Start()
					defer trackers.ExecutionDuration.Finish()
					fn()
				}()
			}()

		}()

	default:
		// impossible decision, log it
	}
}
