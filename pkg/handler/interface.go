package handler

import (
	"net/http"
)

type Exemption interface {
	IsExempt(*http.Request) (bool, error)
}

type ErrorHandler interface {
	HandleError(w http.ResponseWriter, r *http.Request, err error)
}

type ResponseCallback interface {
	OnServed(http.ResponseWriter, *http.Request)
	OnRejected(http.ResponseWriter, *http.Request)
	OnExempt(http.ResponseWriter, *http.Request)
}
