package defaults

import (
	"fmt"
	"net/http"
)

func NewDefaultErrorHandler() *rrrorHandler {
	return &rrrorHandler{}
}

type rrrorHandler struct{}

func (h *rrrorHandler) HandleError(w http.ResponseWriter, r *http.Request, _ error) {
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

func NewDefaultResponseCallback() defaultResponseCallback {
	return defaultResponseCallback{}
}

type defaultResponseCallback struct{}

func (d defaultResponseCallback) OnServed(w http.ResponseWriter, _ *http.Request) {}

func (d defaultResponseCallback) OnRejected(w http.ResponseWriter, _ *http.Request) {
	// Return a 429 status indicating "Too Many Requests"
	w.Header().Set("Retry-After", "1")
	http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
}

func (d defaultResponseCallback) OnExempt(w http.ResponseWriter, r *http.Request) {}
