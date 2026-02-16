package sdkclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mailerlite/mailerlite-go"
)

// CLIError represents an API error with full field-level validation details.
type CLIError struct {
	StatusCode int
	Message    string              `json:"message"`
	Errors     map[string][]string `json:"errors,omitempty"`
	RawBody    json.RawMessage     `json:"-"`
}

func (e *CLIError) Error() string {
	if len(e.Errors) > 0 {
		maxLen := 0
		for field := range e.Errors {
			if len(field) > maxLen {
				maxLen = len(field)
			}
		}

		var b strings.Builder
		fmt.Fprintf(&b, "API error %d: %s\n", e.StatusCode, e.Message)
		for field, msgs := range e.Errors {
			for _, msg := range msgs {
				fmt.Fprintf(&b, "\n  %-*s  %s", maxLen, field, msg)
			}
		}
		return b.String()
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// WrapError converts SDK errors into CLIError with full field-level details.
// It uses the transport's captured response body to extract validation errors
// that the SDK discards.
func WrapError(transport *CLITransport, err error) error {
	if err == nil {
		return nil
	}

	cliErr := &CLIError{}

	// Try to extract status code and message from SDK error types.
	var errResp *mailerlite.ErrorResponse
	var authErr *mailerlite.AuthError
	var rateLimitErr *mailerlite.RateLimitError
	switch {
	case errors.As(err, &authErr):
		if authErr.Response != nil {
			cliErr.StatusCode = authErr.Response.StatusCode
		}
		cliErr.Message = authErr.Message
	case errors.As(err, &rateLimitErr):
		if rateLimitErr.Response != nil {
			cliErr.StatusCode = rateLimitErr.Response.StatusCode
		}
		cliErr.Message = rateLimitErr.Message
	case errors.As(err, &errResp):
		if errResp.Response != nil {
			cliErr.StatusCode = errResp.Response.StatusCode
		}
		cliErr.Message = errResp.Message
		if len(errResp.Errors) > 0 {
			cliErr.Errors = errResp.Errors
		}
	default:
		// Not an SDK API error — return as-is.
		return err
	}

	// Try to parse the captured response body for field-level errors.
	rawBody := transport.LastErrorBody()
	if len(rawBody) > 0 {
		cliErr.RawBody = json.RawMessage(rawBody)

		var parsed struct {
			Message string              `json:"message"`
			Errors  map[string][]string `json:"errors"`
		}
		if json.Unmarshal(rawBody, &parsed) == nil {
			if parsed.Message != "" {
				cliErr.Message = parsed.Message
			}
			if len(parsed.Errors) > 0 {
				cliErr.Errors = parsed.Errors
			}
		}
	}

	if cliErr.Message == "" {
		cliErr.Message = err.Error()
	}

	return cliErr
}
