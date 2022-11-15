package core

import (
	"net/http"
)

type SchedulerFactory interface {
	GetScheduler(*http.Request) Scheduler
}

type Scheduler interface {
	Schedule(*http.Request) (Finisher, error)
}

type Finisher interface {
	Finish(func())
}

type Disposer interface {
	Dispose()
}

type DisposerFunc func()

func (d DisposerFunc) Dispose() {
	d()
}

type SimpleSchedulerFactory struct {
	Scheduler Scheduler
}

func (sf SimpleSchedulerFactory) GetScheduler(_ *http.Request) Scheduler {
	return sf.Scheduler
}
