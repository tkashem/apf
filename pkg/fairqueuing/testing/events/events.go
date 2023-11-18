package events

import (
	clocktesting "k8s.io/utils/clock/testing"
)

type fake struct {
	clock clocktesting.FakePassiveClock
}

func (f *fake) Run() {}
