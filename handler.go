package sail

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/json"
	"github.com/gorilla/schema"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gopkg.in/ezzarghili/recaptcha-go.v4"

	"github.com/demianbucik/sail/config"
	"github.com/demianbucik/sail/utils"
)

const (
	tries            = 3
	retryBackOff     = 10 * time.Millisecond
	reCaptchaTimeout = 2 * time.Second
)

var (
	once utils.TryOnce

	formDecoder          *schema.Decoder
	confirmationTemplate *template.Template
)

func Init(parseFunc func(*config.Environ) error) {
	err := once.TryDo(func() error {
		log.SetHandler(json.Default)
		log.SetLevel(log.InfoLevel)

		err := config.ParseEnv(parseFunc)
		if err != nil {
			return fmt.Errorf("init error: %w", err)
		}

		formDecoder = schema.NewDecoder()
		formDecoder.IgnoreUnknownKeys(true)

		confirmationTemplate, err = template.New("confirmation").
			Option("missingkey=error").
			Parse(config.Env.ConfirmationTemplate)
		if err != nil {
			return fmt.Errorf("init error: %w", err)
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
}

var SendEmailHandler = utils.ApplyMiddlewares(
	sendEmailHandler,
	utils.CORSMiddleware,
).ServeHTTP

func sendEmailHandler(writer http.ResponseWriter, request *http.Request) {
	logEntry := log.WithField("userAgent", request.UserAgent()).
		WithField("remoteAddr", request.RemoteAddr).
		WithField("url", request.RequestURI).
		WithField("headers", request.Header)

	defer func() {
		if rvr := recover(); rvr != nil {
			logEntry.WithField("panic", rvr).
				WithField("stackTrace", string(debug.Stack())).
				Error("Handler panicked")

			http.Error(writer, "internal server error", http.StatusInternalServerError)
		}
	}()

	Init(config.ParseFromOSEnv)

	client, reCaptcha := newClients()
	if err := sendEmailAndConfirmation(client, reCaptcha, request); err != nil {
		logEntry.WithError(err).WithField("form", request.Form).Error("Sending email failed")

		http.Redirect(writer, request, config.Env.ErrorPage, http.StatusSeeOther)
		return
	}

	logEntry.WithField("form", request.Form).Info("Email successfully sent")

	http.Redirect(writer, request, config.Env.ThankYouPage, http.StatusSeeOther)
}

var newClients = newClientsImpl

func newClientsImpl() (utils.SendGridClient, utils.ReCaptcha) {
	client := sendgrid.NewSendClient(config.Env.SendGridApiKey)
	if !config.Env.ShouldVerifyReCaptcha() {
		return client, nil
	}
	// Error can only occur if secret key is empty
	reCaptcha, _ := recaptcha.NewReCAPTCHA(config.Env.ReCaptchaSecretKey, config.Env.GetReCaptchaVersion(), reCaptchaTimeout)
	return client, &reCaptcha
}

func sendEmailAndConfirmation(client utils.SendGridClient, reCaptcha utils.ReCaptcha, request *http.Request) error {
	form, err := parseForm(request)
	if err != nil {
		return Error{"invalid form", err}
	}

	err = checkHoneypot(form)
	if err != nil {
		return Error{"honeypot check failed", err}
	}

	err = verifyReCaptcha(reCaptcha, form.ReCaptcha, request.RemoteAddr)
	if err != nil {
		return Error{"recaptcha failed", err}
	}

	message := newMessage(form)
	err = utils.Retry(tries, retryBackOff, sendMessageFunc(client, message))
	if err != nil {
		return Error{"sending email failed", err}
	}

	confirmation, err := newConfirmation(form)
	if err != nil {
		return Error{"template error", err}
	}

	err = utils.Retry(tries, retryBackOff, sendMessageFunc(client, confirmation))
	if err != nil {
		return Error{"sending confirmation failed", err}
	}

	return nil
}

func checkHoneypot(form *EmailForm) error {
	if form.Honeypot != "" {
		return fmt.Errorf("invalid '%s' value '%s'", config.Env.HoneypotField, form.Honeypot)
	}
	return nil
}

func verifyReCaptcha(reCaptcha utils.ReCaptcha, challenge, remoteAddr string) error {
	if reCaptcha == nil {
		return nil
	}
	var err error
	_ = utils.Retry(tries, retryBackOff, func() error {
		err = reCaptcha.VerifyWithOptions(challenge, recaptcha.VerifyOption{
			RemoteIP: remoteAddr,
			// Threshold is only used for recaptcha v3
			// and using a zero value defaults to 0.5.
			Threshold: config.Env.ReCaptchaV3Threshold,
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

func sendMessageFunc(client utils.SendGridClient, message *mail.SGMailV3) func() error {
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
	from := mail.NewEmail(config.Env.NoReplyName, config.Env.NoReplyEmail)
	to := mail.NewEmail(config.Env.RecipientName, config.Env.RecipientEmail)
	replyTo := mail.NewEmail(form.Name, form.Email)
	contentType := getContentType([]byte(form.Message))

	message := mail.NewSingleEmail(from, form.Subject, to, "", "")
	message.AddContent(mail.NewContent(contentType, form.Message))
	message.SetReplyTo(replyTo)

	return message
}

func newConfirmation(form *EmailForm) (*mail.SGMailV3, error) {
	from := mail.NewEmail(config.Env.NoReplyName, config.Env.NoReplyEmail)
	to := mail.NewEmail(form.Name, form.Email)
	replyTo := mail.NewEmail(config.Env.RecipientName, config.Env.RecipientEmail)

	buf := &bytes.Buffer{}
	err := confirmationTemplate.Execute(buf, map[string]interface{}{
		"FORM_NAME":       form.Name,
		"FORM_EMAIL":      form.Email,
		"FORM_SUBJECT":    form.Subject,
		"FORM_MESSAGE":    form.Message,
		"NOREPLY_NAME":    config.Env.NoReplyName,
		"NOREPLY_EMAIL":   config.Env.NoReplyEmail,
		"RECIPIENT_NAME":  config.Env.RecipientName,
		"RECIPIENT_EMAIL": config.Env.RecipientEmail,
	})
	if err != nil {
		return nil, err
	}

	contentType := getContentType(buf.Bytes())

	message := mail.NewSingleEmail(from, form.Subject, to, "", "")
	message.AddContent(mail.NewContent(contentType, buf.String()))
	message.SetReplyTo(replyTo)

	return message, nil
}

func getContentType(data []byte) string {
	contentType := http.DetectContentType(data)
	if strings.HasPrefix(contentType, "text/html") {
		return "text/html"
	}
	return "text/plain"
}
