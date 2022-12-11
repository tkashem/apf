package scheduler

import (
	"net/http"
)

type Events interface {
	Arrived(*http.Request)
	QueueAssigned(*http.Request)
	Enqueued(*http.Request)
	Dequeued(*http.Request)
	DecisionChanged(*http.Request, DecisionType)

	ExecutionStarting(*http.Request)
	ExecutionEnded(*http.Request)
	Disposed(*http.Request)
	Timeout(*http.Request)
}

func NoEvents() noEvent {
	return noEvent{}
}

type noEvent struct{}

func (noEvent) Arrived(*http.Request)                       {}
func (noEvent) QueueAssigned(*http.Request)                 {}
func (noEvent) Enqueued(*http.Request)                      {}
func (noEvent) Dequeued(*http.Request)                      {}
func (noEvent) DecisionChanged(*http.Request, DecisionType) {}
func (noEvent) ExecutionStarting(*http.Request)             {}
func (noEvent) ExecutionEnded(*http.Request)                {}
func (noEvent) Disposed(*http.Request)                      {}
func (noEvent) Timeout(*http.Request)                       {}
