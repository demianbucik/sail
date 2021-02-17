package utils

import (
	"errors"
	"time"
)

func Retry(tries int, backOff time.Duration, fn func() error) error {
	err := errors.New("never tried")
	for i := 0; i < tries; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(backOff)
	}
	return err
}
