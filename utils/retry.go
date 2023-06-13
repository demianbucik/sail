package utils

import (
	"errors"
	"time"
)

var ErrNeverTried = errors.New("never tried")

func Retry(retries int, backOff time.Duration, fn func() error) error {
	err := ErrNeverTried
	for i := 0; i < retries+1; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(backOff)
		backOff = 2 * backOff
	}
	return err
}
