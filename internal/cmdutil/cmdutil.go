package cmdutil

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/mailerlite/mailerlite-cli/internal/config"
	"github.com/mailerlite/mailerlite-cli/internal/sdkclient"
	"github.com/mailerlite/mailerlite-go"
	"github.com/spf13/cobra"
)

// ProfileFlag returns the --profile persistent flag value.
func ProfileFlag(cmd *cobra.Command) string {
	v, _ := cmd.Root().PersistentFlags().GetString("profile")
	return v
}

// VerboseFlag returns the --verbose persistent flag value.
func VerboseFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	return v
}

// JSONFlag returns the --json persistent flag value.
func JSONFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Root().PersistentFlags().GetBool("json")
	return v
}

// YesFlag returns the --yes persistent flag value.
func YesFlag(cmd *cobra.Command) bool {
	v, _ := cmd.Root().PersistentFlags().GetBool("yes")
	return v
}

// SetVersion configures the SDK client user-agent with the CLI version.
func SetVersion(v string) {
	sdkclient.SetUserAgent("mailerlite-cli/" + v)
}

// NewSDKClient creates a mailerlite-go SDK client with CLI-specific behavior
// injected via a custom HTTP transport (retry, verbose, user-agent, base URL).
// Returns both the SDK client and the transport (needed for error body access).
func NewSDKClient(cmd *cobra.Command) (*mailerlite.Client, *sdkclient.CLITransport, error) {
	token, err := config.GetToken(ProfileFlag(cmd))
	if err != nil {
		return nil, nil, err
	}

	transport := &sdkclient.CLITransport{
		Base:      http.DefaultTransport,
		Verbose:   VerboseFlag(cmd),
		AccountID: config.GetAccountID(ProfileFlag(cmd)),
	}

	if base := os.Getenv("MAILERLITE_API_BASE_URL"); base != "" {
		transport.BaseURL = base
	}

	ml := mailerlite.NewClient(token)
	ml.SetHttpClient(&http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	})

	return ml, transport, nil
}

// RawHTTPClient creates an *http.Client and API key for raw HTTP e-commerce calls.
func RawHTTPClient(cmd *cobra.Command) (*http.Client, string, *sdkclient.CLITransport, error) {
	token, err := config.GetToken(ProfileFlag(cmd))
	if err != nil {
		return nil, "", nil, err
	}

	transport := &sdkclient.CLITransport{
		Base:      http.DefaultTransport,
		Verbose:   VerboseFlag(cmd),
		AccountID: config.GetAccountID(ProfileFlag(cmd)),
	}

	httpClient := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}

	return httpClient, token, transport, nil
}

// ParseDate accepts a date string in YYYY-MM-DD format or a raw unix
// timestamp and returns the corresponding unix timestamp as int64.
func ParseDate(value string) (int64, error) {
	t, err := time.Parse("2006-01-02", value)
	if err == nil {
		return t.Unix(), nil
	}
	ts, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid date %q: use YYYY-MM-DD or a unix timestamp", value)
	}
	return ts, nil
}

// DefaultDateRange returns parsed dateFrom/dateTo timestamps. If either value
// is empty, it defaults to the last 7 days (dateTo = now, dateFrom = now - 7d).
func DefaultDateRange(dateFromStr, dateToStr string, now time.Time) (int64, int64, error) {
	var dateFrom, dateTo int64
	var err error

	if dateFromStr == "" && dateToStr == "" {
		dateTo = now.Unix()
		dateFrom = now.AddDate(0, 0, -7).Unix()
		return dateFrom, dateTo, nil
	}

	if dateFromStr != "" {
		dateFrom, err = ParseDate(dateFromStr)
		if err != nil {
			return 0, 0, err
		}
	} else {
		dateFrom = now.AddDate(0, 0, -7).Unix()
	}

	if dateToStr != "" {
		dateTo, err = ParseDate(dateToStr)
		if err != nil {
			return 0, 0, err
		}
	} else {
		dateTo = now.Unix()
	}

	return dateFrom, dateTo, nil
}
