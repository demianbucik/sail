package sail

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/gorilla/schema"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gopkg.in/ezzarghili/recaptcha-go.v4"

	"github.com/demianbucik/sail/utils"
)

const (
	tries        = 3
	retryBackOff = 10 * time.Millisecond
)

var (
	once utils.TryOnce

	env               *utils.Environ
	formDecoder       *schema.Decoder
	formEncoder       *schema.Encoder
	reCaptcha         *recaptcha.ReCAPTCHA
	confirmationTempl *template.Template
)

func Init(parseFunc func(*utils.Environ) error) {
	err := once.TryDo(func() error {
		var err error
		env, err = utils.ParseEnv(parseFunc)
		if err != nil {
			return fmt.Errorf("init error: %w", err)
		}

		formDecoder = schema.NewDecoder()
		formEncoder = schema.NewEncoder()
		formDecoder.IgnoreUnknownKeys(true)

		if env.ShouldVerifyReCaptcha() {
			// Error can only occur if secret key is empty
			rc, _ := recaptcha.NewReCAPTCHA(env.ReCaptchaSecretKey, env.ParseReCaptchaVersion(), 10*time.Second)
			reCaptcha = &rc
		}

		confirmationTempl, err = template.New("confirmation").
			Option("missingkey=error").Parse(env.ConfirmationTemplate)
		if err != nil {
			return fmt.Errorf("init error: %w", err)
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
}

func SendEmailHandler(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		if rvr := recover(); rvr != nil {
			log.Println("panic:", rvr)
			http.Error(writer, "internal server error", http.StatusInternalServerError)
		}
	}()

	Init(utils.ParseFromOSEnv)

	if err := sendEmailAndConfirmation(request); err != nil {
		log.Println(err)
		http.Redirect(writer, request, env.ErrorPage, http.StatusSeeOther)
		return
	}

	http.Redirect(writer, request, env.ThankYouPage, http.StatusSeeOther)
}

func verifyReCaptcha(challenge, remoteAddr string) error {
	var err error
	_ = utils.Retry(tries, retryBackOff, func() error {
		err = reCaptcha.VerifyWithOptions(challenge, recaptcha.VerifyOption{
			RemoteIP: remoteAddr,
		})
		if err != nil {
			var rcErr *recaptcha.Error
			if errors.As(err, &rcErr) && rcErr.RequestError {
				return err
			}
		}
		return nil
	})
	return err
}

func sendEmailAndConfirmation(request *http.Request) error {
	form, err := parseForm(request)
	if err != nil {
		return &Error{"invalid form", nil, err}
	}

	if env.ShouldVerifyReCaptcha() {
		err = verifyReCaptcha(form.Recaptcha, request.RemoteAddr)
		if err != nil {
			return &Error{"recaptcha failed", form.MessageForm, err}
		}
	}

	client := sendgrid.NewSendClient(env.SendGridApiKey)

	message := newMessage(form)
	err = utils.Retry(tries, retryBackOff, getSendMessageFunc(client, message))
	if err != nil {
		return &Error{"sending email failed", form.MessageForm, err}
	}

	confirmation, err := newConfirmation(form)
	if err != nil {
		return &Error{"template error", form.MessageForm, err}
	}
	err = utils.Retry(tries, retryBackOff, getSendMessageFunc(client, confirmation))
	if err != nil {
		return &Error{"sending confirmation failed", form.MessageForm, err}
	}

	return nil
}

func getSendMessageFunc(client *sendgrid.Client, message *mail.SGMailV3) func() error {
	return func() error {
		resp, err := client.Send(message)
		if err != nil {
			return err
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("response code '%d' not ok, body '%s'", resp.StatusCode, resp.Body)
		}
		return nil
	}
}

func newMessage(form *EmailForm) *mail.SGMailV3 {
	from := mail.NewEmail(env.NoReplyName, env.NoReplyEmail)
	to := mail.NewEmail(env.RecipientName, env.RecipientEmail)
	replyTo := mail.NewEmail(form.Name, form.Email)

	message := mail.NewSingleEmail(from, form.Subject, to, form.Message, "").SetReplyTo(replyTo)
	return message
}

func newConfirmation(form *EmailForm) (*mail.SGMailV3, error) {
	from := mail.NewEmail(env.NoReplyName, env.NoReplyEmail)
	to := mail.NewEmail(form.Name, form.Email)
	replyTo := mail.NewEmail(env.RecipientName, env.RecipientEmail)

	buf := &bytes.Buffer{}
	err := confirmationTempl.Execute(buf, map[string]interface{}{
		"FormName":       form.Name,
		"FormEmail":      form.Email,
		"FormSubject":    form.Subject,
		"FormMessage":    form.Message,
		"NoReplyName":    env.NoReplyName,
		"NoReplyEmail":   env.NoReplyEmail,
		"RecipientName":  env.RecipientName,
		"RecipientEmail": env.RecipientEmail,
	})
	if err != nil {
		return nil, err
	}

	message := mail.NewSingleEmail(from, form.Subject, to, buf.String(), "").SetReplyTo(replyTo)
	return message, nil
}
