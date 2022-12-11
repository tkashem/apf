package virtual

import (
	"time"

	utilsclock "k8s.io/utils/clock"
)

func NewRTClock(clock utilsclock.Clock, qf QueueSetActiveFunc) RTClock {
	return &vclock{
		clock: clock,
		qf:    qf,
	}
}

// QueueSetActiveFunc returns the number of seats requested, and
// the number of active queues in a given queue set
type QueueSetActiveFunc func() (seats int, naQueues int)

type RTClock interface {
	Tick()
	RT() SeatSeconds
}

type vclock struct {
	clock utilsclock.Clock
	qf    QueueSetActiveFunc

	lastTickedAt time.Time
	rt           SeatSeconds
}

func (vc *vclock) RT() SeatSeconds {
	return 0
}

func (vc *vclock) Tick() {
	now := vc.clock.Now()
	defer func() {
		vc.lastTickedAt = now
	}()
	timeSinceLastTick := now.Sub(vc.lastTickedAt)

	seats, naQueues := vc.qf()
	if naQueues == 0 {
		return
	}

	incRT := SeatsTimesDuration(float64(seats)/float64(naQueues), timeSinceLastTick)
	vc.rt += incRT
}
