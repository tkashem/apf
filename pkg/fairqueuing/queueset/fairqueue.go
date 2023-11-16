package queueset

import (
	"math"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
)

type fairQueue struct {
	fifo FIFO

	// requests is the count in the real world.
	requests fairqueuing.RequestCount

	// seat(s) are the total number of "seats" currently occupied
	// by all the requests that are currently executing in this queue,
	// or waiting to be executed
	seats fairqueuing.SeatCount

	vclock virtual.RTClock
	// Finish time of the oldest request
	nextFinishR virtual.SeatSeconds
}

func (q *fairQueue) GetNextFinishR() virtual.SeatSeconds {
	return q.nextFinishR
}

func (q *fairQueue) Enqueue(r fairqueuing.Request) (fairqueuing.QueueCleanupCallbacks, error) {
	if q.fifo.Length() == 0 && q.seats.InUse == 0 {
		q.nextFinishR = virtual.MinSeatSeconds
	}

	removeFn := q.fifo.Enqueue(r)
	seats, width := r.EstimateCost()

	q.seats.Waiting += seats
	q.requests.Waiting += 1

	rt := q.vclock.RT()
	startR := virtual.SeatSeconds(math.Max(float64(rt), float64(q.nextFinishR)))
	finishR := startR + width

	q.nextFinishR = finishR
	r.OnStart(rt, startR, finishR)

	return fairqueuing.QueueCleanupCallbacks{
		PostExecution: fairqueuing.DisposerFunc(func() {}),
		PostTimeout: fairqueuing.DisposerFunc(func() {
			removeFn()
			q.seats.Waiting -= seats
			q.requests.Waiting -= 1
		}),
	}, nil
}

func (q *fairQueue) DequeueForExecution(dequeued, decided func(r fairqueuing.Request)) (fairqueuing.Request, bool) {
	request, ok := q.fifo.Dequeue()
	if !ok {
		return nil, false
	}

	seats, _ := request.EstimateCost()
	q.seats.Waiting -= seats
	q.requests.Waiting -= 1
	dequeued(request)

	if ok := request.SetDecision(fairqueuing.DecisionExecute); !ok {
		return nil, false
	}
	q.requests.Executing += 1
	decided(request)

	return request, true
}

func (q *fairQueue) Peek() (fairqueuing.Request, bool) {
	return q.fifo.Peek()
}

func (q *fairQueue) Length() int {
	return q.fifo.Length()
}

func (q *fairQueue) GetWork() fairqueuing.SeatCount {
	return q.seats
}
