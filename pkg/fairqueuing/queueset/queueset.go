package queueset

import (
	"fmt"
	"math"
	"sync"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
	"k8s.io/utils/clock"
)

var (
	accommodationErr = fmt.Errorf("cannot accommodate request")
	queueEmptyErr    = fmt.Errorf("selected queue should not be empty")
	enqueueErr       = fmt.Errorf("failed to enqueue the request")
)

func NewQueueSet(config *Config) (*queueset, error) {
	if config.TotalSeats < 1 {
		return nil, fmt.Errorf("seats must be positive")
	}

	qs := &queueset{clock: config.Clock}
	vclock := virtual.NewRTClock(qs.clock, qs.getWorkLocked)
	qs.vclock = vclock

	queues := make([]fairqueue, config.QueuingConfig.NQueues)
	for i := range queues {
		queues[i] = &fairQueue{
			id:     uint32(i + 1),
			fifo:   NewFIFO(),
			vclock: vclock,
		}
	}
	qs.queues = queues
	qs.assigner = config.QueueSelector

	qs.totalSeats = config.TotalSeats
	qs.queueMaxLength = config.QueuingConfig.QueueMaxLength
	qs.events = config.Events
	return qs, nil
}

type Counter struct {
	Executing uint32
	Waiting   uint32
}

type queueset struct {
	lock       sync.Mutex
	totalSeats uint32

	// requests is the count in the real world.
	requests fairqueuing.RequestCount

	// seat(s) are the total number of "seats" currently occupied
	// by all the requests that are currently executing in this queue,
	// or waiting to be executed
	seats fairqueuing.SeatCount

	queueMaxLength int
	queues         []fairqueue
	robinIndex     int
	events         Events
	clock          clock.Clock
	vclock         virtual.RTClock
	assigner       fairqueuing.QueueSelector
}

func (qs *queueset) Name() string {
	return ""
}

func (qs *queueset) TotalQueues() int {
	return len(qs.queues)
}

func (qs *queueset) GetFairQueue(idx int) fairqueuing.FairQueue {
	return qs.queues[idx]
}

func (qs *queueset) Enqueue(r fairqueuing.Request) (*queuedFinisher, error) {
	trackers := r.LatencyTrackers()
	trackers.TotalDuration.Start()

	qs.lock.Lock()
	defer qs.lock.Unlock()

	selected, err := qs.assigner.SelectQueue(qs, r.GetFlowID())
	if err != nil {
		return nil, fmt.Errorf("error assigning queue - %v", err)
	}

	queue := selected.(fairqueue)
	qs.events.QueueSelected(queue, r)

	// can we fit the request?
	if qs.seats.InUse >= qs.totalSeats && queue.Length() >= qs.queueMaxLength {
		return nil, accommodationErr
	}

	queuePostExecution, queuePostTimeout, err := queue.Enqueue(r)
	if err != nil {
		return nil, enqueueErr
	}
	trackers.QueueWait.Start()

	qs.vclock.Tick()
	seats, _ := r.EstimateCost()
	qs.seats.Waiting += seats
	qs.requests.Waiting += 1

	qs.events.Enqueued(queue, r)

	// if a request has been executed, that means the Dispatch method
	// had already dequeued, and scheduled it for execution, in this
	// case we no longer need to remove it from the queue.
	postExecution := disposerFunc(func() {
		defer qs.events.Disposed(r)
		func() {
			qs.lock.Lock()
			defer qs.lock.Unlock()

			queuePostExecution.Dispose()
			qs.finishLocked(r)
		}()
	})

	postTimeout := disposerFunc(func() {
		// if a request has timed out while waiting to be executed, that
		// means the Dispatch method had not had a successful attempt to
		// schedule it for execution, and thus it remains in the queue, so
		// we should remove it from its queue.
		defer qs.events.Timeout(r)
		func() {
			qs.lock.Lock()
			defer qs.lock.Unlock()

			queuePostTimeout.Dispose()
			qs.timeoutLocked(r)
		}()
	})

	// cleanup after execution
	return &queuedFinisher{request: r, postExecution: postExecution, postTimeout: postTimeout}, nil
}

