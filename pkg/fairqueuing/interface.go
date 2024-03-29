package fairqueuing

import (
	"context"

	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
)

type FlowIDType uint64

type Request interface {
	virtual.RTracker
	DecisionWaiterSetter

	GetFlowID() FlowIDType
	EstimateCost() (seats uint32, width virtual.SeatSeconds)
	Context() context.Context
	CancelFunc() context.CancelFunc
	String() string
	LatencyTrackers() LatencyTrackers
}

type FairQueueAccessor interface {
	TotalQueues() int
	GetFairQueue(int) FairQueue
}

type QueueSelector interface {
	SelectQueue(FairQueueAccessor, FlowIDType) (FairQueue, error)
}

type LatencyTracker interface {
	Start()
	Finish()
}

type LatencyTrackers struct {
	QueueWait                 LatencyTracker
	PostDecisionExecutionWait LatencyTracker
	ExecutionDuration         LatencyTracker
	TotalDuration             LatencyTracker
}

type SeatCount struct {
	InUse   uint32
	Waiting uint32
}

func (sc SeatCount) Total() uint32 {
	return sc.InUse + sc.Waiting
}

type RequestCount struct {
	Executing uint32
	Waiting   uint32
}

func (rc RequestCount) Total() uint32 {
	return rc.Executing + rc.Waiting
}

// A decision about a request
type DecisionType int

// Values passed through a request's decision
const (
	DecisionNone DecisionType = iota

	// This one's context timed out / was canceled
	DecisionTimeout

	// Serve this one
	DecisionExecute
)

type DecisionSetter interface {
	SetDecision(DecisionType) bool
}

type DecisionWaiter interface {
	WaitForDecision() DecisionType
}

type DecisionWaiterSetter interface {
	DecisionWaiter
	DecisionSetter
}

type FairQueue interface {
	GetNextFinishR() virtual.SeatSeconds
	Peek() (Request, bool)
	Length() int
	GetWork() SeatCount
	String() string
	ID() uint32
}

type Finisher interface {
	Finish(func())
}

type FairQueueSet interface {
	Name() string
	Enqueue(Request) (Finisher, error)
	Dispatch() (bool, error)
	EnqueueAndDispatch(Request) (Finisher, error)
}
