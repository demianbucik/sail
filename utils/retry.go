package utils

import (
	"errors"
	"time"
)

var ErrNeverTried = errors.New("never tried")

func Retry(tries int, backOff time.Duration, fn func() error) error {
	err := ErrNeverTried
	for i := 0; i < tries; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(backOff)
	}
	return err
}
