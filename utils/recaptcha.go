//go:generate mockery -inpkg -name=ReCaptcha

package utils

import (
	"gopkg.in/ezzarghili/recaptcha-go.v4"
)

type ReCaptcha interface {
	Verify(challenge string) error
	VerifyWithOptions(challenge string, options recaptcha.VerifyOption) error
}
