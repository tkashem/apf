package fairqueuing

import (
	"net/http"

	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
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

type SeatCount struct {
	InUse   uint32
	Waiting uint32
}

func (sc SeatCount) Total() uint32 {
	return sc.InUse + sc.Waiting
}

type RequestCount struct {
	Executing uint32
	Waiting   uint32
}

func (rc RequestCount) Total() uint32 {
	return rc.Executing + rc.Waiting
}

type Disposer interface {
	Dispose()
}

type DisposerFunc func()

func (d DisposerFunc) Dispose() {
	d()
}

type QueueCleanupCallbacks struct {
	PostExecution Disposer
	PostTimeout   Disposer
}

type FairQueue interface {
	Enqueue(*http.Request) (QueueCleanupCallbacks, error)
	DequeueForExecution(dequeued, decided func(*http.Request)) (*http.Request, bool)

	GetNextFinishR() virtual.SeatSeconds

	Peek() (*http.Request, bool)
	Length() int
	GetWork() SeatCount
}

type FairQueueSet interface {
	Name() string
	Enqueue(*http.Request) (QueueCleanupCallbacks, error)
	Dispatch() (bool, error)
}
