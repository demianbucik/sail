package sail

import (
	"fmt"
	"net/url"
)

type Error struct {
	Message string
	Form    url.Values
	Err     error
}

func (err Error) Error() string {
	if len(err.Form) > 0 {
		return fmt.Sprintf("%s, form '%s': %s", err.Message, err.Form.Encode(), err.Err)
	}
	return fmt.Sprintf("%s: %s", err.Message, err.Err)
}

func (err Error) Unwrap() error {
	return err.Err
}
