package apf

import (
	"context"
	"net/http"
)

type APFContextKeyType int

const APFContextKey APFContextKeyType = iota

type APFContext interface {
	Exempt() bool
	Flow() string

	WorkEstimator() WorkEstimator
}

type WorkEstimator interface {
	Estimate(r http.Request) WorkEstimate
}

type WorkEstimate struct {
	Seats int
}

func APFContextFrom(ctx context.Context) APFContext {
	if apfContext, ok := ctx.Value(APFContextKey); ok {
		return apfContext
	}
	return nil
}
