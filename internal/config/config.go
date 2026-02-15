package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	oauthClientID = "1118"
	oauthTokenURL = "https://dashboard.mailerlite.com/oauth/token"
)

type Profile struct {
	APIToken          string `yaml:"api_token,omitempty"`
	OAuthToken        string `yaml:"oauth_token,omitempty"`
	OAuthRefreshToken string `yaml:"oauth_refresh_token,omitempty"`
	OAuthExpiresAt    string `yaml:"oauth_expires_at,omitempty"`
	AccountID         string `yaml:"account_id,omitempty"`
}

type Config struct {
	ActiveProfile string             `yaml:"active_profile"`
	Profiles      map[string]Profile `yaml:"profiles"`
}

func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "mailerlite"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "mailerlite"), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Profiles: make(map[string]Profile)}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(p, data, 0600)
}

func ActiveProfile(cfg *Config) (string, Profile, error) {
	name := cfg.ActiveProfile
	if name == "" {
		if len(cfg.Profiles) == 0 {
			return "", Profile{}, fmt.Errorf("no profiles configured — run 'mailerlite auth login' or 'mailerlite profile add <name>'")
		}
		for n := range cfg.Profiles {
			name = n
			break
		}
	}

	p, ok := cfg.Profiles[name]
	if !ok {
		return "", Profile{}, fmt.Errorf("profile %q not found", name)
	}
	return name, p, nil
}

func GetToken(profileOverride string) (string, error) {
	// Environment variable takes highest precedence.
	if token := os.Getenv("MAILERLITE_API_TOKEN"); token != "" {
		return token, nil
	}

	cfg, err := Load()
	if err != nil {
		return "", err
	}

	var profName string
	var prof Profile
	if profileOverride != "" {
		p, ok := cfg.Profiles[profileOverride]
		if !ok {
			return "", fmt.Errorf("profile %q not found", profileOverride)
		}
		profName = profileOverride
		prof = p
	} else {
		n, p, err := ActiveProfile(cfg)
		if err != nil {
			return "", err
		}
		profName = n
		prof = p
	}

	if prof.APIToken != "" {
		return prof.APIToken, nil
	}
	if prof.OAuthToken != "" {
		// Check if token is expired and refresh if needed.
		if prof.OAuthExpiresAt != "" && prof.OAuthRefreshToken != "" {
			expiresAt, err := time.Parse(time.RFC3339, prof.OAuthExpiresAt)
			if err == nil && time.Now().After(expiresAt.Add(-5*time.Minute)) {
				refreshed, refreshErr := refreshOAuthToken(prof.OAuthRefreshToken)
				if refreshErr == nil {
					refreshed.AccountID = prof.AccountID
					cfg.Profiles[profName] = refreshed
					_ = Save(cfg)
					return refreshed.OAuthToken, nil
				}
				// If refresh fails but token isn't actually expired yet, use it anyway.
				if time.Now().Before(expiresAt) {
					return prof.OAuthToken, nil
				}
				return "", fmt.Errorf("OAuth token expired and refresh failed: %w", refreshErr)
			}
		}
		return prof.OAuthToken, nil
	}
	return "", fmt.Errorf("no token found — run 'mailerlite auth login' or set MAILERLITE_API_TOKEN")
}

// GetAccountID returns the stored account ID for the active (or overridden) profile.
func GetAccountID(profileOverride string) string {
	cfg, err := Load()
	if err != nil {
		return ""
	}

	var prof Profile
	if profileOverride != "" {
		p, ok := cfg.Profiles[profileOverride]
		if !ok {
			return ""
		}
		prof = p
	} else {
		_, p, err := ActiveProfile(cfg)
		if err != nil {
			return ""
		}
		prof = p
	}

	return prof.AccountID
}

// refreshOAuthToken exchanges a refresh token for a new access token.
func refreshOAuthToken(refreshToken string) (Profile, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {oauthClientID},
		"refresh_token": {refreshToken},
	}

	resp, err := http.Post(oauthTokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode())) //nolint:gosec,noctx
	if err != nil {
		return Profile{}, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return Profile{}, fmt.Errorf("refresh failed (HTTP %d)", resp.StatusCode)
	}

	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return Profile{}, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	if tok.AccessToken == "" {
		return Profile{}, fmt.Errorf("server returned empty access token on refresh")
	}

	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second).Format(time.RFC3339)

	return Profile{
		OAuthToken:        tok.AccessToken,
		OAuthRefreshToken: tok.RefreshToken,
		OAuthExpiresAt:    expiresAt,
	}, nil
}
