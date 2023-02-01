package queueset

import (
	"fmt"
	"math"
	"net/http"
	"sync"

	apfcontext "github.com/tkashem/apf/pkg/context"
	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/container"
	"github.com/tkashem/apf/pkg/fairqueuing/queueassigner"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
	"github.com/tkashem/apf/pkg/scheduler"
	"k8s.io/utils/clock"
)

var (
	accommodationErr = fmt.Errorf("cannot accommodate request")
	queueEmptyErr    = fmt.Errorf("selected queue should not be empty")
	enqueueErr       = fmt.Errorf("failed to enqueue the request")
)

func NewQueueSet(config *CompletedConfig) (*queueset, error) {
	if config.TotalSeats < 1 {
		return nil, fmt.Errorf("seats must be positive")
	}

	qs := &queueset{}

	clock := clock.RealClock{}
	qs.clock = clock

	vclock := virtual.NewRTClock(clock, qs.getWorkLocked)
	qs.vclock = vclock

	queues := make([]fairqueuing.FairQueue, config.NQueues)
	for i := range queues {
		queues[i] = &fairQueue{
			fifo:   container.NewFIFO(),
			vclock: vclock,
		}
	}
	qs.queues = queues

	config.QueueAssignerFactory.WithQueueSetAccessor(qs)
	assigner, err := config.QueueAssignerFactory.New()
	if err != nil {
		return nil, err
	}
	qs.assigner = assigner

	qs.totalSeats = config.TotalSeats
	qs.queueMaxLength = config.QueueMaxLength
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
	queues         []fairqueuing.FairQueue
	robinIndex     int
	events         scheduler.Events
	clock          clock.Clock
	vclock         virtual.RTClock
	assigner       queueassigner.QueueAssigner
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

func (qs *queueset) Enqueue(r *http.Request) (fairqueuing.QueueCleanupCallbacks, error) {
	scoped, err := apfcontext.RequestScopedFrom(r.Context())
	if err != nil {
		return fairqueuing.QueueCleanupCallbacks{}, err
	}

	qs.lock.Lock()
	defer qs.lock.Unlock()

	queue, err := qs.assigner.Assign(scoped.Flow)
	if err != nil {
		return fairqueuing.QueueCleanupCallbacks{}, fmt.Errorf("error assigning queue - %v", err)
	}
	qs.events.QueueAssigned(r)

	// can we fit the request?
	if qs.seats.InUse >= qs.totalSeats && queue.Length() >= qs.queueMaxLength {
		return fairqueuing.QueueCleanupCallbacks{}, accommodationErr
	}

	queueDisposer, err := queue.Enqueue(r)
	if err != nil {
		return fairqueuing.QueueCleanupCallbacks{}, enqueueErr
	}
	scoped.QueueWaitLatency = apfcontext.NewLatencyTracker(qs.clock)
	qs.vclock.Tick()
	qs.seats.Waiting += scoped.Estimate.GetSeats()
	qs.requests.Waiting += 1

	qs.events.Enqueued(r)

	disposer := fairqueuing.QueueCleanupCallbacks{
		// if a request has been executed, that means the Dispatch method
		// had already dequeued, and scheduled it for execution, in this
		// case we no longer need to remove it from the queue.
		PostExecution: fairqueuing.DisposerFunc(func() {
			defer qs.events.Disposed(r)
			func() {
				qs.lock.Lock()
				defer qs.lock.Unlock()

				queueDisposer.PostExecution.Dispose()
				qs.finishLocked(r)
			}()
		}),
		PostTimeout: fairqueuing.DisposerFunc(func() {
			// if a request has timed out while waiting to be executed, that
			// means the Dispatch method had not had a successful attempt to
			// schedule it for execution, and thus it remains in the queue, so
			// we should remove it from its queue.
			defer qs.events.Timeout(r)
			func() {
				qs.lock.Lock()
				defer qs.lock.Unlock()

				queueDisposer.PostTimeout.Dispose()
				qs.timeoutLocked(r)
			}()
		}),
	}

	// cleanup after execution
	return disposer, nil
}

func (qs *queueset) Dispatch() (bool, error) {
	qs.lock.Lock()
	defer qs.lock.Unlock()

	var minQueue fairqueuing.FairQueue
	var minIndex int
	var minScoped *apfcontext.RequestScoped
	minFinishR := virtual.MaxSeatSeconds
	for range qs.queues {
		qs.robinIndex = (qs.robinIndex + 1) % len(qs.queues)
		queue := qs.queues[qs.robinIndex]
		oldest, ok := queue.Peek()
		if !ok {
			continue
		}
		scoped, err := apfcontext.RequestScopedFrom(oldest.Context())
		if err != nil {
			// impossible case, log error
			continue
		}

		thisFinishR := scoped.RTracker.FinishR()
		if thisFinishR < minFinishR {
			minFinishR = thisFinishR
			minQueue = queue
			minScoped = scoped
		}
	}
	if minQueue == nil || minScoped == nil {
		return false, nil
	}

	// we set the round robin indexing to start at the chose queue
	// for the next round.  This way the non-selected queues
	// win in the case that the virtual finish times are the same
	qs.robinIndex = minIndex

	seats := minScoped.Estimate.GetSeats()
	if qs.seats.InUse+seats > qs.totalSeats {
		return false, accommodationErr
	}

	// remove the request from queue for execution
	_, ok := minQueue.DequeueForExecution(func(r *http.Request) {
		defer qs.events.Dequeued(r)

		qs.vclock.Tick()
		qs.requests.Waiting -= 1
		qs.seats.Waiting -= seats

		// There are two ways a request is dequeued:
		//  a) scheduler dequeues it and schedules it for execution
		//  b) the request times out while waiting in the queue, and it
		//     is being removed from the queue before being rejected.
		// we are here for a, and we want to track how much the request
		// spent inside of the queue waiting.
		minScoped.QueueWaitLatency.Done()
	}, func(r *http.Request) {
		defer qs.events.DecisionChanged(r, scheduler.DecisionExecute)

		qs.requests.Executing += 1
		qs.seats.InUse += seats

		// we have just made a decision to execute the request, we want
		// to track the latency from here until the user handler starts
		// executing.
		minScoped.PostDecisionExecutionWait = apfcontext.NewLatencyTracker(qs.clock)
	})
	if !ok {
		return false, queueEmptyErr
	}

	return true, nil
}

func (qs *queueset) finishLocked(r *http.Request) {
	scoped, err := apfcontext.RequestScopedFrom(r.Context())
	if err != nil {
		// impossible case, log error
		return
	}

	qs.seats.InUse -= scoped.Estimate.GetSeats()
	qs.requests.Executing -= 1
	qs.vclock.Tick()
	scoped.RTracker.OnDone(qs.vclock.RT())
}

func (qs *queueset) timeoutLocked(r *http.Request) {
	scoped, err := apfcontext.RequestScopedFrom(r.Context())
	if err != nil {
		// impossible case, log error
		return
	}

	qs.seats.Waiting -= scoped.Estimate.GetSeats()
	qs.requests.Waiting -= 1
	qs.vclock.Tick()
	scoped.RTracker.OnDone(qs.vclock.RT())

	// There are two ways a request is dequeued:
	//  a) scheduler dequeues it and schedules it for execution
	//  b) the request times out while waiting in the queue, and it
	//     is being removed from the queue before being rejected.
	//
	// we are here for b, and we want to track how much the request
	// spent inside of the queue waiting
	scoped.QueueWaitLatency.Done()
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
