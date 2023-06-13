package sail

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/gorilla/schema"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/demianbucik/sail/config"
	"github.com/demianbucik/sail/utils"
)

const (
	retries          = 2
	retryBackOff     = 10 * time.Millisecond
	reCaptchaTimeout = 2 * time.Second
)

//go:embed templates
var templatesFS embed.FS

var (
	once    sync.Once
	service *sailService
)

var SendEmailHandler = utils.MiddlewareWrap(
	initAndServe,
	utils.CORSMiddleware,
	utils.LogAndRecoverMiddleware,
).ServeHTTP

func initAndServe(writer http.ResponseWriter, request *http.Request) {
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

type sailService struct {
	env *config.Environ

	emailClient     SendGridClient
	reCaptchaClient ReCaptchaClient

	formDecoder *schema.Decoder
	templates   *template.Template
}

func newSailService(env *config.Environ) (*sailService, error) {
	emailClient := sendgrid.NewSendClient(env.SendGridApiKey)

	reCaptchaClient := &utils.ReCaptcha{
		Client:  http.Client{Timeout: reCaptchaTimeout},
		Secret:  env.ReCaptchaSecretKey,
		Version: env.ReCaptchaVersion,
	}

	formDecoder := schema.NewDecoder()
	formDecoder.IgnoreUnknownKeys(true)

	templates, err := template.ParseFS(templatesFS, "templates/*")
	if err != nil {
		return nil, err
	}

	return &sailService{
		env:             env,
		emailClient:     emailClient,
		reCaptchaClient: reCaptchaClient,
		formDecoder:     formDecoder,
		templates:       templates.Option("missingkey=error"),
	}, nil
}

func (service *sailService) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	reqCtx := request.Context().Value(utils.RequestCtxKey).(*utils.RequestContext)

	form, err := service.parseForm(request)
	if err != nil {
		reqCtx.RequestLog.Finalize()
		reqCtx.LogEntry.WithError(err).WithField("httpForm", request.Form).Info("Email rejected - invalid form")

		http.Redirect(writer, request, service.env.ErrorPage, http.StatusSeeOther)
		return
	}

	reqCtx.LogEntry = reqCtx.LogEntry.WithField("emailForm", form)

	clientIp := reqCtx.RequestLog.RemoteIp
	if err = service.verify(form, clientIp); err != nil {
		reqCtx.RequestLog.Finalize()
		reqCtx.LogEntry.WithError(err).Info("Email rejected - verification")

		http.Redirect(writer, request, service.env.ErrorPage, http.StatusSeeOther)
		return
	}

	if err = service.sendEmailAndConfirmation(form); err != nil {
		reqCtx.RequestLog.Finalize()
		reqCtx.LogEntry.WithError(err).Warn("Sending email failed")

		http.Redirect(writer, request, service.env.ErrorPage, http.StatusSeeOther)
		return
	}

	reqCtx.RequestLog.Finalize()
	reqCtx.LogEntry.Info("Email sent successfully")

	http.Redirect(writer, request, service.env.SuccessPage, http.StatusSeeOther)
}

func (service *sailService) verify(form *EmailForm, clientIp string) error {
	if err := service.verifyReCaptcha(form.ReCaptchaResponse, clientIp); err != nil {
		return fmt.Errorf("recaptcha verification failed: %w", err)
	}
	if err := service.checkHoneypot(form); err != nil {
		return fmt.Errorf("honeypot check failed: %w", err)
	}
	return nil
}

func (service *sailService) sendEmailAndConfirmation(form *EmailForm) error {
	message, err := service.newEmail(form)
	if err != nil {
		return fmt.Errorf("creating email failed: %w", err)
	}
	if err = service.sendEmail(message); err != nil {
		return fmt.Errorf("sending email failed: %w", err)
	}

	confirmation, err := service.newConfirmation(form)
	if err != nil {
		return fmt.Errorf("creating confirmation failed: %w", err)
	}
	if err = service.sendEmail(confirmation); err != nil {
		return fmt.Errorf("sending confirmation failed: %w", err)
	}

	return nil
}

func (service *sailService) checkHoneypot(form *EmailForm) error {
	if !service.env.HoneypotCheckEnabled() {
		return nil
	}
	if form.HoneypotValue != "" {
		return fmt.Errorf("invalid '%s' value '%s'", service.env.HoneypotField, form.HoneypotValue)
	}
	return nil
}

func (service *sailService) verifyReCaptcha(response, clientIp string) error {
	if !service.env.ReCaptchaEnabled() {
		return nil
	}
	if response == "" {
		return errors.New("recaptcha response is empty")
	}
	var err error
	utils.Retry(retries, retryBackOff, func() error {
		err = service.reCaptchaClient.Verify(response, utils.VerifyOptions{
			RemoteIp:       clientIp,
			ScoreThreshold: float64(service.env.ReCaptchaV3Threshold),
		})
		if v, ok := err.(utils.VerifyError); ok && v.IsHttpError {
			return err
		}
		return nil
	})
	return err
}

func (service *sailService) sendEmail(message *mail.SGMailV3) error {
	return utils.Retry(retries, retryBackOff, func() error {
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

func (service *sailService) newEmail(form *EmailForm) (*mail.SGMailV3, error) {
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

func (service *sailService) newConfirmation(form *EmailForm) (*mail.SGMailV3, error) {
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

func (service *sailService) createBodyFromTemplate(name string, form *EmailForm) ([]byte, error) {
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
