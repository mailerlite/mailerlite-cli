package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/mailerlite/mailerlite-cli/internal/config"
	"github.com/mailerlite/mailerlite-cli/internal/output"
	"github.com/mailerlite/mailerlite-cli/internal/prompt"
	"github.com/spf13/cobra"
)

const (
	oauthClientID     = "1118"
	oauthAuthorizeURL = "https://dashboard.mailerlite.com/oauth/authorize"
	oauthTokenURL     = "https://dashboard.mailerlite.com/oauth/token"
	oauthScopes       = "manager-app"
)

var Cmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with MailerLite",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to MailerLite",
	Long:  "Authenticate via API token or OAuth browser flow.",
	RunE:  runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored credentials",
	RunE:  runLogout,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE:  runStatus,
}

func init() {
	loginCmd.Flags().String("method", "", "auth method: token or oauth")
	loginCmd.Flags().String("token", "", "API token (for token method)")
	loginCmd.Flags().String("profile", "", "profile name to save credentials to (default: uses active profile or 'default')")
	Cmd.AddCommand(loginCmd, logoutCmd, statusCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	method, _ := cmd.Flags().GetString("method")
	token, _ := cmd.Flags().GetString("token")
	profName, _ := cmd.Flags().GetString("profile")

	if method == "" && prompt.IsInteractive() {
		var err error
		method, err = prompt.SelectLabeled("Authentication method", []string{"OAuth (Recommended)", "API Token (less secure)"}, []string{"oauth", "token"})
		if err != nil {
			return err
		}
	}
	if method == "" {
		method = "oauth"
	}

	if profName == "" {
		profName = "default"
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch method {
	case "token":
		if token == "" {
			if !prompt.IsInteractive() {
				return fmt.Errorf("--token is required in non-interactive mode")
			}
			token, err = prompt.Input("API Token", "")
			if err != nil {
				return err
			}
		}
		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}
		cfg.Profiles[profName] = config.Profile{APIToken: token}

	case "oauth":
		prof, err := oauthBrowserFlow()
		if err != nil {
			return fmt.Errorf("OAuth login failed: %w", err)
		}
		cfg.Profiles[profName] = prof

	default:
		return fmt.Errorf("unknown auth method: %s (use 'token' or 'oauth')", method)
	}

	cfg.ActiveProfile = profName
	if err := config.Save(cfg); err != nil {
		return err
	}

	output.Success(fmt.Sprintf("Logged in successfully. Profile: %s", profName))
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	profFlag, _ := cmd.Root().PersistentFlags().GetString("profile")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	name := profFlag
	if name == "" {
		name = cfg.ActiveProfile
	}
	if name == "" {
		name = "default"
	}

	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	delete(cfg.Profiles, name)
	if cfg.ActiveProfile == name {
		cfg.ActiveProfile = ""
		for n := range cfg.Profiles {
			cfg.ActiveProfile = n
			break
		}
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	output.Success(fmt.Sprintf("Logged out from profile: %s", name))
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	profFlag, _ := cmd.Root().PersistentFlags().GetString("profile")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	name := profFlag
	if name == "" {
		name, _, err = config.ActiveProfile(cfg)
		if err != nil {
			output.Error("Not logged in. Run 'mailerlite auth login' to authenticate.")
			return nil
		}
	}

	prof, ok := cfg.Profiles[name]
	if !ok {
		output.Error(fmt.Sprintf("Profile %q not found.", name))
		return nil
	}

	jsonFlag, _ := cmd.Root().PersistentFlags().GetBool("json")
	if jsonFlag {
		return output.JSON(map[string]interface{}{
			"profile":    name,
			"has_token":  prof.APIToken != "",
			"has_oauth":  prof.OAuthToken != "",
			"expires_at": prof.OAuthExpiresAt,
			"account_id": prof.AccountID,
		})
	}

	method := "API Token"
	if prof.OAuthToken != "" {
		method = "OAuth"
	}
	maskedToken := "none"
	if prof.APIToken != "" {
		t := prof.APIToken
		if len(t) > 10 {
			maskedToken = t[:7] + "..." + t[len(t)-4:]
		} else {
			maskedToken = "***"
		}
	}

	rows := [][]string{
		{"Profile", name},
		{"Method", method},
		{"Token", maskedToken},
		{"Active", "Yes"},
	}

	if prof.OAuthExpiresAt != "" {
		rows = append(rows, []string{"Expires", prof.OAuthExpiresAt})
	}
	if prof.AccountID != "" {
		rows = append(rows, []string{"Account ID", prof.AccountID})
	}

	output.Table(
		[]string{"Field", "Value"},
		rows,
	)
	return nil
}

// oauthBrowserFlow performs the full OAuth 2.0 Authorization Code flow with PKCE.
// It starts a local HTTP server, opens the browser to the authorize URL,
// captures the authorization code, and exchanges it for access/refresh tokens.
func oauthBrowserFlow() (config.Profile, error) {
	state, err := randomHex(16)
	if err != nil {
		return config.Profile{}, err
	}

	verifier, challenge, err := generatePKCE()
	if err != nil {
		return config.Profile{}, err
	}

	const callbackPort = "19821"
	callbackURL := "http://127.0.0.1:" + callbackPort + "/callback"

	listener, err := net.Listen("tcp", "127.0.0.1:"+callbackPort)
	if err != nil {
		return config.Profile{}, fmt.Errorf("failed to start local server on port %s: %w", callbackPort, err)
	}
	defer listener.Close() //nolint:errcheck

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code != "" {
			codeCh <- code
			fmt.Fprintf(w, "<html><body><h2>Authentication successful!</h2><p>You can close this window.</p></body></html>") //nolint:errcheck
		} else {
			errMsg := r.URL.Query().Get("error")
			errCh <- fmt.Errorf("OAuth error: %s", errMsg)
			fmt.Fprintf(w, "<html><body><h2>Authentication failed</h2><p>%s</p></body></html>", errMsg) //nolint:errcheck
		}
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Shutdown(context.Background()) //nolint:errcheck // best-effort shutdown

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&code_challenge=%s&code_challenge_method=S256",
		oauthAuthorizeURL,
		oauthClientID,
		url.QueryEscape(callbackURL),
		url.QueryEscape(oauthScopes),
		state,
		challenge,
	)

	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", authURL)
	openBrowser(authURL)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return config.Profile{}, err
	case <-time.After(5 * time.Minute):
		return config.Profile{}, fmt.Errorf("authentication timed out after 5 minutes")
	}

	// Exchange authorization code for tokens.
	return exchangeCodeForTokens(code, callbackURL, verifier)
}

// tokenResponse represents the JSON response from the OAuth token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// exchangeCodeForTokens POSTs to the token endpoint with the
// authorization code and PKCE verifier to obtain access and refresh tokens.
func exchangeCodeForTokens(code, redirectURI, codeVerifier string) (config.Profile, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {oauthClientID},
		"redirect_uri":  {redirectURI},
		"code":          {code},
		"code_verifier": {codeVerifier},
	}

	resp, err := http.Post(oauthTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode())) //nolint:gosec,noctx
	if err != nil {
		return config.Profile{}, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		var body map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return config.Profile{}, fmt.Errorf("token exchange failed (HTTP %d): %v", resp.StatusCode, body)
	}

	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return config.Profile{}, fmt.Errorf("failed to parse token response: %w", err)
	}

	if tok.AccessToken == "" {
		return config.Profile{}, fmt.Errorf("server returned empty access token")
	}

	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second).Format(time.RFC3339)

	return config.Profile{
		OAuthToken:        tok.AccessToken,
		OAuthRefreshToken: tok.RefreshToken,
		OAuthExpiresAt:    expiresAt,
	}, nil
}

// generatePKCE creates a PKCE code verifier and its S256 challenge.
func generatePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)

	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
