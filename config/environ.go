package config

import (
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"

	"github.com/demianbucik/sail/utils"
)

type Environ struct {
	envRequired  `yaml:",inline"`
	envReCaptcha `yaml:",inline"`
	// Optional fields
	HoneypotField string `yaml:"HONEYPOT_FIELD"`
}

func (env Environ) HoneypotCheckEnabled() bool {
	return env.HoneypotField != ""
}

type envRequired struct {
	SendGridApiKey           string `yaml:"SENDGRID_API_KEY"`
	NoReplyEmail             string `yaml:"NOREPLY_EMAIL"`
	NoReplyName              string `yaml:"NOREPLY_NAME"`
	RecipientEmail           string `yaml:"RECIPIENT_EMAIL"`
	RecipientName            string `yaml:"RECIPIENT_NAME"`
	SuccessPage              string `yaml:"SUCCESS_PAGE"`
	ErrorPage                string `yaml:"ERROR_PAGE"`
	EmailTemplateFile        string `yaml:"EMAIL_TEMPLATE_FILE"`
	ConfirmationTemplateFile string `yaml:"CONFIRMATION_TEMPLATE_FILE"`
}

type envReCaptcha struct {
	ReCaptchaVersion     utils.RecaptchaVersion `yaml:"RECAPTCHA_VERSION"`
	ReCaptchaSecretKey   string                 `yaml:"RECAPTCHA_SECRET_KEY"`
	ReCaptchaV3Threshold floatAsStr             `yaml:"RECAPTCHA_V3_THRESHOLD"`
}

func (env envReCaptcha) ReCaptchaEnabled() bool {
	return env.ReCaptchaVersion != "" && env.ReCaptchaVersion != "off"
}

func ParseEnv(parseFunc func(*Environ) error) (*Environ, error) {
	env := &Environ{}
	if err := parseFunc(env); err != nil {
		return nil, err
	}
	if err := validate(env); err != nil {
		return nil, err
	}
	return env, nil
}

func ParseFromOSEnv(env *Environ) error {
	env.HoneypotField = os.Getenv("HONEYPOT_FIELD")
	env.SendGridApiKey = os.Getenv("SENDGRID_API_KEY")
	env.NoReplyEmail = os.Getenv("NOREPLY_EMAIL")
	env.NoReplyName = os.Getenv("NOREPLY_NAME")
	env.RecipientEmail = os.Getenv("RECIPIENT_EMAIL")
	env.RecipientName = os.Getenv("RECIPIENT_NAME")
	env.SuccessPage = os.Getenv("SUCCESS_PAGE")
	env.ErrorPage = os.Getenv("ERROR_PAGE")
	env.EmailTemplateFile = os.Getenv("EMAIL_TEMPLATE_FILE")
	env.ConfirmationTemplateFile = os.Getenv("CONFIRMATION_TEMPLATE_FILE")
	env.ReCaptchaSecretKey = os.Getenv("RECAPTCHA_SECRET_KEY")
	env.ReCaptchaVersion = utils.RecaptchaVersion(os.Getenv("RECAPTCHA_VERSION"))
	if err := env.ReCaptchaV3Threshold.UnmarshalText([]byte(os.Getenv("RECAPTCHA_V3_THRESHOLD"))); err != nil {
		return err
	}

	return nil
}

// GetParseFromYAMLFunc can be used for local development.
func GetParseFromYAMLFunc(filePath string) func(*Environ) error {
	return func(env *Environ) error {
		fileBytes, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(fileBytes, env); err != nil {
			return err
		}
		return nil
	}
}

func validate(env *Environ) error {
	if err := validateNonEmpty(&env.envRequired); err != nil {
		return err
	}
	if err := validateReCaptcha(&env.envReCaptcha); err != nil {
		return err
	}
	return nil
}

func validateNonEmpty(envStruct any) error {
	structVal := reflect.ValueOf(envStruct)
	structType := reflect.TypeOf(envStruct)
	if structVal.Kind() == reflect.Ptr {
		structVal = structVal.Elem()
		structType = structType.Elem()
	}

	for i := 0; i < structVal.NumField(); i++ {
		if structVal.Field(i).Kind() == reflect.Struct {
			continue
		}

		if structVal.Field(i).IsZero() {
			return fmt.Errorf(
				"%s value should not be empty",
				structType.Field(i).Tag.Get("yaml"),
			)
		}
	}

	return nil
}

func validateReCaptcha(env *envReCaptcha) error {
	if !env.ReCaptchaEnabled() {
		return nil
	}
	if env.ReCaptchaVersion != utils.ReCaptchaV2 && env.ReCaptchaVersion != utils.ReCaptchaV3 && env.ReCaptchaVersion != utils.ReCaptchaCf {
		return fmt.Errorf(
			"invalid RECAPTCHA_VERSION value '%s', valid options are 'v2', 'v3' and 'cf', to disable recaptcha use '' or 'off'",
			env.ReCaptchaVersion,
		)
	}
	if env.ReCaptchaV3Threshold < 0 || env.ReCaptchaV3Threshold > 1 {
		return fmt.Errorf("invalid RECAPTCHA_V3_THRESHOLD value '%v', use a value between 0 and 1", env.ReCaptchaV3Threshold)
	}
	if env.ReCaptchaSecretKey == "" {
		return fmt.Errorf("RECAPTCHA_SECRET_KEY value should not be empty")
	}

	return nil
}
