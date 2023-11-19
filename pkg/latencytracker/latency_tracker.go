package latencytracker

import (
	"time"

	"k8s.io/utils/clock"
)

func NewLatencyTracker(clock clock.PassiveClock) *latencyTracker {
	return &latencyTracker{clock: clock}
}

type latencyTracker struct {
	clock     clock.PassiveClock
	startedAt time.Time
	duration  time.Duration
}

func (t *latencyTracker) Start() {
	t.startedAt = t.clock.Now()
}

func (t *latencyTracker) Finish() {
	t.duration = t.clock.Since(t.startedAt)
}

func (t *latencyTracker) Get() (time.Time, time.Duration) {
	return t.startedAt, t.duration
}
