package utils

import (
	"sync"
	"sync/atomic"
)

const (
	onceNotDone = 0
	onceDone    = 1
)

type TryOnce struct {
	mu   sync.Mutex
	done uint32
}

// TryDo will call the provided function fn once if
// it hasn't yet returned nil during the previous calls.
func (once *TryOnce) TryDo(fn func() error) (err error) {
	if atomic.LoadUint32(&once.done) == onceNotDone {
		return once.doSlow(fn)
	}
	return nil
}

func (once *TryOnce) doSlow(fn func() error) (err error) {
	once.mu.Lock()
	defer once.mu.Unlock()

	if once.done == onceDone {
		return nil
	}

	defer func() {
		if err == nil {
			atomic.StoreUint32(&once.done, onceDone)
		}
	}()

	return fn()
}
