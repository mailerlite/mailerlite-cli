package sdkclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const rawDefaultBaseURL = "https://connect.mailerlite.com/api"

// DoRaw performs a raw HTTP request for endpoints not covered by the SDK
// (e.g., e-commerce). It uses the provided httpClient (which should have
// the CLITransport configured) for retry/verbose behavior.
func DoRaw(ctx context.Context, httpClient *http.Client, apiKey, method, path string, body, result interface{}) (*http.Response, error) {
	url := rawDefaultBaseURL + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		cliErr := &CLIError{StatusCode: resp.StatusCode}
		if len(respBody) > 0 {
			cliErr.RawBody = respBody
			var parsed struct {
				Message string              `json:"message"`
				Errors  map[string][]string `json:"errors"`
			}
			if json.Unmarshal(respBody, &parsed) == nil {
				cliErr.Message = parsed.Message
				cliErr.Errors = parsed.Errors
			}
		}
		if cliErr.Message == "" {
			cliErr.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return resp, cliErr
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return resp, fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return resp, nil
}

// DoRawQuery is like DoRaw but accepts query parameters as a map.
func DoRawQuery(ctx context.Context, httpClient *http.Client, apiKey, method, path string, query map[string]string, body, result interface{}) (*http.Response, error) {
	if len(query) > 0 {
		sep := "?"
		for k, v := range query {
			path += sep + k + "=" + v
			sep = "&"
		}
	}
	return DoRaw(ctx, httpClient, apiKey, method, path, body, result)
}
