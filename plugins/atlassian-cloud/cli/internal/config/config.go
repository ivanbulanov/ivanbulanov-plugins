package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const appName = "atlassian-cloud"

const (
	AuthMethodOAuth2 = "oauth2"
	AuthMethodToken  = "token"
)

type AuthConfig struct {
	DefaultSite string              `json:"default_site"`
	Sites       map[string]SiteAuth `json:"sites"`
}

type SiteAuth struct {
	Method       string   `json:"method"`
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	TokenExpiry  string   `json:"token_expiry,omitempty"`
	CloudID      string   `json:"cloud_id,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Email        string   `json:"email,omitempty"`
	APIToken     string   `json:"api_token,omitempty"`
}

// ExpiryTime parses TokenExpiry and returns the resulting time.
// Returns the zero time and an error if parsing fails or TokenExpiry is empty.
func (s *SiteAuth) ExpiryTime() (time.Time, error) {
	if s.TokenExpiry == "" {
		return time.Time{}, fmt.Errorf("token expiry not set")
	}
	return time.Parse(time.RFC3339, s.TokenExpiry)
}

func Dir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}

	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}

	// Ensure correct permissions even if directory already existed
	if err := os.Chmod(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot set config directory permissions: %w", err)
	}

	return dir, nil
}

func authConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth.json"), nil
}

func LoadAuthConfig() (*AuthConfig, error) {
	path, err := authConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &AuthConfig{Sites: make(map[string]SiteAuth)}, nil
		}
		return nil, fmt.Errorf("cannot read auth config: %w", err)
	}

	var cfg AuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse auth config: %w", err)
	}
	if cfg.Sites == nil {
		cfg.Sites = make(map[string]SiteAuth)
	}

	return &cfg, nil
}

func SaveAuthConfig(cfg *AuthConfig) error {
	path, err := authConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal auth config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("cannot write auth config: %w", err)
	}

	// Ensure correct permissions even if file already existed
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("cannot set auth config permissions: %w", err)
	}

	return nil
}
