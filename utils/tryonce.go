package utils

import (
	"sync"
	"sync/atomic"
)

type TryOnce struct {
	mu   sync.Mutex
	done uint32
}

// Each call of TryDo will call the function fn once if
// it hasn't yet returned nil in the previous calls
func (once *TryOnce) TryDo(fn func() error) (err error) {
	if atomic.LoadUint32(&once.done) == 0 {
		return once.doSlow(fn)
	}
	return nil
}

func (once *TryOnce) doSlow(fn func() error) (err error) {
	once.mu.Lock()
	defer once.mu.Unlock()

	if once.done == 1 {
		return nil
	}

	defer func() {
		if err == nil {
			atomic.StoreUint32(&once.done, 1)
		}
	}()

	return fn()
}
