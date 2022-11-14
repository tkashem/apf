package apf

import (
  "sync"
)

type ExecuteFunc func()

type Dispatcher interface {

}

type dispatcher struct {
  lock sync.Mutex
  ceiling int
}

func (d *dispatcher)
