package fairqueuing

import (
	"net/http"

	"github.com/tkashem/apf/pkg/core"
)

type fairQueue struct {
	queue core.FIFO

	// requests is the count in the real world.
	requests Counter

	// seat(s) are the total number of "seats" currently occupied
	// by all the requests that are currently executing in this queue,
	// or waiting to be executed
	seats Counter
}

func (d *fairQueue) Enqueue(r *http.Request) (core.DisposerFunc, error) {
	scoped, err := core.RequestScopedFrom(r.Context())
	if err != nil {
		return nil, err
	}

	removeFn := d.queue.Enqueue(r)
	d.seats.Waiting += scoped.Estimate.Seats
	d.requests.Waiting += 1

	return func() {
		removeFn()
		d.seats.Waiting -= scoped.Estimate.Seats
		d.requests.Waiting -= 1
	}, nil
}

func (d *fairQueue) DequeueForExecution(dequeued, decided func()) (*http.Request, bool) {
	request, ok := d.queue.Dequeue()
	if !ok {
		return nil, false
	}
	scoped, err := core.RequestScopedFrom(request.Context())
	if err != nil {
		return nil, false
	}
	d.seats.Waiting -= scoped.Estimate.Seats
	d.requests.Waiting -= 1
	dequeued()

	if ok := scoped.Decision.Set(core.DecisionExecute); !ok {
		return nil, false
	}
	decided()
	return request, true
}

func (d *fairQueue) Peek() (*http.Request, bool) {
	return d.queue.Peek()
}

func (d *fairQueue) Length() int {
	return d.queue.Length()
}

func (d *fairQueue) GetVirtualFinish() core.SeatSeconds {
	return core.MinSeatSeconds
}
