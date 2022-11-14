package apf

import (
	"container/list"
	"net/http"
)

type APFRequest struct {
	Request   *http.Request
	ExecuteFn ExecuteFunc
}

// removeFromFIFOFunc removes a designated element from the list
// if that element is in the list.
// The complexity of the runtime cost is O(1).
// The returned value is the element removed, if indeed one was removed,
// otherwise `nil`.
type removeFromFIFOFunc func() *APFRequest

// walkFunc is called for each request in the list in the
// oldest -> newest order.
// ok: if walkFunc returns false then the iteration stops immediately.
// walkFunc may remove the given request from the fifo,
// but may not mutate the fifo in any othe way.
type walkFunc func(*APFRequest) (ok bool)

// Internal interface to abstract out the implementation details
// of the underlying list used to maintain the requests.
//
// Note that a fifo, including the removeFromFIFOFuncs returned from Enqueue,
// is not safe for concurrent use by multiple goroutines.
type fifo interface {
	// Enqueue enqueues the specified request into the list and
	// returns a removeFromFIFOFunc function that can be used to remove the
	// request from the list
	Enqueue(*APFRequest) removeFromFIFOFunc

	// Dequeue pulls out the oldest request from the list.
	Dequeue() (*APFRequest, bool)

	// Peek returns the oldest request without removing it.
	Peek() (*APFRequest, bool)

	// Length returns the number of requests in the list.
	Length() int

	// Walk iterates through the list in order of oldest -> newest
	// and executes the specified walkFunc for each request in that order.
	//
	// if the specified walkFunc returns false the Walk function
	// stops the walk an returns immediately.
	Walk(walkFunc)
}

// the FIFO list implementation is not safe for concurrent use by multiple
// goroutines.
type requestFIFO struct {
	*list.List
}

func newRequestFIFO() fifo {
	return &requestFIFO{
		List: list.New(),
	}
}

func (l *requestFIFO) Length() int {
	return l.Len()
}

func (l *requestFIFO) Enqueue(req *APFRequest) removeFromFIFOFunc {
	e := l.PushBack(req)

	return func() *APFRequest {
		if e.Value == nil {
			return nil
		}
		l.Remove(e)
		e.Value = nil
		return req
	}
}

func (l *requestFIFO) Dequeue() (*APFRequest, bool) {
	return l.getFirst(true)
}

func (l *requestFIFO) Peek() (*APFRequest, bool) {
	return l.getFirst(false)
}

func (l *requestFIFO) getFirst(remove bool) (*APFRequest, bool) {
	e := l.Front()
	if e == nil {
		return nil, false
	}

	if remove {
		defer func() {
			l.Remove(e)
			e.Value = nil
		}()
	}

	request, ok := e.Value.(*APFRequest)
	return request, ok
}

func (l *requestFIFO) Walk(f walkFunc) {
	var next *list.Element
	for current := l.Front(); current != nil; current = next {
		next = current.Next() // f is allowed to remove current
		if r, ok := current.Value.(*APFRequest); ok {
			if !f(r) {
				return
			}
		}
	}
}
