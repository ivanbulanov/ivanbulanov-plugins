package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/config"
)

// ---------------------------------------------------------------------------
// TestLoadOAuthConfigFromEnv
// ---------------------------------------------------------------------------

func TestLoadOAuthConfigFromEnv(t *testing.T) {
	t.Run("both vars set", func(t *testing.T) {
		t.Setenv("ATLASSIAN_CLIENT_ID", "test-client-id")
		t.Setenv("ATLASSIAN_CLIENT_SECRET", "test-client-secret")

		cfg := LoadOAuthConfigFromEnv()
		if cfg == nil {
			t.Fatal("expected non-nil config when both env vars are set")
		}
		if cfg.ClientID != "test-client-id" {
			t.Errorf("ClientID = %q, want %q", cfg.ClientID, "test-client-id")
		}
		if cfg.ClientSecret != "test-client-secret" {
			t.Errorf("ClientSecret = %q, want %q", cfg.ClientSecret, "test-client-secret")
		}
		wantRedirect := fmt.Sprintf("http://localhost:%d/callback", CallbackPort)
		if cfg.RedirectURI != wantRedirect {
			t.Errorf("RedirectURI = %q, want %q", cfg.RedirectURI, wantRedirect)
		}
		if len(cfg.Scopes) == 0 {
			t.Error("expected non-empty Scopes")
		}
	})

	t.Run("client id missing", func(t *testing.T) {
		t.Setenv("ATLASSIAN_CLIENT_ID", "")
		t.Setenv("ATLASSIAN_CLIENT_SECRET", "test-client-secret")

		cfg := LoadOAuthConfigFromEnv()
		if cfg != nil {
			t.Error("expected nil config when ATLASSIAN_CLIENT_ID is empty")
		}
	})

	t.Run("client secret missing", func(t *testing.T) {
		t.Setenv("ATLASSIAN_CLIENT_ID", "test-client-id")
		t.Setenv("ATLASSIAN_CLIENT_SECRET", "")

		cfg := LoadOAuthConfigFromEnv()
		if cfg != nil {
			t.Error("expected nil config when ATLASSIAN_CLIENT_SECRET is empty")
		}
	})

	t.Run("both vars missing", func(t *testing.T) {
		t.Setenv("ATLASSIAN_CLIENT_ID", "")
		t.Setenv("ATLASSIAN_CLIENT_SECRET", "")

		cfg := LoadOAuthConfigFromEnv()
		if cfg != nil {
			t.Error("expected nil config when both env vars are empty")
		}
	})
}

// ---------------------------------------------------------------------------
// TestAuthRequiredError
// ---------------------------------------------------------------------------

func TestAuthRequiredError(t *testing.T) {
	msg := "authentication failed; run: atlassian-cloud auth login"
	err := &AuthRequiredError{Message: msg}

	// Verify it satisfies the error interface.
	var _ error = err

	if err.Error() != msg {
		t.Errorf("Error() = %q, want %q", err.Error(), msg)
	}

	// Verify errors.As works.
	var target *AuthRequiredError
	if !errors.As(err, &target) {
		t.Error("errors.As did not match *AuthRequiredError")
	}
}

// ---------------------------------------------------------------------------
// TestWrapAPIError
// ---------------------------------------------------------------------------

func TestWrapAPIError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		if got := WrapAPIError(200, nil); got != nil {
			t.Errorf("WrapAPIError(200, nil) = %v, want nil", got)
		}
	})

	t.Run("nil error with zero code returns nil", func(t *testing.T) {
		if got := WrapAPIError(0, nil); got != nil {
			t.Errorf("WrapAPIError(0, nil) = %v, want nil", got)
		}
	})

	t.Run("401 returns AuthRequiredError", func(t *testing.T) {
		orig := errors.New("unauthorized")
		got := WrapAPIError(401, orig)
		if got == nil {
			t.Fatal("expected non-nil error for 401")
		}
		var authErr *AuthRequiredError
		if !errors.As(got, &authErr) {
			t.Fatalf("expected *AuthRequiredError, got %T", got)
		}
		if authErr.Message == "" {
			t.Error("AuthRequiredError message should not be empty")
		}
	})

	t.Run("non-401 passes through original error", func(t *testing.T) {
		orig := errors.New("server error")
		got := WrapAPIError(500, orig)
		if got != orig {
			t.Errorf("expected original error to be returned unchanged; got %v, want %v", got, orig)
		}
	})

	t.Run("code 0 with non-nil error passes through", func(t *testing.T) {
		orig := errors.New("connection failed")
		got := WrapAPIError(0, orig)
		if got != orig {
			t.Errorf("expected original error for code 0; got %v, want %v", got, orig)
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers for NewClients tests
// ---------------------------------------------------------------------------

// writeAuthConfig writes a config.AuthConfig as JSON to the standard location
// under the given XDG_CONFIG_HOME directory.
func writeAuthConfig(t *testing.T, xdgDir string, cfg *config.AuthConfig) {
	t.Helper()
	dir := filepath.Join(xdgDir, "atlassian-cloud")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("cannot create config dir: %v", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("cannot marshal config: %v", err)
	}
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("cannot write config: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestNewClients_EmptySite
// ---------------------------------------------------------------------------

func TestNewClients_EmptySite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write a config with no sites and no default site.
	writeAuthConfig(t, tmpDir, &config.AuthConfig{
		Version:     1,
		DefaultSite: "",
		Sites:       map[string]config.SiteAuth{},
	})

	_, err := NewClients("")
	if err == nil {
		t.Fatal("expected error when no sites configured and empty site arg")
	}

	var authErr *AuthRequiredError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthRequiredError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// TestNewClients_UnknownSite
// ---------------------------------------------------------------------------

func TestNewClients_UnknownSite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write a config with one known site.
	writeAuthConfig(t, tmpDir, &config.AuthConfig{
		Version:     1,
		DefaultSite: "existing.atlassian.net",
		Sites: map[string]config.SiteAuth{
			"existing.atlassian.net": {
				Method:   config.AuthMethodToken,
				Email:    "user@example.com",
				APIToken: "tok",
			},
		},
	})

	_, err := NewClients("nonexistent.atlassian.net")
	if err == nil {
		t.Fatal("expected error for unknown site")
	}

	var authErr *AuthRequiredError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected *AuthRequiredError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// TestNewClients_UnknownMethod
// ---------------------------------------------------------------------------

func TestNewClients_UnknownMethod(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	siteName := "test.atlassian.net"
	writeAuthConfig(t, tmpDir, &config.AuthConfig{
		Version:     1,
		DefaultSite: siteName,
		Sites: map[string]config.SiteAuth{
			siteName: {
				Method: "unknown",
			},
		},
	})

	_, err := NewClients(siteName)
	if err == nil {
		t.Fatal("expected error for unknown auth method")
	}

	// Should NOT be an AuthRequiredError -- it's a generic error about an unknown method.
	var authErr *AuthRequiredError
	if errors.As(err, &authErr) {
		t.Fatalf("did not expect *AuthRequiredError, got one: %v", authErr)
	}

	// Verify the error message mentions the unknown method.
	if got := err.Error(); !containsSubstring(got, "unknown auth method") {
		t.Errorf("error message %q should mention 'unknown auth method'", got)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
