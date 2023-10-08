package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ReCaptchaURL = "https://www.google.com/recaptcha/api/siteverify"
	TurnstileURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
)

type RecaptchaVersion string

const (
	ReCaptchaV2 RecaptchaVersion = "v2"
	ReCaptchaV3 RecaptchaVersion = "v3"
	ReCaptchaCf RecaptchaVersion = "cf"
)

type ReCaptcha struct {
	Client  http.Client
	Secret  string
	Version RecaptchaVersion
}

type VerifyOptions struct {
	RemoteIp       string
	Action         string
	ScoreThreshold float64
}

type VerifyError struct {
	IsHttpError bool
	Err         error
}

func (err VerifyError) Error() string {
	return err.Err.Error()
}

type SiteVerifyResponse struct {
	Success     bool      `json:"success"`
	Score       float64   `json:"score"`
	Action      string    `json:"action"`
	ChallengeTS time.Time `json:"challenge_ts"`
	Hostname    string    `json:"hostname"`
	ErrorCodes  []string  `json:"error-codes"`
}

// Verify the provided reCaptcha token depending on version.
func (c *ReCaptcha) Verify(response string, opts VerifyOptions) error {
	serviceUrl := ReCaptchaURL
	if c.Version == ReCaptchaCf {
		serviceUrl = TurnstileURL
	}

	query := make(url.Values)
	query.Add("secret", c.Secret)
	query.Add("response", response)
	if opts.RemoteIp != "" {
		query.Add("remoteip", opts.RemoteIp)
	}

	resp, err := c.Client.PostForm(serviceUrl, query)
	if err != nil {
		return VerifyError{IsHttpError: true, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return VerifyError{IsHttpError: true, Err: fmt.Errorf("status code '%d' not ok", resp.StatusCode)}
	}

	var body SiteVerifyResponse
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return VerifyError{IsHttpError: true, Err: err}
	}

	if !body.Success {
		return VerifyError{Err: errors.New(strings.Join(body.ErrorCodes, ", "))}
	}

	if c.Version == ReCaptchaV3 {
		if body.Score < opts.ScoreThreshold {
			return VerifyError{Err: fmt.Errorf("score '%.3f' is below '%.3f'", body.Score, opts.ScoreThreshold)}
		}
	}

	return nil
}
