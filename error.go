package sail

import (
	"fmt"
)

type Error struct {
	Message string
	Err     error
}

func (err Error) Error() string {
	return fmt.Sprintf("%s: %s", err.Message, err.Err)
}

func (err Error) Unwrap() error {
	return err.Err
}
