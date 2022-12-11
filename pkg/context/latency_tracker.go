package context

import (
	"time"

	"k8s.io/utils/clock"
)

type latencyTracker struct {
	clock     clock.PassiveClock
	startedAt time.Time
	duration  time.Duration
}

func (t latencyTracker) Done() {
	t.duration = t.clock.Since(t.startedAt)
}

func (t latencyTracker) Get() (time.Time, time.Duration) {
	return t.startedAt, t.duration
}

func NewLatencyTracker(clock clock.PassiveClock) latencyTracker {
	return latencyTracker{
		clock:     clock,
		startedAt: clock.Now(),
	}
}
