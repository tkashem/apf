package defaults

import (
	"net/http"

	"github.com/tkashem/apf/pkg/fairqueuing/estimator"
	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
)

var (
	defaultCostEstimate = &estimate{}
)

func NewCostEstimator() estimator.CostEstimator {
	return uniformEstimator{}
}

type uniformEstimator struct{}

func (e uniformEstimator) EstimateCost(*http.Request) (estimator.CostEstimate, error) {
	return defaultCostEstimate, nil
}

type estimate struct{}

func (e estimate) GetWidth() virtual.SeatSeconds { return 10 }
func (e estimate) GetSeats() uint32              { return 1 }
