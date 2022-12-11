package estimator

import (
	"fmt"
	"net/http"
	"time"

	"github.com/tkashem/apf/pkg/fairqueuing/virtual"
)

const (
	DefaultServiceDuration = 1 * time.Second
	DefaultMinimumSeats    = 1
)

func NewCostEstimator(fn UserProvidedCostEstimatorFunc) (*estimator, error) {
	if fn == nil {
		return nil, fmt.Errorf("UserProvidedCostEstimatorFunc must be supplied")
	}
	return &estimator{estimatorFn: fn}, nil
}

// UserProvidedCostEstimatorFunc is the user supplied function that estimates
// the cost of a given request
//   - seats returns the number of seats occupied by this request
//   - duration is the amount of time the request can take in real
//     time to complete, if duration is zero PF will use the default
//     duration DefaultServiceDurations
type UserProvidedCostEstimatorFunc func(*http.Request) (seats uint32, duration time.Duration)

type CostEstimator interface {
	EstimateCost(*http.Request) (CostEstimate, error)
}

type CostEstimate interface {
	GetWidth() virtual.SeatSeconds
	GetSeats() uint32
}

type estimator struct {
	estimatorFn UserProvidedCostEstimatorFunc
}

func (e estimator) EstimateCost(r *http.Request) (CostEstimate, error) {
	seats, duration := e.estimatorFn(r)
	if seats == 0 {
		seats = DefaultMinimumSeats
	}
	if duration == 0 {
		duration = DefaultServiceDuration
	}
	return estimate{seats: seats, duration: duration}, nil
}

type estimate struct {
	seats    uint32
	duration time.Duration
}

func (e estimate) GetWidth() virtual.SeatSeconds {
	return virtual.SeatsTimesDuration(float64(e.seats), e.duration)
}
func (e estimate) GetSeats() uint32 { return e.seats }
