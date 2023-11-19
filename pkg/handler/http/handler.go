package http

import (
	"context"
	"net/http"

	"github.com/tkashem/apf/pkg/fairqueuing"
	"k8s.io/utils/clock"
)

type Exempt interface {
	IsExempt(r *http.Request) (bool, error)
}

type ContextTransformer interface {
	Derive(*http.Request) (context.Context, context.CancelFunc)
}

type ErrorHandler interface {
	HandleError(http.ResponseWriter, *http.Request, error)
}

type Events interface {
	Arrived(*http.Request)
	OnServed(http.ResponseWriter, *http.Request)
	OnRejected(http.ResponseWriter, *http.Request)
	OnExempt(http.ResponseWriter, *http.Request)
}

type Converter interface {
	Convert(*http.Request) (fairqueuing.Request, error)
}

type EnqueueAndDispatcher interface {
	EnqueueAndDispatch(fairqueuing.Request) (fairqueuing.Finisher, error)
}

type Config struct {
	Exempt       Exempt
	ErrorHandler ErrorHandler
	Events       Events
	Clock        clock.Clock
	Converter    Converter
}

func NewAPFHandler(inner http.Handler, dispatcher EnqueueAndDispatcher, c *Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := c.Events
		e.Arrived(r)

		exempt, err := c.Exempt.IsExempt(r)
		if err != nil {
			c.ErrorHandler.HandleError(w, r, err)
			return
		}
		if exempt {
			e.OnExempt(w, r)
			inner.ServeHTTP(w, r)
			return
		}

		fqr, err := c.Converter.Convert(r)
		if err != nil {
			c.ErrorHandler.HandleError(w, r, err)
			return
		}
		if cancel := fqr.CancelFunc(); cancel != nil {
			defer cancel()
		}

		finisher, err := dispatcher.EnqueueAndDispatch(fqr)
		if err != nil {
			c.ErrorHandler.HandleError(w, r, err)
			return
		}

		var served bool
		finisher.Finish(func() {
			served = true
			inner.ServeHTTP(w, r)
		})

		if !served {
			e.OnRejected(w, r)
			return
		}
		e.OnServed(w, r)
	})
}
