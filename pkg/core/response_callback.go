package core

import (
	"net/http"
)

func NewDefaultResponseCallback() ResponseCallback {
	return defaultResponseCallback{}
}

type ResponseCallback interface {
	OnServed(http.ResponseWriter, *http.Request)
	OnRejected(http.ResponseWriter, *http.Request)
	OnExempt(http.ResponseWriter, *http.Request)
}

type defaultResponseCallback struct{}

func (d defaultResponseCallback) OnServed(w http.ResponseWriter, r *http.Request) {}

func (d defaultResponseCallback) OnRejected(w http.ResponseWriter, r *http.Request) {
	// Return a 429 status indicating "Too Many Requests"
	w.Header().Set("Retry-After", "1")
	http.Error(w, "Too many requests, please try again later.", http.StatusTooManyRequests)
}

func (d defaultResponseCallback) OnExempt(w http.ResponseWriter, r *http.Request) {}
