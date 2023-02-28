package sail

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/apex/log"
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

//go:embed templates
var templatesFS embed.FS

var (
	once    sync.Once
	service *sailService
)

type sailService struct {
	env *config.Environ

	emailClient SendGridClient
	reCaptcha   ReCaptchaClient

	formDecoder *schema.Decoder
	templates   *template.Template
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
	var initErr error
	once.Do(func() {
		log.SetHandler(utils.DefaultJSONLogHandler)
		log.SetLevel(log.InfoLevel)

		env, err := config.ParseEnv(parseFunc)
		if err != nil {
			initErr = err
			return
		}

		service, initErr = newSailService(env)
	})
	if initErr != nil {
		once = sync.Once{}
		panic(initErr)
	}
}

func newSailService(env *config.Environ) (*sailService, error) {
	sendClient := sendgrid.NewSendClient(env.SendGridApiKey)

	var reCaptcha ReCaptchaClient
	if env.ReCaptchaEnabled() {
		// Error can only occur if secret key is empty
		re, _ := recaptcha.NewReCAPTCHA(env.ReCaptchaSecretKey, env.GetReCaptchaVersion(), reCaptchaTimeout)
		reCaptcha = &re
	}

	formDecoder := schema.NewDecoder()
	formDecoder.IgnoreUnknownKeys(true)

	templates, err := template.ParseFS(templatesFS, "templates/*")
	if err != nil {
		return nil, err
	}

	return &sailService{
		env:         env,
		emailClient: sendClient,
		reCaptcha:   reCaptcha,
		formDecoder: formDecoder,
		templates:   templates.Option("missingkey=error"),
	}, nil
}

func (service *sailService) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	logEntry := request.Context().Value(utils.LogEntryCtxKey).(log.Interface)

	if err := service.sendEmailAndConfirmation(request); err != nil {
		logEntry.WithError(err).WithField("form", request.Form).Warn("Sending email failed")

		http.Redirect(writer, request, service.env.ErrorPage, http.StatusSeeOther)
		return
	}

	logEntry.WithField("form", request.Form).Info("Email sent successfully")

	http.Redirect(writer, request, service.env.SuccessPage, http.StatusSeeOther)
}

func (service *sailService) sendEmailAndConfirmation(request *http.Request) error {
	form, err := service.parseForm(request)
	if err != nil {
		return fmt.Errorf("invalid form: %w", err)
	}

	err = service.checkHoneypot(form)
	if err != nil {
		return fmt.Errorf("honeypot check failed: %w", err)
	}

	err = service.verifyReCaptcha(form.ReCaptchaChallenge, request.RemoteAddr)
	if err != nil {
		return fmt.Errorf("recaptcha failed: %w", err)
	}

	message, err := service.newEmail(form)
	if err != nil {
		return fmt.Errorf("creating email failed: %w", err)
	}
	err = service.sendEmail(message)
	if err != nil {
		return fmt.Errorf("sending email failed: %w", err)
	}

	confirmation, err := service.newConfirmation(form)
	if err != nil {
		return fmt.Errorf("creating confirmation failed: %w", err)
	}

	err = service.sendEmail(confirmation)
	if err != nil {
		return fmt.Errorf("sending confirmation failed: %w", err)
	}

	return nil
}

func (service *sailService) checkHoneypot(form *emailForm) error {
	if !service.env.HoneypotCheckEnabled() {
		return nil
	}
	if form.HoneypotValue != "" {
		return fmt.Errorf("invalid '%s' value '%s'", service.env.HoneypotField, form.HoneypotValue)
	}
	return nil
}

func (service *sailService) verifyReCaptcha(challenge, remoteAddr string) error {
	if !service.env.ReCaptchaEnabled() {
		return nil
	}
	if challenge == "" {
		return errors.New("recaptcha challenge is empty")
	}
	var err error
	_ = utils.Retry(tries, retryBackOff, func() error {
		err = service.reCaptcha.VerifyWithOptions(challenge, recaptcha.VerifyOption{
			RemoteIP: remoteAddr,
			// Threshold is only used for recaptcha v3
			// and using a zero value defaults to 0.5.
			Threshold: float32(service.env.ReCaptchaV3Threshold),
		})
		var rcErr *recaptcha.Error
		if errors.As(err, &rcErr) && rcErr.RequestError {
			return err
		}
		return nil
	})
	return err
}

func (service *sailService) sendEmail(message *mail.SGMailV3) error {
	return utils.Retry(tries, retryBackOff, func() error {
		resp, err := service.emailClient.Send(message)
		if err != nil {
			return err
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("response code '%d' not ok, body '%s'", resp.StatusCode, resp.Body)
		}
		return nil
	})
}

func (service *sailService) newEmail(form *emailForm) (*mail.SGMailV3, error) {
	from := mail.NewEmail(service.env.NoReplyName, service.env.NoReplyEmail)
	to := mail.NewEmail(service.env.RecipientName, service.env.RecipientEmail)
	replyTo := mail.NewEmail(form.Name, form.Email)

	body, err := service.createBodyFromTemplate(service.env.EmailTemplateFile, form)
	if err != nil {
		return nil, err
	}

	contentType := getContentType(body)

	email := mail.NewSingleEmail(from, form.Subject, to, "", "")
	email.AddContent(mail.NewContent(contentType, string(body)))
	email.SetReplyTo(replyTo)

	return email, nil
}

func (service *sailService) newConfirmation(form *emailForm) (*mail.SGMailV3, error) {
	from := mail.NewEmail(service.env.NoReplyName, service.env.NoReplyEmail)
	to := mail.NewEmail(form.Name, form.Email)
	replyTo := mail.NewEmail(service.env.RecipientName, service.env.RecipientEmail)

	body, err := service.createBodyFromTemplate(service.env.ConfirmationTemplateFile, form)
	if err != nil {
		return nil, err
	}

	contentType := getContentType(body)

	email := mail.NewSingleEmail(from, form.Subject, to, "", "")
	email.AddContent(mail.NewContent(contentType, string(body)))
	email.SetReplyTo(replyTo)

	return email, nil
}

func (service *sailService) createBodyFromTemplate(name string, form *emailForm) ([]byte, error) {
	buf := &bytes.Buffer{}
	err := service.templates.ExecuteTemplate(buf, name, map[string]any{
		"FORM_NAME":       form.Name,
		"FORM_EMAIL":      form.Email,
		"FORM_SUBJECT":    form.Subject,
		"FORM_MESSAGE":    form.Message,
		"NOREPLY_NAME":    service.env.NoReplyName,
		"NOREPLY_EMAIL":   service.env.NoReplyEmail,
		"RECIPIENT_NAME":  service.env.RecipientName,
		"RECIPIENT_EMAIL": service.env.RecipientEmail,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func getContentType(data []byte) string {
	contentType := http.DetectContentType(data)
	if strings.HasPrefix(contentType, "text/html") {
		return "text/html"
	}
	return "text/plain"
}
