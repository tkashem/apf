package scheduler

import (
	"net/http"
)

type SchedulerFactory interface {
	GetScheduler(*http.Request) Scheduler
}

type Scheduler interface {
	Schedule(*http.Request) (Finisher, error)
}

type SimpleSchedulerFactory struct {
	Scheduler Scheduler
}

func (sf SimpleSchedulerFactory) GetScheduler(_ *http.Request) Scheduler {
	return sf.Scheduler
}
