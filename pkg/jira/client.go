package jira

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Config is the struct that holds the Jira API configuration options
type Config struct {
	Username       string
	Password       string
	EndpointPrefix string
}

// Jira represents the base struct for using Jira API
type Jira struct {
	Config
	client *http.Client
}

// New creates Jira instance
func New(config Config, opts ...Option) (*Jira, error) {

	if len(config.Username) == 0 || len(config.Password) == 0 {
		return nil, errors.New("username and password are required")
	}

	a := Jira{
		Config: config,
		client: &http.Client{},
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&a)
		}
	}

	return &a, nil
}

// Option allows for custom configuration overrides.
type Option func(*Jira)

// WithTimeout allows for a custom timeout to be provided to the underlying
// HTTP client that's used to communicate with the API.
func WithTimeout(d time.Duration) func(*Jira) {
	return func(a *Jira) {
		a.client.Timeout = d
	}
}

// WithTransport allows customer HTTP transports to be provided to the client
func WithTransport(transport http.RoundTripper) func(*Jira) {
	return func(a *Jira) {
		a.client.Transport = transport
	}
}

// execute an http request againt Jira API
func (a *Jira) execute(ctx context.Context, req *http.Request) (*http.Response, error) {

	req = req.WithContext(ctx)

	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:93.0) Gecko/20100101 Firefox/93.0")
	req.Header.Add("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Content-Type", "application/json")

	req.SetBasicAuth(a.Username, a.Password)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, handleError(resp)
	}

	return resp, nil
}

// ErrorResponse represents a error/refusal response from Jira
type ErrorResponse struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

func (e ErrorResponse) Error() string {
	if len(e.ErrorMessages) > 0 {
		return fmt.Sprintf("ErrorMessages: %v", e.ErrorMessages)
	}
	if len(e.Errors) > 0 {
		return fmt.Sprintf("Errors: %v", e.Errors)
	}
	return "Unknown error"
}

// handleError handles 4xx and 5xx responses and transform them to ErrorResponse struct
func handleError(r *http.Response) error {
	var er ErrorResponse
	if err := json.NewDecoder(r.Body).Decode(&er); err != nil {
		return err
	}
	return er
}