func (qs *queueset) Dispatch() (bool, error) {
	qs.lock.Lock()
	defer qs.lock.Unlock()

	var minQueue fairqueue
	var minIndex int
	var minRequest fairqueuing.Request
	minFinishR := virtual.MaxSeatSeconds
	for range qs.queues {
		qs.robinIndex = (qs.robinIndex + 1) % len(qs.queues)
		queue := qs.queues[qs.robinIndex]
		oldest, ok := queue.Peek()
		if !ok {
			continue
		}

		thisFinishR := oldest.FinishR()
		if thisFinishR < minFinishR {
			minFinishR = thisFinishR
			minQueue = queue
			minRequest = oldest
		}
	}
	if minQueue == nil || minRequest == nil {
		return false, nil
	}

	// we set the round robin indexing to start at the chose queue
	// for the next round.  This way the non-selected queues
	// win in the case that the virtual finish times are the same
	qs.robinIndex = minIndex

	trackers := minRequest.LatencyTrackers()
	seats, _ := minRequest.EstimateCost()
	if qs.seats.InUse+seats > qs.totalSeats {
		return false, accommodationErr
	}

	_, queuePreExecution, ok := minQueue.Dequeue()
	if !ok {
		// we should never be here
		return false, queueEmptyErr
	}

	// do the things necessary before the decision is set
	func() {
		defer qs.events.Dequeued(minQueue, minRequest)

		qs.vclock.Tick()
		qs.requests.Waiting -= 1
		qs.seats.Waiting -= seats

		// There are two ways a request is dequeued:
		//  a) scheduler dequeues it and schedules it for execution
		//  b) the request times out while waiting in the queue, and it
		//     is being removed from the queue before being rejected.
		// we are here for a, and we want to track how much the request
		// spent inside of the queue waiting.
		trackers.QueueWait.Finish()
	}()

	if ok := minRequest.SetDecision(fairqueuing.DecisionExecute); !ok {
		return false, fmt.Errorf("failed to set  a decision for the request")
	}

	func() {
		defer qs.events.DecisionChanged(minRequest, fairqueuing.DecisionExecute)

		queuePreExecution.Dispose()
		qs.requests.Executing += 1
		qs.seats.InUse += seats

		// we have just made a decision to execute the request, we want
		// to track the latency from here until the user handler starts
		// executing.
		trackers.PostDecisionExecutionWait.Start()
	}()
	return true, nil
}

func (qs *queueset) finishLocked(r fairqueuing.Request) {
	seats, _ := r.EstimateCost()
	qs.seats.InUse -= seats
	qs.requests.Executing -= 1
	qs.vclock.Tick()
	r.OnDone(qs.vclock.RT())
}

func (qs *queueset) timeoutLocked(r fairqueuing.Request) {
	seats, _ := r.EstimateCost()
	qs.seats.Waiting -= seats
	qs.requests.Waiting -= 1
	qs.vclock.Tick()
	r.OnDone(qs.vclock.RT())

	// There are two ways a request is dequeued:
	//  a) scheduler dequeues it and schedules it for execution
	//  b) the request times out while waiting in the queue, and it
	//     is being removed from the queue before being rejected.
	//
	// we are here for b, and we want to track how much the request
	// spent inside of the queue waiting
	r.LatencyTrackers().QueueWait.Finish()
}

func (qs *queueset) getWorkLocked() (int, int) {
	naQueues := 0
	seatsRequested := 0
	for _, queue := range qs.queues {
		sc := queue.GetWork()
		if queue.Length() > 0 || sc.InUse > 0 {
			naQueues++
		}
		seatsRequested += int(sc.Total())
	}

	return int(math.Min(float64(seatsRequested), float64(qs.totalSeats))), naQueues
}
