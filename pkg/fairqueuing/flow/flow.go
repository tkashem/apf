package flow

import (
	"crypto/sha256"
	"encoding/binary"
	"net/http"
)

type FlowHashType uint64

type RequestFlow struct {
	Hash FlowHashType
}

type FlowDistinguisherFunc func(*http.Request) []string

type FlowComputerFunc func(*http.Request, FlowDistinguisherFunc) RequestFlow

func (f FlowComputerFunc) ComputeFlow(r *http.Request, fn FlowDistinguisherFunc) RequestFlow {
	return f(r, fn)
}

func ComputeFlow(r *http.Request, distinguisherFn FlowDistinguisherFunc) RequestFlow {
	if distinguisherFn == nil {
		return RequestFlow{}
	}
	distinguishers := distinguisherFn(r)
	if len(distinguishers) == 0 {
		return RequestFlow{}
	}

	hash := sha256.New()
	var sep = [1]byte{0}
	for _, distinguisher := range distinguishers {
		hash.Write([]byte(distinguisher))
		hash.Write(sep[:])
	}

	var sum [32]byte
	hash.Sum(sum[:0])
	return RequestFlow{
		Hash: FlowHashType(binary.LittleEndian.Uint64(sum[:8])),
	}
}
