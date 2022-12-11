package virtual

func NewRTracker() rtracker {
	return rtracker{}
}

type RTracker interface {
	OnStart(arrivalR, startR, finishR SeatSeconds)
	OnDone(SeatSeconds)
	FinishR() SeatSeconds
}

type rtracker struct {
	arrivalR, startR, finishR SeatSeconds
	doneR                     SeatSeconds
}

func (t rtracker) OnStart(arrivalR, startR, finishR SeatSeconds) {
	t.arrivalR = arrivalR
	t.startR = startR
	t.finishR = finishR
}

func (t rtracker) OnDone(doneR SeatSeconds) {
	t.doneR = doneR
}

func (t rtracker) FinishR() SeatSeconds {
	return t.finishR
}
