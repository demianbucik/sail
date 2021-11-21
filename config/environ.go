package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"

	"gopkg.in/ezzarghili/recaptcha-go.v4"
	"gopkg.in/yaml.v2"
)

const (
	reCaptchaV2 = "v2"
	reCaptchaV3 = "v3"
)

type Environ struct {
	envRequired  `yaml:",inline"`
	envReCaptcha `yaml:",inline"`
	// Optional fields
	HoneypotField string `yaml:"HONEYPOT_FIELD"`
	// GCP requires string values inside environment variables YAML file, using for example 0.25 fails.
	// The YAML parser can only decode "0.25" as a string. So we need to use 2 variables here.
	ReCaptchaV3ThresholdStr string `yaml:"RECAPTCHA_V3_THRESHOLD"`
	ReCaptchaV3Threshold    float32
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

func (env envReCaptcha) GetReCaptchaVersion() recaptcha.VERSION {
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
	env.SendGridApiKey = os.Getenv("SENDGRID_API_KEY")
	env.NoReplyEmail = os.Getenv("NOREPLY_EMAIL")
	env.NoReplyName = os.Getenv("NOREPLY_NAME")
	env.RecipientEmail = os.Getenv("RECIPIENT_EMAIL")
	env.RecipientName = os.Getenv("RECIPIENT_NAME")
	env.ThankYouPage = os.Getenv("THANK_YOU_PAGE")
	env.ErrorPage = os.Getenv("ERROR_PAGE")
	env.ConfirmationTemplate = os.Getenv("CONFIRMATION_TEMPLATE")
	env.ReCaptchaSecretKey = os.Getenv("RECAPTCHA_SECRET_KEY")
	env.ReCaptchaVersion = os.Getenv("RECAPTCHA_VERSION")
	thresholdStr := os.Getenv("RECAPTCHA_V3_THRESHOLD")
	if thresholdStr != "" {
		threshold, err := strconv.ParseFloat(thresholdStr, 32)
		if err != nil {
			return err
		}
		env.ReCaptchaV3Threshold = float32(threshold)
	}

	return nil
}

// GetParseFromYAMLFunc can be used for local development.
func GetParseFromYAMLFunc(filePath string) func(*Environ) error {
	return func(env *Environ) error {
		fileBytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(fileBytes, env); err != nil {
			return err
		}
		if env.ReCaptchaV3ThresholdStr != "" {
			threshold, err := strconv.ParseFloat(env.ReCaptchaV3ThresholdStr, 32)
			if err != nil {
				return err
			}
			env.ReCaptchaV3Threshold = float32(threshold)
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
			return fmt.Errorf(
				"invalid recaptcha version '%s', use 'v2', 'v3', or '' to turn it off",
				env.ReCaptchaVersion,
			)
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

		if structVal.Field(i).IsZero() {
			return fmt.Errorf(
				"environment variable '%s' should not be empty",
				structType.Field(i).Tag.Get("yaml"),
			)
		}
	}

	return nil
}
