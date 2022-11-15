package scheduler

import (
	"net/http"

	"github.com/tkashem/apf/pkg/core"
)

type SchedulingEvents struct{}

func (c SchedulingEvents) Arrived(*http.Request)                            {}
func (c SchedulingEvents) QueueAssigned(*http.Request)                      {}
func (c SchedulingEvents) Enqueued(*http.Request)                           {}
func (c SchedulingEvents) Dequeued(*http.Request)                           {}
func (c SchedulingEvents) DecisionChanged(*http.Request, core.DecisionType) {}
func (c SchedulingEvents) ExecutionStarting(*http.Request)                  {}
func (c SchedulingEvents) ExecutionEnded(*http.Request)                     {}
func (c SchedulingEvents) Disposed(*http.Request)                           {}
func (c SchedulingEvents) Timeout(*http.Request)                            {}
