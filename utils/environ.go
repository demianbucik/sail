package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	"gopkg.in/ezzarghili/recaptcha-go.v4"
	"gopkg.in/yaml.v2"
)

const (
	reCaptchaV2 = "v2"
	reCaptchaV3 = "v3"
)

type Environ struct {
	envRequired   `yaml:",inline"`
	envReCaptcha  `yaml:",inline"`
	HoneypotField string `yaml:"HONEYPOT_FIELD"`
}

type envRequired struct {
	SendGridApiKey       string `yaml:"SENDGRID_API_KEY"`
	NoReplyEmail         string `yaml:"NOREPLY_EMAIL"`
	NoReplyName          string `yaml:"NOREPLY_NAME"`
	RecipientEmail       string `yaml:"RECIPIENT_EMAIL"`
	RecipientName        string `yaml:"RECIPIENT_NAME"`
	ThankYouPage         string `yaml:"THANK_YOU_PAGE"`
	ErrorPage            string `yaml:"ERROR_PAGE"`
	ConfirmationTemplate string `yaml:"CONFIRMATION_TEMPLATE"`
}

type envReCaptcha struct {
	ReCaptchaSecretKey string `yaml:"RECAPTCHA_SECRET_KEY"`
	ReCaptchaVersion   string `yaml:"RECAPTCHA_VERSION"`
}

func (env envReCaptcha) ShouldVerifyReCaptcha() bool {
	return env.ReCaptchaVersion != ""
}

func (env envReCaptcha) ParseReCaptchaVersion() recaptcha.VERSION {
	if env.ReCaptchaVersion == reCaptchaV2 {
		return recaptcha.V2
	}
	return recaptcha.V3
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
	env.envRequired = envRequired{
		SendGridApiKey:       os.Getenv("SENDGRID_API_KEY"),
		NoReplyEmail:         os.Getenv("NOREPLY_EMAIL"),
		NoReplyName:          os.Getenv("NOREPLY_NAME"),
		RecipientEmail:       os.Getenv("RECIPIENT_EMAIL"),
		RecipientName:        os.Getenv("RECIPIENT_NAME"),
		ThankYouPage:         os.Getenv("THANK_YOU_PAGE"),
		ErrorPage:            os.Getenv("ERROR_PAGE"),
		ConfirmationTemplate: os.Getenv("CONFIRMATION_TEMPLATE"),
	}
	env.envReCaptcha = envReCaptcha{
		ReCaptchaSecretKey: os.Getenv("RECAPTCHA_SECRET_KEY"),
		ReCaptchaVersion:   os.Getenv("RECAPTCHA_VERSION"),
	}
	return nil
}

// For local development
func GetParseFromYAMLFunc(filePath string) func(*Environ) error {
	return func(env *Environ) error {
		fileBytes, err := ioutil.ReadFile(filePath)
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
	if err := verifyNonEmpty(&env.envRequired); err != nil {
		return err
	}

	if env.ShouldVerifyReCaptcha() {
		if err := verifyNonEmpty(&env.envReCaptcha); err != nil {
			return err
		}
		if env.ReCaptchaVersion != reCaptchaV2 && env.ReCaptchaVersion != reCaptchaV3 {
			return fmt.Errorf("invalid recaptcha version '%s', use 'v2', 'v3', or '' to turn it off", env.ReCaptchaVersion)
		}
	}

	return nil
}

func verifyNonEmpty(envStruct interface{}) error {
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

		val := structVal.Field(i).Interface().(string)
		if val == "" {
			return fmt.Errorf(
				"environment field '%s' should not be empty",
				structType.Field(i).Tag.Get("yaml"),
			)
		}
	}

	return nil
}
