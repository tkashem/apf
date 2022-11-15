package core

import (
	"math"
	"net/http"
)

// walkFunc is called for each request in the list in the
// oldest -> newest order.
// ok: if walkFunc returns false then the iteration stops immediately.
// walkFunc may remove the given request from the fifo,
// but may not mutate the fifo in any other way.
type WalkFunc func(r *http.Request) (ok bool)

// Internal interface to abstract out the implementation details
// of the underlying list used to maintain the requests.
//
// Note that a fifo, including the DisposerFunc returned from Enqueue,
// is not safe for concurrent use by multiple goroutines.
type FIFO interface {
	// Enqueue enqueues the specified request into the list and
	// returns a DisposerFunc function that can be used to remove the
	// request from the list
	Enqueue(*http.Request) DisposerFunc

	// Dequeue pulls out the oldest request from the list.
	Dequeue() (*http.Request, bool)

	// Peek returns the oldest request without removing it.
	Peek() (*http.Request, bool)

	// Length returns the number of requests in the list.
	Length() int

	// Walk iterates through the list in order of oldest -> newest
	// and executes the specified walkFunc for each request in that order.
	//
	// if the specified walkFunc returns false the Walk function
	// stops the walk an returns immediately.
	Walk(WalkFunc)
}

type SeatSeconds uint64

// MaxSeatsSeconds is the maximum representable value of SeatSeconds
const MaxSeatSeconds = SeatSeconds(math.MaxUint64)

// MinSeatSeconds is the lowest representable value of SeatSeconds
const MinSeatSeconds = SeatSeconds(0)

type FairQueue interface {
	Enqueue(*http.Request) (DisposerFunc, error)
	DequeueForExecution(dequeued, decided func()) (*http.Request, bool)
	GetVirtualFinish() SeatSeconds
	Peek() (*http.Request, bool)
	Length() int
}

// used by the shuffle sharding logic to pick a queue
type FairQueueAccessor interface {
	Total() int
	Get(int) FairQueue
}

type QueueAssigner interface {
	Assign(accessor FairQueueAccessor) FairQueue
}

type SchedulingEvents interface {
	Arrived(*http.Request)
	QueueAssigned(*http.Request)
	Enqueued(*http.Request)
	Dequeued(*http.Request)
	DecisionChanged(*http.Request, DecisionType)

	ExecutionStarting(*http.Request)
	ExecutionEnded(*http.Request)
	Disposed(*http.Request)
	Timeout(*http.Request)
}

type PostDequeueCallbacks struct {
	PostExecution Disposer
	PostTimeout   Disposer
}

type FairQueueSet interface {
	Name() string
	Enqueue(*http.Request, QueueAssigner) (PostDequeueCallbacks, error)
	Dispatch() (bool, error)
}
