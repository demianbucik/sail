package sail

import (
	"fmt"
)

type sendErr struct {
	Message string
	Err     error
}

func (err sendErr) Error() string {
	return fmt.Sprintf("%s: %s", err.Message, err.Err)
}

func (err sendErr) Unwrap() error {
	return err.Err
}
