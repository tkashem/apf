package core

import (
	"net/http"
)

func NewUniformWorkEstimator() WorkEstimator {
	return uniformEstimator{}
}

type WorkEstimate struct {
	Seats int32
}

type WorkEstimator interface {
	Estimate(r *http.Request) (WorkEstimate, error)
}

type uniformEstimator struct{}

func (e uniformEstimator) Estimate(r *http.Request) (WorkEstimate, error) {
	return WorkEstimate{
		Seats: 1,
	}, nil
}
