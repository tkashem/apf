package core

import (
	"fmt"
	"net/http"
)

type ErrorHandler interface {
	HandleError(w http.ResponseWriter, r *http.Request, err error)
}

func NewDefaultErrorHandler() ErrorHandler {
	return &defaultErrorHandler{}
}

type defaultErrorHandler struct{}

func (h *defaultErrorHandler) HandleError(w http.ResponseWriter, r *http.Request, _ error) {
	errorMsg := fmt.Sprintf("Internal Server Error: %#v", r.RequestURI)
	http.Error(w, errorMsg, http.StatusInternalServerError)
}
