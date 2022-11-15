package core

import (
	"net/http"
)

type Exemption interface {
	IsExempt(*http.Request) (bool, error)
}

func NewNoExemption() Exemption {
	return noExempt{}
}

type noExempt struct{}

func (ne noExempt) IsExempt(r *http.Request) (bool, error) {
	return false, nil
}
