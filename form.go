package sail

import (
	"net/http"

	"github.com/demianbucik/sail/config"
)

type messageForm struct {
	Name     string `schema:"name,required" json:"name"`
	Email    string `schema:"email,required" json:"email"`
	Subject  string `schema:"subject,required" json:"subject"`
	Message  string `schema:"message,required" json:"message"`
	Honeypot string `json:"honeypot"`
}

type reCaptchaForm struct {
	ReCaptcha string `schema:"g-recaptcha-response,required" json:"g-recaptcha-response"`
}

type emailForm struct {
	messageForm
	reCaptchaForm
}

func (service *sailService) parseForm(request *http.Request) (*emailForm, error) {
	if err := request.ParseForm(); err != nil {
		return nil, err
	}

	form := &emailForm{}
	if err := service.formDecoder.Decode(&form.messageForm, request.Form); err != nil {
		return nil, err
	}

	if config.Env.ShouldVerifyReCaptcha() {
		if err := service.formDecoder.Decode(&form.reCaptchaForm, request.Form); err != nil {
			return nil, err
		}
	}
	if config.Env.HoneypotField != "" {
		form.Honeypot = request.Form.Get(config.Env.HoneypotField)
	}

	return form, nil
}
