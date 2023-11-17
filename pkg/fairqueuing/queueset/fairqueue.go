package queueset

import (
	"math"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
)

type fairQueue struct {
	fifo fifo

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

func (q *fairQueue) Enqueue(r fairqueuing.Request) (disposer, disposer, error) {
	if q.fifo.Length() == 0 && q.seats.InUse == 0 {
		q.nextFinishR = virtual.MinSeatSeconds
	}

	disposer := q.fifo.Enqueue(r)
	seats, width := r.EstimateCost()

	q.seats.Waiting += seats
	q.requests.Waiting += 1

	rt := q.vclock.RT()
	startR := virtual.SeatSeconds(math.Max(float64(rt), float64(q.nextFinishR)))
	finishR := startR + width

	q.nextFinishR = finishR
	r.OnStart(rt, startR, finishR)

	postExecution := disposerFunc(func() {
		q.seats.InUse -= seats
		q.requests.Executing -= 1
	})
	postTimeout := disposerFunc(func() {
		disposer.Dispose()
		q.seats.Waiting -= seats
		q.requests.Waiting -= 1
	})
	return postExecution, postTimeout, nil
}

func (q *fairQueue) Dequeue() (fairqueuing.Request, disposer, bool) {
	request, ok := q.fifo.Dequeue()
	if !ok {
		return nil, nil, false
	}

	seats, _ := request.EstimateCost()
	q.seats.Waiting -= seats
	q.requests.Waiting -= 1

	// before execution begins
	preExecution := disposerFunc(func() {
		q.seats.InUse += seats
		q.requests.Executing += 1
	})
	return request, preExecution, true
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
