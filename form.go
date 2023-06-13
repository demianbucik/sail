package sail

import (
	"net/http"
)

type EmailForm struct {
	Name    string `schema:"name,required" json:"name"`
	Email   string `schema:"email,required" json:"email"`
	Subject string `schema:"subject,required" json:"subject"`
	Message string `schema:"message,required" json:"message"`

	ReCaptchaResponse string `schema:"g-recaptcha-response" json:"g-recaptcha-response"`

	HoneypotValue string `schema:"-" json:"honeypot-value"`
}

func (service *sailService) parseForm(request *http.Request) (*EmailForm, error) {
	if err := request.ParseForm(); err != nil {
		return nil, err
	}

	form := &EmailForm{}
	if err := service.formDecoder.Decode(form, request.Form); err != nil {
		return nil, err
	}

	if service.env.HoneypotCheckEnabled() {
		form.HoneypotValue = request.Form.Get(service.env.HoneypotField)
	}

	return form, nil
}
