package fairqueuing

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/tkashem/apf/pkg/core"
	"k8s.io/utils/clock"
)

var (
	accommodationErr = fmt.Errorf("cannot accommodate request")
	queueEmptyErr    = fmt.Errorf("selected queue should not be empty")
	enqueueErr       = fmt.Errorf("failed to enqueue the request")
)

func NewQueueset(totalSeats int32, queueMaxLength int, nQueues int, events core.SchedulingEvents) *queueset {
	queues := make([]core.FairQueue, nQueues)
	for i := range queues {
		queues[i] = &fairQueue{
			queue: newRequestFIFO(),
		}
	}
	return &queueset{
		totalSeats:     totalSeats,
		queueMaxLength: queueMaxLength,
		queues:         queues,
		events:         events,
	}
}

type Counter struct {
	Executing int32
	Waiting   int32
}

type queueset struct {
	lock       sync.Mutex
	totalSeats int32

	// requests is the count in the real world.
	requests Counter

	// seat(s) are the total number of "seats" currently occupied
	// by all the requests that are currently executing in this queue,
	// or waiting to be executed
	seats Counter

	queueMaxLength int
	queues         []core.FairQueue
	robinIndex     int
	events         core.SchedulingEvents
	clock          clock.Clock
}

type queues []core.FairQueue

func (q queues) Total() int {
	return len(q)
}

func (q queues) Get(idx int) core.FairQueue {
	return q[idx]
}

func (qs *queueset) Name() string {
	return ""
}

func (qs *queueset) Enqueue(r *http.Request, assigner core.QueueAssigner) (core.PostDequeueCallbacks, error) {
	scoped, err := core.RequestScopedFrom(r.Context())
	if err != nil {
		return core.PostDequeueCallbacks{}, err
	}

	qs.lock.Lock()
	defer qs.lock.Unlock()

	queue := assigner.Assign(queues(qs.queues))
	if queue == nil {
		return core.PostDequeueCallbacks{}, fmt.Errorf("assigned queue can not be nil")
	}
	qs.events.QueueAssigned(r)

	if qs.seats.Executing >= qs.totalSeats && queue.Length() >= qs.queueMaxLength {
		return core.PostDequeueCallbacks{}, accommodationErr
	}

	queueRemoveFn, err := queue.Enqueue(r)
	if err != nil {
		return core.PostDequeueCallbacks{}, enqueueErr
	}

	qs.seats.Waiting += scoped.Estimate.Seats
	qs.requests.Waiting += 1
	scoped.QueueWaitLatency = core.NewLatencyTracker(qs.clock)
	qs.events.Enqueued(r)

	disposers := core.PostDequeueCallbacks{
		// if a request has been executed, that means the Dispatch method
		// had already dequeued, and scheduled it for execution, in this
		// case we no longer need to remove it from the queue.
		PostExecution: core.DisposerFunc(func() {
			defer qs.events.Disposed(r)
			qs.finish(r)
		}),
		PostTimeout: core.DisposerFunc(func() {
			// if a request has timed out while waiting to be executed, that
			// means the Dispatch method had not had a successful attempt to
			// schedule it for execution, and thus it remains in the queue, so
			// we should remove it from its queue.
			defer qs.events.Timeout(r)

			qs.timeout(r, queueRemoveFn)
		}),
	}

	// cleanup after execution
	return disposers, nil
}

func (qs *queueset) Dispatch() (bool, error) {
	qs.lock.Lock()
	defer qs.lock.Unlock()

	var minQueue core.FairQueue
	var minIndex int
	minVirtualFinish := core.MaxSeatSeconds
	for range qs.queues {
		qs.robinIndex = (qs.robinIndex + 1) % len(qs.queues)
		queue := qs.queues[qs.robinIndex]
		if queue.Length() == 0 {
			continue
		}
		thisVirtualFinish := queue.GetVirtualFinish()
		if thisVirtualFinish < minVirtualFinish {
			minVirtualFinish = thisVirtualFinish
			minQueue = queue
		}
	}

	if minQueue != nil {
		// we set the round robin indexing to start at the chose queue
		// for the next round.  This way the non-selected queues
		// win in the case that the virtual finish times are the same
		qs.robinIndex = minIndex
	}

	// can we accommodate this request?
	request, ok := minQueue.Peek()
	if !ok {
		return false, queueEmptyErr
	}
	scoped, err := core.RequestScopedFrom(request.Context())
	if err != nil {
		return false, err
	}

	seats := scoped.Estimate.Seats
	if qs.seats.Executing+seats > qs.totalSeats {
		return false, accommodationErr
	}

	// remove the request from queue for execution
	request, ok = minQueue.DequeueForExecution(func() {
		defer qs.events.Dequeued(request)

		qs.requests.Waiting -= 1
		qs.seats.Waiting -= seats

		// There are two ways a request is dequeued:
		//  a) scheduler dequeues it and schedules it for execution
		//  b) the request times out while waiting in the queue, and it
		//     is being removed from the queue before being rejected.
		// we are here for a, and we want to track how much the request
		// spent inside of the queue waiting.
		scoped.QueueWaitLatency.TookSince(qs.clock)
	}, func() {
		defer qs.events.DecisionChanged(request, core.DecisionExecute)

		qs.requests.Executing += 1
		qs.seats.Executing += seats

		// we have just made a decision to execute the request, we want
		// to track the latency from here until the user handler starts
		// executing.
		scoped.PostDecisionExecutionWait = core.NewLatencyTracker(qs.clock)
	})
	if !ok {
		return false, queueEmptyErr
	}

	return true, nil
}

func (qs *queueset) finish(r *http.Request) {
	scoped, err := core.RequestScopedFrom(r.Context())
	if err != nil {
		// impossible case, log error
		return
	}

	qs.lock.Lock()
	defer qs.lock.Unlock()

	qs.seats.Executing -= scoped.Estimate.Seats
	qs.requests.Executing -= 1
}

func (qs *queueset) timeout(r *http.Request, queueDisposer core.DisposerFunc) {
	scoped, err := core.RequestScopedFrom(r.Context())
	if err != nil {
		// impossible case, log error
		return
	}

	func() {
		qs.lock.Lock()
		defer qs.lock.Unlock()

		queueDisposer()
		qs.seats.Waiting -= scoped.Estimate.Seats
		qs.requests.Waiting -= 1
	}()

	// There are two ways a request is dequeued:
	//  a) scheduler dequeues it and schedules it for execution
	//  b) the request times out while waiting in the queue, and it
	//     is being removed from the queue before being rejected.
	//
	// we are here for b, and we want to track how much the request
	// spent inside of the queue waiting
	scoped.QueueWaitLatency.TookSince(qs.clock)
}
