package sdkclient

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL = "https://connect.mailerlite.com/api"
	maxRetries     = 3
)

var userAgent = "mailerlite-cli/dev"

// SetUserAgent sets the User-Agent string used for all API requests.
func SetUserAgent(ua string) {
	userAgent = ua
}

// CLITransport wraps an http.RoundTripper with CLI-specific behavior:
// retry logic, verbose logging, user-agent override, base URL rewrite,
// and error body capture for the error bridge.
type CLITransport struct {
	Base      http.RoundTripper
	Verbose   bool
	BaseURL   string // if set, replaces the SDK's hardcoded base URL
	AccountID string // if set, sends X-Acc-Id header on all requests

	mu       sync.Mutex
	lastBody []byte // stores last error response body
}

// LastErrorBody returns the most recently captured error response body
// and clears it. This is used by WrapError to extract field-level
// validation details that the SDK discards.
func (t *CLITransport) LastErrorBody() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	b := t.lastBody
	t.lastBody = nil
	return b
}

func (t *CLITransport) setLastBody(b []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastBody = b
}

func (t *CLITransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

func (t *CLITransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite base URL if configured.
	if t.BaseURL != "" {
		urlStr := req.URL.String()
		if strings.HasPrefix(urlStr, defaultBaseURL) {
			newURL := t.BaseURL + strings.TrimPrefix(urlStr, defaultBaseURL)
			parsed, err := req.URL.Parse(newURL)
			if err == nil {
				req.URL = parsed
			}
		}
	}

	// Override User-Agent.
	req.Header.Set("User-Agent", userAgent)

	// Set account context header if configured.
	if t.AccountID != "" {
		req.Header.Set("X-Acc-Id", t.AccountID)
	}

	// Capture request body for retries.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	if t.Verbose {
		fmt.Printf("--> %s %s\n", req.Method, req.URL)
		if t.AccountID != "" {
			fmt.Printf("--> X-Acc-Id: %s\n", t.AccountID)
		}
		if len(bodyBytes) > 0 {
			fmt.Printf("--> body: %s\n", string(bodyBytes))
		}
	}

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Reset body for retry.
			if len(bodyBytes) > 0 {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}

		resp, lastErr = t.base().RoundTrip(req)
		if lastErr != nil {
			if t.Verbose {
				fmt.Printf("<-- error: %v\n", lastErr)
			}
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoff)
			continue
		}

		if t.Verbose {
			fmt.Printf("<-- %d %s\n", resp.StatusCode, resp.Status)
		}

		// Capture error response body before the SDK can consume it.
		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close() //nolint:errcheck

			if t.Verbose && len(respBody) > 0 {
				fmt.Printf("<-- body: %s\n", string(respBody))
			}

			// Store for WrapError.
			t.setLastBody(respBody)

			// For retryable errors, retry.
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
					if secs, err := strconv.Atoi(retryAfter); err == nil {
						wait = time.Duration(secs) * time.Second
					}
				}
				if attempt < maxRetries {
					if t.Verbose {
						fmt.Printf("    retrying in %s...\n", wait)
					}
					time.Sleep(wait)
					continue
				}
			}

			// Replace the body so the SDK can still read it.
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
			return resp, nil
		}

		// Success — capture body for verbose logging, then re-wrap.
		if t.Verbose {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close() //nolint:errcheck
			if len(respBody) > 0 {
				fmt.Printf("<-- body: %s\n", string(respBody))
			}
			resp.Body = io.NopCloser(bytes.NewReader(respBody))
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
	}
	return resp, nil
}
