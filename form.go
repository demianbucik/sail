package sail

import (
	"net/http"

	"github.com/demianbucik/sail/config"
)

type MessageForm struct {
	Name     string `schema:"name,required" json:"name"`
	Email    string `schema:"email,required" json:"email"`
	Subject  string `schema:"subject,required" json:"subject"`
	Message  string `schema:"message,required" json:"message"`
	Honeypot string `json:"honeypot"`
}

type ReCaptchaForm struct {
	ReCaptcha string `schema:"g-recaptcha-response,required" json:"g-recaptcha-response"`
}

type EmailForm struct {
	MessageForm
	ReCaptchaForm
}

func parseForm(request *http.Request) (*EmailForm, error) {
	if err := request.ParseForm(); err != nil {
		return nil, err
	}

	form := &EmailForm{}
	if err := formDecoder.Decode(&form.MessageForm, request.Form); err != nil {
		return nil, err
	}

	if config.Env.ShouldVerifyReCaptcha() {
		if err := formDecoder.Decode(&form.ReCaptchaForm, request.Form); err != nil {
			return nil, err
		}
	}
	if config.Env.HoneypotField != "" {
		form.Honeypot = request.Form.Get(config.Env.HoneypotField)
	}

	return form, nil
}
