package apf

import (
	"net/http"
)

func WithAPF(handler http.Handler) (http.Handler, error) {
	return func(w http.ResponseWriter, r *http.Request) {
		apfCtx := APFContextFrom(r.Context())
		if apfCtx == nil {
			handler.ServeHTTP(w, r)
			return
		}

		if apfCtx.Exempt() {
			handler.ServeHTTP(w, r)
			return
		}

		handler := shortLivedRequestHandler{handler: handler}
		handler.ServeHTTP(w, r)
		return
	}
}

type shortLivedRequestHandler struct {
	handler    http.Handler
	apfCtx     APFContext
	dispatcher Dispatcher
}

func (slh *shortLivedRequestHandler) ServeHTTP(w *http.ResponseWriter, r *http.Request) {
	var executed bool
	execute := func() {
		defer func() {
			executed = true
		}()
		slh.handler.ServeHTTP(w, r)
	}()

}
