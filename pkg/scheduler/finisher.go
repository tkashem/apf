package scheduler

type Finisher interface {
	Finish(func())
}

var _ Finisher = RejectedRequest{}

type RejectedRequest struct{}

func (RejectedRequest) Finish(_ func()) {}
