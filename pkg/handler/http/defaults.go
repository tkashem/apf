package http

import (
	"fmt"
	"net/http"
)

func NewDefaultErrorHandler() *errorHandler {
	return &errorHandler{}
}

type errorHandler struct{}

func (h *errorHandler) HandleError(w http.ResponseWriter, r *http.Request, _ error) {
	errorMsg := fmt.Sprintf("Internal Server Error: %#v", r.RequestURI)
	http.Error(w, errorMsg, http.StatusInternalServerError)
}

func NewNoExemption() noExempt {
	return noExempt{}
}

type noExempt struct{}

func (ne noExempt) IsExempt(r *http.Request) (bool, error) {
	return false, nil
}

func NewDefaultEvents() defaultEvents {
	return defaultEvents{}
}

type defaultEvents struct{}

func (d defaultEvents) OnServed(w http.ResponseWriter, _ *http.Request) {}

func (d defaultEvents) OnRejected(w http.ResponseWriter, _ *http.Request) {
	// Return a 429 status indicating "Too Many Requests"
	w.Header().Set("Retry-After", "1")
	http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
}

func (d defaultEvents) OnExempt(w http.ResponseWriter, r *http.Request) {}
func (d defaultEvents) Arrived(*http.Request)                           {}
