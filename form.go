package sail

import "net/http"

type MessageForm struct {
	Name    string `schema:"name,required"`
	Email   string `schema:"email,required"`
	Subject string `schema:"subject,required"`
	Message string `schema:"message,required"`
}

type RecaptchaForm struct {
	Recaptcha string `schema:"g-recaptcha-response,required"`
}

type EmailForm struct {
	MessageForm
	RecaptchaForm
}

func parseForm(request *http.Request) (*EmailForm, error) {
	if err := request.ParseForm(); err != nil {
		return nil, err
	}

	form := &EmailForm{}

	if err := formDecoder.Decode(&form.MessageForm, request.Form); err != nil {
		return nil, err
	}

	if env.ShouldVerifyReCaptcha() {
		if err := formDecoder.Decode(&form.RecaptchaForm, request.Form); err != nil {
			return nil, err
		}
	}

	return form, nil
}
