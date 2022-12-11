package scheduler

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
