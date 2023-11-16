package queueset

import (
	"container/list"

	"github.com/tkashem/apf/pkg/fairqueuing"
)

// the FIFO list implementation is not safe for concurrent use
// by multiple goroutines.
type requestFIFO struct {
	*list.List
}

func NewFIFO() *requestFIFO {
	return &requestFIFO{
		List: list.New(),
	}
}

func (l *requestFIFO) Length() int {
	return l.Len()
}

func (l *requestFIFO) Enqueue(r fairqueuing.Request) fairqueuing.DisposerFunc {
	e := l.PushBack(r)

	return func() {
		if e.Value == nil {
			return
		}
		l.Remove(e)
		e.Value = nil
	}
}

func (l *requestFIFO) Dequeue() (fairqueuing.Request, bool) {
	return l.getFirst(true)
}

func (l *requestFIFO) Peek() (fairqueuing.Request, bool) {
	return l.getFirst(false)
}

func (l *requestFIFO) Walk(f WalkFunc) {
	var next *list.Element
	for current := l.Front(); current != nil; current = next {
		next = current.Next() // f is allowed to remove current
		if r, ok := current.Value.(fairqueuing.Request); ok {
			if !f(r) {
				return
			}
		}
	}
}

func (l *requestFIFO) getFirst(remove bool) (fairqueuing.Request, bool) {
	e := l.Front()
	if e == nil {
		return nil, false
	}

	if remove {
		defer func() {
			l.Remove(e)
			e.Value = nil
		}()
	}

	request, ok := e.Value.(fairqueuing.Request)
	return request, ok
}
