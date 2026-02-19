package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDirCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}

	expected := filepath.Join(tmpDir, "atlassian-cloud")
	if dir != expected {
		t.Errorf("Dir() = %q, want %q", dir, expected)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("dir permissions = %o, want 0700", info.Mode().Perm())
	}
}

func TestAuthConfigRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	site := SiteAuth{
		Method:       AuthMethodOAuth2,
		AccessToken:  "test-access",
		RefreshToken: "test-refresh",
		TokenExpiry:  "2026-02-19T15:30:00Z",
		CloudID:      "abc-123",
		Scopes:       []string{"read:jira-work"},
	}

	auth := &AuthConfig{
		DefaultSite: "test.atlassian.net",
		Sites: map[string]SiteAuth{
			"test.atlassian.net": site,
		},
	}

	if err := SaveAuthConfig(auth); err != nil {
		t.Fatalf("SaveAuthConfig error: %v", err)
	}

	loaded, err := LoadAuthConfig()
	if err != nil {
		t.Fatalf("LoadAuthConfig error: %v", err)
	}

	if loaded.DefaultSite != auth.DefaultSite {
		t.Errorf("DefaultSite = %q, want %q", loaded.DefaultSite, auth.DefaultSite)
	}

	s, ok := loaded.Sites["test.atlassian.net"]
	if !ok {
		t.Fatal("site not found in loaded config")
	}
	if s.AccessToken != "test-access" {
		t.Errorf("AccessToken = %q, want %q", s.AccessToken, "test-access")
	}

	// Verify file permissions
	authPath := filepath.Join(tmpDir, "atlassian-cloud", "auth.json")
	info, err := os.Stat(authPath)
	if err != nil {
		t.Fatalf("stat auth.json error: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("auth.json permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestLoadAuthConfigNoFile(t *testing.T) {
	// Simulate first-run: no auth.json exists yet.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg, err := LoadAuthConfig()
	if err != nil {
		t.Fatalf("LoadAuthConfig() unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadAuthConfig() returned nil config")
	}
	if cfg.Sites == nil {
		t.Fatal("LoadAuthConfig() returned nil Sites map, want initialized map")
	}
	if len(cfg.Sites) != 0 {
		t.Errorf("LoadAuthConfig() Sites has %d entries, want 0", len(cfg.Sites))
	}
	if cfg.DefaultSite != "" {
		t.Errorf("LoadAuthConfig() DefaultSite = %q, want empty string", cfg.DefaultSite)
	}
}

func TestLoadAuthConfigMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create the config directory so we can write auth.json directly.
	configDir := filepath.Join(tmpDir, "atlassian-cloud")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	authPath := filepath.Join(configDir, "auth.json")
	if err := os.WriteFile(authPath, []byte(`{not json`), 0600); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	cfg, err := LoadAuthConfig()
	if err == nil {
		t.Fatal("LoadAuthConfig() expected error for malformed JSON, got nil")
	}
	if cfg != nil {
		t.Errorf("LoadAuthConfig() expected nil config on error, got %+v", cfg)
	}
}

func TestDirWithoutXDG(t *testing.T) {
	// Ensure XDG_CONFIG_HOME is unset so Dir() falls back to ~/.config.
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error: %v", err)
	}

	expected := filepath.Join(home, ".config", "atlassian-cloud")
	if dir != expected {
		t.Errorf("Dir() = %q, want %q", dir, expected)
	}
}
