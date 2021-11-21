package sail

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
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
	once    utils.TryOnce
	service *sailService
)

type sailService struct {
	sendClient           utils.SendGridClient
	reCaptcha            utils.ReCaptcha
	formDecoder          *schema.Decoder
	confirmationTemplate *template.Template
}

var SendEmailHandler = utils.ApplyMiddlewares(
	sendEmailHandler,
	utils.CORSMiddleware,
	utils.LogEntryAndRecoverMiddleware,
).ServeHTTP

func sendEmailHandler(writer http.ResponseWriter, request *http.Request) {
	Init(config.ParseFromOSEnv)
	service.ServeHTTP(writer, request)
}

func Init(parseFunc func(*config.Environ) error) {
	err := once.TryDo(func() error {
		log.SetHandler(json.Default)
		log.SetLevel(log.InfoLevel)

		err := config.ParseEnv(parseFunc)
		if err != nil {
			return fmt.Errorf("init error: %w", err)
		}

		service, err = newDefaultService()
		if err != nil {
			return fmt.Errorf("init error: %w", err)
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
}

func newDefaultService() (*sailService, error) {
	sendClient := sendgrid.NewSendClient(config.Env.SendGridApiKey)

	var reCaptcha utils.ReCaptcha
	if config.Env.ShouldVerifyReCaptcha() {
		// sendErr can only occur if secret key is empty
		re, _ := recaptcha.NewReCAPTCHA(config.Env.ReCaptchaSecretKey, config.Env.GetReCaptchaVersion(), reCaptchaTimeout)
		reCaptcha = &re
	}

	formDecoder := schema.NewDecoder()
	formDecoder.IgnoreUnknownKeys(true)

	confirmationTemplate, err := template.New("confirmation").
		Option("missingkey=error").
		Parse(config.Env.ConfirmationTemplate)
	if err != nil {
		return nil, err
	}

	return &sailService{
		sendClient:           sendClient,
		reCaptcha:            reCaptcha,
		formDecoder:          formDecoder,
		confirmationTemplate: confirmationTemplate,
	}, nil
}

func (service *sailService) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	logEntry := request.Context().Value(utils.LogEntryCtxKey).(log.Interface)

	if err := service.sendEmailAndConfirmation(request); err != nil {
		logEntry.WithError(err).WithField("form", request.Form).Error("Sending email failed")

		http.Redirect(writer, request, config.Env.ErrorPage, http.StatusSeeOther)
		return
	}

	logEntry.WithField("form", request.Form).Info("Email successfully sent")

	http.Redirect(writer, request, config.Env.ThankYouPage, http.StatusSeeOther)
}

func (service *sailService) sendEmailAndConfirmation(request *http.Request) error {
	form, err := service.parseForm(request)
	if err != nil {
		return sendErr{"invalid form", err}
	}

	err = service.checkHoneypot(form)
	if err != nil {
		return sendErr{"honeypot check failed", err}
	}

	err = service.verifyReCaptcha(form.ReCaptcha, request.RemoteAddr)
	if err != nil {
		return sendErr{"recaptcha failed", err}
	}

	message := service.newMessage(form)
	err = utils.Retry(tries, retryBackOff, service.sendMessageFunc(message))
	if err != nil {
		return sendErr{"sending email failed", err}
	}

	confirmation, err := service.newConfirmation(form)
	if err != nil {
		return sendErr{"template error", err}
	}

	err = utils.Retry(tries, retryBackOff, service.sendMessageFunc(confirmation))
	if err != nil {
		return sendErr{"sending confirmation failed", err}
	}

	return nil
}

func (service *sailService) checkHoneypot(form *emailForm) error {
	if form.Honeypot != "" {
		return fmt.Errorf("invalid '%s' value '%s'", config.Env.HoneypotField, form.Honeypot)
	}
	return nil
}

func (service *sailService) verifyReCaptcha(challenge, remoteAddr string) error {
	if service.reCaptcha == nil {
		return nil
	}
	var err error
	_ = utils.Retry(tries, retryBackOff, func() error {
		err = service.reCaptcha.VerifyWithOptions(challenge, recaptcha.VerifyOption{
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

func (service *sailService) sendMessageFunc(message *mail.SGMailV3) func() error {
	return func() error {
		resp, err := service.sendClient.Send(message)
		if err != nil {
			return err
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("response code '%d' not ok, body '%s'", resp.StatusCode, resp.Body)
		}
		return nil
	}
}

func (service *sailService) newMessage(form *emailForm) *mail.SGMailV3 {
	from := mail.NewEmail(config.Env.NoReplyName, config.Env.NoReplyEmail)
	to := mail.NewEmail(config.Env.RecipientName, config.Env.RecipientEmail)
	replyTo := mail.NewEmail(form.Name, form.Email)
	contentType := getContentType([]byte(form.Message))

	message := mail.NewSingleEmail(from, form.Subject, to, "", "")
	message.AddContent(mail.NewContent(contentType, form.Message))
	message.SetReplyTo(replyTo)

	return message
}

func (service *sailService) newConfirmation(form *emailForm) (*mail.SGMailV3, error) {
	from := mail.NewEmail(config.Env.NoReplyName, config.Env.NoReplyEmail)
	to := mail.NewEmail(form.Name, form.Email)
	replyTo := mail.NewEmail(config.Env.RecipientName, config.Env.RecipientEmail)

	buf := &bytes.Buffer{}
	err := service.confirmationTemplate.Execute(buf, map[string]interface{}{
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
