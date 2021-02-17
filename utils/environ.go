package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	"gopkg.in/yaml.v2"
)

type Environ struct {
	SendGridApiKey       string `yaml:"SENDGRID_API_KEY"`
	ReCaptchaSecretKey   string `yaml:"RECAPTCHA_SECRET_KEY"`
	ReCaptchaVersion     string `yaml:"RECAPTCHA_VERSION"`
	NoReplyEmail         string `yaml:"NOREPLY_EMAIL"`
	NoReplyName          string `yaml:"NOREPLY_NAME"`
	RecipientEmail       string `yaml:"RECIPIENT_EMAIL"`
	RecipientName        string `yaml:"RECIPIENT_NAME"`
	ThankYouPage         string `yaml:"THANK_YOU_PAGE"`
	ErrorPage            string `yaml:"ERROR_PAGE"`
	ConfirmationTemplate string `yaml:"CONFIRMATION_TEMPLATE"`
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
	env = &Environ{
		SendGridApiKey:       os.Getenv("SENDGRID_API_KEY"),
		ReCaptchaSecretKey:   os.Getenv("RECAPTCHA_SECRET_KEY"),
		NoReplyEmail:         os.Getenv("NOREPLY_EMAIL"),
		NoReplyName:          os.Getenv("NOREPLY_NAME"),
		RecipientEmail:       os.Getenv("RECIPIENT_EMAIL"),
		RecipientName:        os.Getenv("RECIPIENT_NAME"),
		ThankYouPage:         os.Getenv("THANK_YOU_PAGE"),
		ErrorPage:            os.Getenv("ERROR_PAGE"),
		ConfirmationTemplate: os.Getenv("CONFIRMATION_TEMPLATE"),
	}
	return nil
}

// For local development
func GetParseFromYAMLFunc(filename string) func(*Environ) error {
	return func(env *Environ) error {
		bytes, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}

		if err := yaml.Unmarshal(bytes, env); err != nil {
			return err
		}

		return nil
	}
}

func validate(env *Environ) error {
	structVal := reflect.ValueOf(env).Elem()
	for i := 0; i < structVal.NumField(); i++ {
		val := structVal.Field(i).Interface().(string)
		if val == "" {
			return fmt.Errorf(
				"environment field '%s' should not be empty",
				structVal.Type().Field(i).Name,
			)
		}
	}

	if env.ReCaptchaVersion != "v2" && env.ReCaptchaVersion != "v3" {
		return fmt.Errorf("invalid recaptcha version '%s', use 'v2' or 'v3'", env.ReCaptchaVersion)
	}

	return nil
}
