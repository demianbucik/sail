package sail

import (
	"fmt"
	"net/url"
)

type Error struct {
	Message string
	Form    *MessageForm
	Err     error
}

func (err Error) Error() string {
	if err.Form != nil {
		formValues := make(url.Values)
		if formErr := formEncoder.Encode(err.Form, formValues); formErr == nil {
			return fmt.Sprintf("%s, form '%s': %s", err.Message, formValues.Encode(), err.Err)
		}
	}
	return fmt.Sprintf("%s: %s", err.Message, err.Err)
}

func (err Error) Unwrap() error {
	return err.Err
}
