package queueset

import (
	"github.com/tkashem/apf/pkg/fairqueuing"
)

type Events interface {
	QueueSelected(fairqueuing.FairQueue, fairqueuing.Request)
	Enqueued(fairqueuing.FairQueue, fairqueuing.Request)
	Dequeued(fairqueuing.FairQueue, fairqueuing.Request)
	DecisionChanged(fairqueuing.Request, fairqueuing.DecisionType)

	Disposed(fairqueuing.Request)
	Timeout(fairqueuing.Request)
}

// walkFunc is called for each request in the list in the
// oldest -> newest order.
// ok: if walkFunc returns false then the iteration stops immediately.
// walkFunc may remove the given request from the fifo,
// but may not mutate the fifo in any other way.
type WalkFunc func(r fairqueuing.Request) (ok bool)

// Internal interface to abstract out the implementation details
// of the underlying list used to maintain the requests.
//
// Note that a fifo, including the DisposerFunc returned from Enqueue,
// is not safe for concurrent use by multiple goroutines.
type FIFO interface {
	// Enqueue enqueues the specified request into the list and
	// returns a DisposerFunc function that can be used to remove the
	// request from the list
	Enqueue(fairqueuing.Request) fairqueuing.DisposerFunc

	// Dequeue pulls out the oldest request from the list.
	Dequeue() (fairqueuing.Request, bool)

	// Peek returns the oldest request without removing it.
	Peek() (fairqueuing.Request, bool)

	// Length returns the number of requests in the list.
	Length() int

	// Walk iterates through the list in order of oldest -> newest
	// and executes the specified walkFunc for each request in that order.
	//
	// if the specified walkFunc returns false the Walk function
	// stops the walk an returns immediately.
	Walk(WalkFunc)
}
