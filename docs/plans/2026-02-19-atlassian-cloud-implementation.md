# Atlassian Cloud Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Claude Code plugin that provides context-efficient Jira and Confluence Cloud access via a Go CLI built on go-atlassian.

**Architecture:** A single Go CLI binary (`atlassian-cloud`) with Cobra subcommands for auth, jira, and confluence operations. Plugin skills invoke the CLI via bash. OAuth2 3LO with automatic token renewal is the primary auth method, API tokens are the fallback. Output is clean markdown with progressive disclosure (summary → description → comments → attachments).

**Tech Stack:** Go 1.21+, cobra (CLI), go-atlassian v2 (Atlassian API), Confluence v1+v2 clients (v1 for CQL search, v2 for pages/attachments).

**Design doc:** `docs/plans/2026-02-19-atlassian-cloud-plugin-design.md`

---

## Task 1: Scaffold Plugin Structure & Go Module

**Files:**
- Create: `plugins/atlassian-cloud/.claude-plugin/plugin.json`
- Create: `plugins/atlassian-cloud/README.md`
- Create: `plugins/atlassian-cloud/cli/go.mod`
- Create: `plugins/atlassian-cloud/cli/main.go`
- Create: `plugins/atlassian-cloud/cli/cmd/root.go`
- Create: `plugins/atlassian-cloud/cli/.gitignore`
- Modify: `.claude-plugin/marketplace.json`

**Step 1: Create plugin directory structure**

```bash
mkdir -p plugins/atlassian-cloud/.claude-plugin
mkdir -p plugins/atlassian-cloud/cli/cmd
mkdir -p plugins/atlassian-cloud/cli/internal/auth
mkdir -p plugins/atlassian-cloud/cli/internal/output
mkdir -p plugins/atlassian-cloud/cli/internal/config
mkdir -p plugins/atlassian-cloud/cli/internal/urlparse
mkdir -p plugins/atlassian-cloud/skills/atlassian-jira/scripts
mkdir -p plugins/atlassian-cloud/skills/atlassian-confluence
mkdir -p plugins/atlassian-cloud/references
```

**Step 2: Create plugin.json**

Write `plugins/atlassian-cloud/.claude-plugin/plugin.json`:
```json
{
  "name": "atlassian-cloud",
  "version": "0.1.0",
  "description": "Context-efficient Jira and Confluence Cloud access via Go CLI with OAuth2 authentication, progressive disclosure, and ADF-to-markdown conversion",
  "author": {
    "name": "Ivan Bulanov",
    "url": "https://github.com/ivanbulanov"
  },
  "repository": "https://github.com/ivanbulanov/ivanbulanov-plugins",
  "license": "MIT",
  "keywords": ["jira", "confluence", "atlassian", "oauth2", "cloud"]
}
```

**Step 3: Update marketplace.json**

Add to the `plugins` array in `.claude-plugin/marketplace.json`:
```json
{
  "name": "atlassian-cloud",
  "description": "Context-efficient Jira and Confluence Cloud access via Go CLI with OAuth2, progressive disclosure, and ADF-to-markdown conversion",
  "version": "0.1.0",
  "author": {
    "name": "Ivan Bulanov",
    "url": "https://github.com/ivanbulanov"
  },
  "source": "./plugins/atlassian-cloud",
  "category": "productivity",
  "homepage": "https://github.com/ivanbulanov/ivanbulanov-plugins",
  "keywords": ["jira", "confluence", "atlassian", "oauth2", "cloud"]
}
```

**Step 4: Initialize Go module**

```bash
cd plugins/atlassian-cloud/cli
```

Ensure Go is installed (use mise if needed):
```bash
which go || mise install go@latest
```

Initialize module:
```bash
go mod init github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli
```

**Step 5: Create cli/.gitignore**

Write `plugins/atlassian-cloud/cli/.gitignore`:
```
bin/
atlassian-cloud
```

**Step 6: Create main.go**

Write `plugins/atlassian-cloud/cli/main.go`:
```go
package main

import "github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/cmd"

func main() {
	cmd.Execute()
}
```

**Step 7: Create root command**

Write `plugins/atlassian-cloud/cli/cmd/root.go`:
```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "atlassian-cloud",
	Short: "Context-efficient Jira and Confluence Cloud CLI",
	Long:  "Access Jira issues, Confluence pages, and more with progressive disclosure and OAuth2 authentication.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

**Step 8: Add dependencies**

```bash
cd plugins/atlassian-cloud/cli
go get github.com/spf13/cobra@latest
go get github.com/ctreminiom/go-atlassian/v2@latest
go mod tidy
```

**Step 9: Verify it builds**

```bash
cd plugins/atlassian-cloud/cli
go build -o bin/atlassian-cloud .
./bin/atlassian-cloud --help
```

Expected: Help output with "Context-efficient Jira and Confluence Cloud CLI".

**Step 10: Commit**

```bash
git add plugins/atlassian-cloud/ .claude-plugin/marketplace.json
git commit -m "feat: scaffold atlassian-cloud plugin with Go module"
```

---

## Task 2: Config & Auth Storage

**Files:**
- Create: `plugins/atlassian-cloud/cli/internal/config/config.go`
- Create: `plugins/atlassian-cloud/cli/internal/config/config_test.go`

**Step 1: Write the config test**

Write `plugins/atlassian-cloud/cli/internal/config/config_test.go`:
```go
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
		Method:       "oauth2",
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
```

**Step 2: Run test to verify it fails**

```bash
cd plugins/atlassian-cloud/cli
go test ./internal/config/ -v
```

Expected: FAIL — package doesn't exist yet.

**Step 3: Write config implementation**

Write `plugins/atlassian-cloud/cli/internal/config/config.go`:
```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const appName = "atlassian-cloud"

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
```

**Step 4: Run test to verify it passes**

```bash
cd plugins/atlassian-cloud/cli
go test ./internal/config/ -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/config/
git commit -m "feat: add config storage with secure file permissions"
```

---

## Task 3: URL Parser

**Files:**
- Create: `plugins/atlassian-cloud/cli/internal/urlparse/urlparse.go`
- Create: `plugins/atlassian-cloud/cli/internal/urlparse/urlparse_test.go`

**Step 1: Write the URL parser test**

Write `plugins/atlassian-cloud/cli/internal/urlparse/urlparse_test.go`:
```go
package urlparse

import "testing"

func TestParseJiraURL(t *testing.T) {
	tests := []struct {
		input   string
		wantKey string
		wantSite string
		wantOk  bool
	}{
		{"DEV-123", "DEV-123", "", true},
		{"PROJ-1", "PROJ-1", "", true},
		{"https://acme-corp.atlassian.net/browse/ACME-5136", "ACME-5136", "acme-corp.atlassian.net", true},
		{"https://company.atlassian.net/browse/PROJ-456?focusedCommentId=123", "PROJ-456", "company.atlassian.net", true},
		{"not-a-ticket", "", "", false},
		{"", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, ok := ParseJiraRef(tt.input)
			if ok != tt.wantOk {
				t.Fatalf("ParseJiraRef(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if result.IssueKey != tt.wantKey {
				t.Errorf("IssueKey = %q, want %q", result.IssueKey, tt.wantKey)
			}
			if result.Site != tt.wantSite {
				t.Errorf("Site = %q, want %q", result.Site, tt.wantSite)
			}
		})
	}
}

func TestParseConfluenceURL(t *testing.T) {
	tests := []struct {
		input    string
		wantID   string
		wantSite string
		wantOk   bool
	}{
		{"https://acme-corp.atlassian.net/wiki/spaces/ENG/pages/1234567890/API+Guidelines", "1234567890", "acme-corp.atlassian.net", true},
		{"https://co.atlassian.net/wiki/spaces/DEV/pages/123456", "123456", "co.atlassian.net", true},
		{"1234567890", "1234567890", "", true},
		{"not-a-page", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, ok := ParseConfluenceRef(tt.input)
			if ok != tt.wantOk {
				t.Fatalf("ParseConfluenceRef(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if result.PageID != tt.wantID {
				t.Errorf("PageID = %q, want %q", result.PageID, tt.wantID)
			}
			if result.Site != tt.wantSite {
				t.Errorf("Site = %q, want %q", result.Site, tt.wantSite)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd plugins/atlassian-cloud/cli
go test ./internal/urlparse/ -v
```

Expected: FAIL.

**Step 3: Write URL parser implementation**

Write `plugins/atlassian-cloud/cli/internal/urlparse/urlparse.go`:
```go
package urlparse

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	issueKeyRe      = regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`)
	jiraBrowseRe    = regexp.MustCompile(`/browse/([A-Z][A-Z0-9]+-\d+)`)
	confluencePageRe = regexp.MustCompile(`/wiki/spaces/[^/]+/pages/(\d+)`)
	numericRe       = regexp.MustCompile(`^\d+$`)
)

type JiraRef struct {
	IssueKey string
	Site     string
}

type ConfluenceRef struct {
	PageID string
	Site   string
	Space  string
}

func ParseJiraRef(input string) (JiraRef, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return JiraRef{}, false
	}

	// Direct issue key
	if issueKeyRe.MatchString(input) {
		return JiraRef{IssueKey: input}, true
	}

	// URL
	u, err := url.Parse(input)
	if err != nil || u.Host == "" {
		return JiraRef{}, false
	}

	matches := jiraBrowseRe.FindStringSubmatch(u.Path)
	if len(matches) < 2 {
		return JiraRef{}, false
	}

	return JiraRef{
		IssueKey: matches[1],
		Site:     u.Host,
	}, true
}

func ParseConfluenceRef(input string) (ConfluenceRef, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return ConfluenceRef{}, false
	}

	// Numeric page ID
	if numericRe.MatchString(input) {
		return ConfluenceRef{PageID: input}, true
	}

	// URL
	u, err := url.Parse(input)
	if err != nil || u.Host == "" {
		return ConfluenceRef{}, false
	}

	matches := confluencePageRe.FindStringSubmatch(u.Path)
	if len(matches) < 2 {
		return ConfluenceRef{}, false
	}

	// Extract space key from path: /wiki/spaces/{SPACE}/pages/...
	parts := strings.Split(u.Path, "/")
	var space string
	for i, p := range parts {
		if p == "spaces" && i+1 < len(parts) {
			space = parts[i+1]
			break
		}
	}

	return ConfluenceRef{
		PageID: matches[1],
		Site:   u.Host,
		Space:  space,
	}, true
}
```

**Step 4: Run test to verify it passes**

```bash
cd plugins/atlassian-cloud/cli
go test ./internal/urlparse/ -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/urlparse/
git commit -m "feat: add Jira and Confluence URL parser"
```

---

## Task 4: ADF-to-Markdown Converter

**Files:**
- Create: `plugins/atlassian-cloud/cli/internal/output/adf.go`
- Create: `plugins/atlassian-cloud/cli/internal/output/adf_test.go`

**Step 1: Write ADF converter tests**

Write `plugins/atlassian-cloud/cli/internal/output/adf_test.go`:
```go
package output

import "testing"

func TestADFToMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "paragraph with text",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`,
			want:  "Hello world\n",
		},
		{
			name: "heading",
			input: `{"type":"doc","version":1,"content":[{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Title"}]}]}`,
			want:  "## Title\n",
		},
		{
			name: "bold text",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"bold","marks":[{"type":"strong"}]}]}]}`,
			want:  "**bold**\n",
		},
		{
			name: "code block",
			input: `{"type":"doc","version":1,"content":[{"type":"codeBlock","attrs":{"language":"go"},"content":[{"type":"text","text":"fmt.Println()"}]}]}`,
			want:  "```go\nfmt.Println()\n```\n",
		},
		{
			name: "bullet list",
			input: `{"type":"doc","version":1,"content":[{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item 1"}]}]},{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"item 2"}]}]}]}]}`,
			want:  "- item 1\n- item 2\n",
		},
		{
			name: "link",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"click","marks":[{"type":"link","attrs":{"href":"https://example.com"}}]}]}]}`,
			want:  "[click](https://example.com)\n",
		},
		{
			name: "mention",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"mention","attrs":{"text":"@John Doe"}}]}]}`,
			want:  "@John Doe\n",
		},
		{
			name: "blockquote",
			input: `{"type":"doc","version":1,"content":[{"type":"blockquote","content":[{"type":"paragraph","content":[{"type":"text","text":"quoted"}]}]}]}`,
			want:  "> quoted\n",
		},
		{
			name: "inline code",
			input: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"code","marks":[{"type":"code"}]}]}]}`,
			want:  "`code`\n",
		},
		{
			name:  "empty doc",
			input: `{"type":"doc","version":1,"content":[]}`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ADFToMarkdown(tt.input)
			if err != nil {
				t.Fatalf("ADFToMarkdown error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ADFToMarkdown:\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd plugins/atlassian-cloud/cli
go test ./internal/output/ -v
```

Expected: FAIL.

**Step 3: Write ADF converter implementation**

Write `plugins/atlassian-cloud/cli/internal/output/adf.go`:
```go
package output

import (
	"encoding/json"
	"fmt"
	"strings"
)

type adfDoc struct {
	Type    string    `json:"type"`
	Version int       `json:"version"`
	Content []adfNode `json:"content"`
}

type adfNode struct {
	Type    string            `json:"type"`
	Text    string            `json:"text,omitempty"`
	Attrs   map[string]any    `json:"attrs,omitempty"`
	Marks   []adfMark         `json:"marks,omitempty"`
	Content []adfNode         `json:"content,omitempty"`
}

type adfMark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs,omitempty"`
}

func ADFToMarkdown(adfJSON string) (string, error) {
	var doc adfDoc
	if err := json.Unmarshal([]byte(adfJSON), &doc); err != nil {
		return "", fmt.Errorf("invalid ADF JSON: %w", err)
	}

	var sb strings.Builder
	for _, node := range doc.Content {
		convertNode(&sb, node, "")
	}
	return sb.String(), nil
}

func convertNode(sb *strings.Builder, node adfNode, prefix string) {
	switch node.Type {
	case "paragraph":
		sb.WriteString(prefix)
		convertInlineContent(sb, node.Content)
		sb.WriteString("\n")

	case "heading":
		level := 1
		if l, ok := node.Attrs["level"]; ok {
			if f, ok := l.(float64); ok {
				level = int(f)
			}
		}
		sb.WriteString(strings.Repeat("#", level))
		sb.WriteString(" ")
		convertInlineContent(sb, node.Content)
		sb.WriteString("\n")

	case "codeBlock":
		lang := ""
		if l, ok := node.Attrs["language"]; ok {
			if s, ok := l.(string); ok {
				lang = s
			}
		}
		sb.WriteString("```")
		sb.WriteString(lang)
		sb.WriteString("\n")
		for _, child := range node.Content {
			sb.WriteString(child.Text)
		}
		sb.WriteString("\n```\n")

	case "blockquote":
		for _, child := range node.Content {
			convertNode(sb, child, "> ")
		}

	case "bulletList":
		for _, item := range node.Content {
			if item.Type == "listItem" {
				for _, child := range item.Content {
					convertNode(sb, child, "- ")
				}
			}
		}

	case "orderedList":
		for i, item := range node.Content {
			if item.Type == "listItem" {
				for _, child := range item.Content {
					convertNode(sb, child, fmt.Sprintf("%d. ", i+1))
				}
			}
		}

	case "table":
		convertTable(sb, node)

	case "mediaSingle", "mediaGroup":
		for _, child := range node.Content {
			if child.Type == "media" {
				alt := ""
				if a, ok := child.Attrs["alt"]; ok {
					if s, ok := a.(string); ok {
						alt = s
					}
				}
				url := ""
				if u, ok := child.Attrs["url"]; ok {
					if s, ok := u.(string); ok {
						url = s
					}
				}
				if url == "" {
					if id, ok := child.Attrs["id"]; ok {
						if s, ok := id.(string); ok {
							url = fmt.Sprintf("attachment:%s", s)
						}
					}
				}
				sb.WriteString(fmt.Sprintf("![%s](%s)\n", alt, url))
			}
		}

	case "rule":
		sb.WriteString("---\n")

	case "panel":
		panelType := "info"
		if pt, ok := node.Attrs["panelType"]; ok {
			if s, ok := pt.(string); ok {
				panelType = s
			}
		}
		sb.WriteString(fmt.Sprintf("> **%s**\n", strings.ToUpper(panelType)))
		for _, child := range node.Content {
			convertNode(sb, child, "> ")
		}

	case "expand":
		title := "Details"
		if t, ok := node.Attrs["title"]; ok {
			if s, ok := t.(string); ok {
				title = s
			}
		}
		sb.WriteString(fmt.Sprintf("<details><summary>%s</summary>\n\n", title))
		for _, child := range node.Content {
			convertNode(sb, child, "")
		}
		sb.WriteString("</details>\n")

	default:
		// Fallback: render children
		for _, child := range node.Content {
			convertNode(sb, child, prefix)
		}
	}
}

func convertInlineContent(sb *strings.Builder, nodes []adfNode) {
	for _, node := range nodes {
		switch node.Type {
		case "text":
			text := node.Text
			text = applyMarks(text, node.Marks)
			sb.WriteString(text)

		case "mention":
			name := "@unknown"
			if t, ok := node.Attrs["text"]; ok {
				if s, ok := t.(string); ok {
					name = s
				}
			}
			sb.WriteString(name)

		case "emoji":
			if shortName, ok := node.Attrs["shortName"]; ok {
				if s, ok := shortName.(string); ok {
					sb.WriteString(s)
				}
			}

		case "inlineCard":
			url := ""
			if u, ok := node.Attrs["url"]; ok {
				if s, ok := u.(string); ok {
					url = s
				}
			}
			sb.WriteString(fmt.Sprintf("[link](%s)", url))

		case "status":
			text := ""
			if t, ok := node.Attrs["text"]; ok {
				if s, ok := t.(string); ok {
					text = s
				}
			}
			sb.WriteString(fmt.Sprintf("[%s]", text))

		case "hardBreak":
			sb.WriteString("\n")

		default:
			// Recurse into unknown inline nodes
			convertInlineContent(sb, node.Content)
		}
	}
}

func applyMarks(text string, marks []adfMark) string {
	for _, mark := range marks {
		switch mark.Type {
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "*" + text + "*"
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = "~~" + text + "~~"
		case "underline":
			text = "<u>" + text + "</u>"
		case "link":
			if href, ok := mark.Attrs["href"]; ok {
				if s, ok := href.(string); ok {
					text = fmt.Sprintf("[%s](%s)", text, s)
				}
			}
		}
	}
	return text
}

func convertTable(sb *strings.Builder, node adfNode) {
	if len(node.Content) == 0 {
		return
	}

	for rowIdx, row := range node.Content {
		if row.Type != "tableRow" {
			continue
		}

		sb.WriteString("|")
		for _, cell := range row.Content {
			sb.WriteString(" ")
			var cellBuf strings.Builder
			for _, child := range cell.Content {
				convertInlineContent(&cellBuf, child.Content)
			}
			sb.WriteString(strings.TrimSpace(cellBuf.String()))
			sb.WriteString(" |")
		}
		sb.WriteString("\n")

		// Separator after first row (header)
		if rowIdx == 0 {
			sb.WriteString("|")
			for range row.Content {
				sb.WriteString("---|")
			}
			sb.WriteString("\n")
		}
	}
}
```

**Step 4: Run test to verify it passes**

```bash
cd plugins/atlassian-cloud/cli
go test ./internal/output/ -v
```

Expected: PASS.

**Step 5: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/output/
git commit -m "feat: add ADF-to-markdown converter"
```

---

## Task 5: Auth Commands (OAuth2 + API Token)

**Files:**
- Create: `plugins/atlassian-cloud/cli/internal/auth/oauth2.go`
- Create: `plugins/atlassian-cloud/cli/internal/auth/clients.go`
- Create: `plugins/atlassian-cloud/cli/cmd/auth.go`

**Step 1: Write OAuth2 flow handler**

Write `plugins/atlassian-cloud/cli/internal/auth/oauth2.go`:
```go
package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/service/common"
)

const callbackPort = 19872

type OAuthResult struct {
	Token     *common.OAuth2Token
	Resources []*common.AccessibleResource
}

func RunOAuthFlow(ctx context.Context, config *common.OAuth2Config) (*OAuthResult, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			fmt.Fprintf(w, "<html><body><h2>Authorization failed</h2><p>%s</p></body></html>", errMsg)
			errCh <- fmt.Errorf("OAuth error: %s", errMsg)
			return
		}

		fmt.Fprint(w, "<html><body><h2>Authorization successful!</h2><p>You can close this window.</p></body></html>")
		codeCh <- code
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", callbackPort))
	if err != nil {
		return nil, fmt.Errorf("cannot start callback server on port %d: %w", callbackPort, err)
	}

	server := &http.Server{Handler: mux}
	go func() {
		if serveErr := server.Serve(listener); serveErr != http.ErrServerClosed {
			errCh <- serveErr
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// This is a placeholder — actual URL generation uses the go-atlassian OAuth service.
	// The real implementation will use jiraClient.OAuth.GetAuthorizationURL()
	// For now, we construct the URL manually based on Atlassian's OAuth2 3LO spec.
	authURL := fmt.Sprintf(
		"https://auth.atlassian.com/authorize?audience=api.atlassian.com&client_id=%s&scope=%s&redirect_uri=%s&state=atlassian-cloud-cli&response_type=code&prompt=consent",
		config.ClientID,
		joinScopes(config.Scopes),
		fmt.Sprintf("http://localhost:%d/callback", callbackPort),
	)

	fmt.Printf("Opening browser for authorization...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", authURL)
	openBrowser(authURL)

	// Wait for callback or timeout
	select {
	case code := <-codeCh:
		return exchangeCode(ctx, config, code)
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func exchangeCode(ctx context.Context, config *common.OAuth2Config, code string) (*OAuthResult, error) {
	// Use go-atlassian's OAuth service for code exchange
	// We create a temporary Jira client just for the OAuth exchange
	jiraClient, err := createTempOAuthClient(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create OAuth client: %w", err)
	}

	token, err := jiraClient.OAuth.ExchangeAuthorizationCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("cannot exchange authorization code: %w", err)
	}

	resources, err := jiraClient.OAuth.GetAccessibleResources(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("cannot get accessible resources: %w", err)
	}

	return &OAuthResult{Token: token, Resources: resources}, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

func joinScopes(scopes []string) string {
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += "%20"
		}
		result += s
	}
	return result
}
```

**Step 2: Write client factory**

Write `plugins/atlassian-cloud/cli/internal/auth/clients.go`:
```go
package auth

import (
	"fmt"
	"net/http"
	"os"
	"time"

	confluence "github.com/ctreminiom/go-atlassian/v2/confluence"
	confluencev2 "github.com/ctreminiom/go-atlassian/v2/confluence/v2"
	jira "github.com/ctreminiom/go-atlassian/v2/jira/v2"
	"github.com/ctreminiom/go-atlassian/v2/service/common"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/config"
)

const (
	ExitCodeAuthRequired = 2
	tokenExpiryBuffer    = 5 * time.Minute
)

type Clients struct {
	Jira         *jira.Client
	ConfluenceV1 *confluence.Client
	ConfluenceV2 *confluencev2.Client
	SiteURL      string
}

func NewClients(site string) (*Clients, error) {
	cfg, err := config.LoadAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot load auth config: %w", err)
	}

	if site == "" {
		site = cfg.DefaultSite
	}
	if site == "" {
		fmt.Fprintln(os.Stderr, "No site configured. Run: atlassian-cloud auth login")
		os.Exit(ExitCodeAuthRequired)
	}

	siteAuth, ok := cfg.Sites[site]
	if !ok {
		fmt.Fprintf(os.Stderr, "No credentials for site %q. Run: atlassian-cloud auth login\n", site)
		os.Exit(ExitCodeAuthRequired)
	}

	siteURL := fmt.Sprintf("https://%s", site)

	switch siteAuth.Method {
	case "oauth2":
		return newOAuth2Clients(siteURL, siteAuth, cfg, site)
	case "token":
		return newTokenClients(siteURL, siteAuth)
	default:
		return nil, fmt.Errorf("unknown auth method: %s", siteAuth.Method)
	}
}

func newOAuth2Clients(siteURL string, siteAuth config.SiteAuth, cfg *config.AuthConfig, site string) (*Clients, error) {
	// Check token expiry and refresh if needed
	if siteAuth.TokenExpiry != "" {
		expiry, err := time.Parse(time.RFC3339, siteAuth.TokenExpiry)
		if err == nil && time.Now().Add(tokenExpiryBuffer).After(expiry) {
			// Token expired or about to expire — need refresh
			oauthConfig := loadOAuthConfig()
			if oauthConfig == nil || siteAuth.RefreshToken == "" {
				fmt.Fprintln(os.Stderr, "Session expired. Run: atlassian-cloud auth login")
				os.Exit(ExitCodeAuthRequired)
			}

			// Create temp client for refresh
			tempClient, err := createTempOAuthClient(oauthConfig)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Cannot refresh token. Run: atlassian-cloud auth login")
				os.Exit(ExitCodeAuthRequired)
			}

			newToken, err := tempClient.OAuth.RefreshAccessToken(nil, siteAuth.RefreshToken)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Token refresh failed. Run: atlassian-cloud auth login")
				os.Exit(ExitCodeAuthRequired)
			}

			// Update stored config
			siteAuth.AccessToken = newToken.AccessToken
			if newToken.RefreshToken != "" {
				siteAuth.RefreshToken = newToken.RefreshToken
			}
			siteAuth.TokenExpiry = time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second).Format(time.RFC3339)
			cfg.Sites[site] = siteAuth
			config.SaveAuthConfig(cfg)
		}
	}

	jiraClient, err := jira.New(http.DefaultClient, siteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Jira client: %w", err)
	}
	jiraClient.Auth.SetBearerToken(siteAuth.AccessToken)

	confV1Client, err := confluence.New(http.DefaultClient, siteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v1 client: %w", err)
	}
	confV1Client.Auth.SetBearerToken(siteAuth.AccessToken)

	confV2Client, err := confluencev2.New(http.DefaultClient, siteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v2 client: %w", err)
	}
	confV2Client.Auth.SetBearerToken(siteAuth.AccessToken)

	return &Clients{
		Jira:         jiraClient,
		ConfluenceV1: confV1Client,
		ConfluenceV2: confV2Client,
		SiteURL:      siteURL,
	}, nil
}

func newTokenClients(siteURL string, siteAuth config.SiteAuth) (*Clients, error) {
	jiraClient, err := jira.New(http.DefaultClient, siteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Jira client: %w", err)
	}
	jiraClient.Auth.SetBasicAuth(siteAuth.Email, siteAuth.APIToken)

	confV1Client, err := confluence.New(http.DefaultClient, siteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v1 client: %w", err)
	}
	confV1Client.Auth.SetBasicAuth(siteAuth.Email, siteAuth.APIToken)

	confV2Client, err := confluencev2.New(http.DefaultClient, siteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v2 client: %w", err)
	}
	confV2Client.Auth.SetBasicAuth(siteAuth.Email, siteAuth.APIToken)

	return &Clients{
		Jira:         jiraClient,
		ConfluenceV1: confV1Client,
		ConfluenceV2: confV2Client,
		SiteURL:      siteURL,
	}, nil
}

func loadOAuthConfig() *common.OAuth2Config {
	clientID := os.Getenv("ATLASSIAN_CLIENT_ID")
	clientSecret := os.Getenv("ATLASSIAN_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil
	}
	return &common.OAuth2Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  fmt.Sprintf("http://localhost:%d/callback", callbackPort),
		Scopes: []string{
			"read:jira-work", "write:jira-work", "read:jira-user",
			"read:confluence-content.all", "read:confluence-space.summary",
			"offline_access",
		},
	}
}

func createTempOAuthClient(config *common.OAuth2Config) (*jira.Client, error) {
	return jira.New(http.DefaultClient, "https://api.atlassian.com", jira.WithOAuth(config))
}
```

**Step 3: Write auth command**

Write `plugins/atlassian-cloud/cli/cmd/auth.go`:
```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	internalAuth "github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via OAuth2 (opens browser)",
	RunE:  runAuthLogin,
}

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Set API token authentication",
	RunE:  runAuthToken,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE:  runAuthStatus,
}

var (
	tokenEmail string
	tokenValue string
	tokenSite  string
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authTokenCmd)
	authCmd.AddCommand(authStatusCmd)

	authTokenCmd.Flags().StringVar(&tokenEmail, "email", "", "Atlassian account email")
	authTokenCmd.Flags().StringVar(&tokenValue, "token", "", "API token")
	authTokenCmd.Flags().StringVar(&tokenSite, "site", "", "Atlassian site (e.g., company.atlassian.net)")
	authTokenCmd.MarkFlagRequired("email")
	authTokenCmd.MarkFlagRequired("token")
	authTokenCmd.MarkFlagRequired("site")
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	clientID := os.Getenv("ATLASSIAN_CLIENT_ID")
	clientSecret := os.Getenv("ATLASSIAN_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("ATLASSIAN_CLIENT_ID and ATLASSIAN_CLIENT_SECRET must be set.\n" +
			"Create an OAuth app at https://developer.atlassian.com/console/myapps/")
	}

	oauthConfig := internalAuth.LoadOAuthConfigFromEnv()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := internalAuth.RunOAuthFlow(ctx, oauthConfig)
	if err != nil {
		return fmt.Errorf("OAuth flow failed: %w", err)
	}

	if len(result.Resources) == 0 {
		return fmt.Errorf("no accessible Atlassian sites found")
	}

	// Use first resource
	resource := result.Resources[0]
	site := extractHostFromURL(resource.URL)

	cfg, err := config.LoadAuthConfig()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	cfg.DefaultSite = site
	cfg.Sites[site] = config.SiteAuth{
		Method:       "oauth2",
		AccessToken:  result.Token.AccessToken,
		RefreshToken: result.Token.RefreshToken,
		TokenExpiry:  time.Now().Add(time.Duration(result.Token.ExpiresIn) * time.Second).Format(time.RFC3339),
		CloudID:      resource.ID,
		Scopes:       oauthConfig.Scopes,
	}

	if err := config.SaveAuthConfig(cfg); err != nil {
		return fmt.Errorf("cannot save config: %w", err)
	}

	fmt.Printf("Authenticated with %s (%s)\n", resource.Name, site)
	return nil
}

func runAuthToken(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadAuthConfig()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	cfg.DefaultSite = tokenSite
	cfg.Sites[tokenSite] = config.SiteAuth{
		Method:   "token",
		Email:    tokenEmail,
		APIToken: tokenValue,
	}

	if err := config.SaveAuthConfig(cfg); err != nil {
		return fmt.Errorf("cannot save config: %w", err)
	}

	fmt.Printf("API token configured for %s\n", tokenSite)
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadAuthConfig()
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	if len(cfg.Sites) == 0 {
		fmt.Println("Not authenticated. Run: atlassian-cloud auth login")
		return nil
	}

	for site, auth := range cfg.Sites {
		marker := "  "
		if site == cfg.DefaultSite {
			marker = "* "
		}
		fmt.Printf("%s%s [%s]", marker, site, auth.Method)
		if auth.Method == "oauth2" && auth.TokenExpiry != "" {
			expiry, err := time.Parse(time.RFC3339, auth.TokenExpiry)
			if err == nil {
				if time.Now().After(expiry) {
					fmt.Print(" (expired)")
				} else {
					fmt.Printf(" (expires in %s)", time.Until(expiry).Round(time.Minute))
				}
			}
		}
		fmt.Println()
	}

	return nil
}

func extractHostFromURL(rawURL string) string {
	// Simple extraction: remove scheme
	host := rawURL
	for _, prefix := range []string{"https://", "http://"} {
		if len(host) > len(prefix) && host[:len(prefix)] == prefix {
			host = host[len(prefix):]
			break
		}
	}
	// Remove trailing slash
	if len(host) > 0 && host[len(host)-1] == '/' {
		host = host[:len(host)-1]
	}
	return host
}
```

**Step 4: Update auth package to export LoadOAuthConfigFromEnv**

Add to `plugins/atlassian-cloud/cli/internal/auth/oauth2.go` — rename `loadOAuthConfig` to `LoadOAuthConfigFromEnv` and export it. Also in `clients.go`, update the reference.

**Step 5: Build and verify**

```bash
cd plugins/atlassian-cloud/cli
go build -o bin/atlassian-cloud .
./bin/atlassian-cloud auth --help
./bin/atlassian-cloud auth status
```

Expected: `auth` subcommands appear; `auth status` says "Not authenticated."

**Step 6: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/auth/ plugins/atlassian-cloud/cli/cmd/auth.go
git commit -m "feat: add OAuth2 and API token authentication commands"
```

---

## Task 6: Jira Issue Get Command

**Files:**
- Create: `plugins/atlassian-cloud/cli/internal/output/jira.go`
- Create: `plugins/atlassian-cloud/cli/cmd/jira_issue.go`

**Step 1: Write Jira markdown formatter**

Write `plugins/atlassian-cloud/cli/internal/output/jira.go`:
```go
package output

import (
	"fmt"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func FormatIssueSummary(issue *models.IssueSchemeV2) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**%s**: %s\n", issue.Key, issue.Fields.Summary))

	if issue.Fields.Status != nil {
		sb.WriteString(fmt.Sprintf("**Status**: %s", issue.Fields.Status.Name))
	}
	if issue.Fields.IssueType != nil {
		sb.WriteString(fmt.Sprintf(" | **Type**: %s", issue.Fields.IssueType.Name))
	}
	if issue.Fields.Priority != nil {
		sb.WriteString(fmt.Sprintf(" | **Priority**: %s", issue.Fields.Priority.Name))
	}
	sb.WriteString("\n")

	if issue.Fields.Assignee != nil {
		sb.WriteString(fmt.Sprintf("**Assignee**: %s\n", issue.Fields.Assignee.DisplayName))
	} else {
		sb.WriteString("**Assignee**: Unassigned\n")
	}

	if issue.Fields.Reporter != nil {
		sb.WriteString(fmt.Sprintf("**Reporter**: %s\n", issue.Fields.Reporter.DisplayName))
	}

	if issue.Fields.Project != nil {
		sb.WriteString(fmt.Sprintf("**Project**: %s (%s)\n", issue.Fields.Project.Name, issue.Fields.Project.Key))
	}

	if len(issue.Fields.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("**Labels**: %s\n", strings.Join(issue.Fields.Labels, ", ")))
	}

	if issue.Fields.Created != nil {
		sb.WriteString(fmt.Sprintf("**Created**: %s\n", issue.Fields.Created))
	}
	if issue.Fields.Updated != nil {
		sb.WriteString(fmt.Sprintf("**Updated**: %s\n", issue.Fields.Updated))
	}

	return sb.String()
}

func FormatIssueDescription(description string) string {
	if description == "" {
		return "\n*No description*\n"
	}

	// Try ADF parsing first
	md, err := ADFToMarkdown(description)
	if err == nil && md != "" {
		return fmt.Sprintf("\n### Description\n\n%s", md)
	}

	// Fallback: plain text
	return fmt.Sprintf("\n### Description\n\n%s\n", description)
}

func FormatComments(comments []*models.IssueCommentSchemeV2) string {
	if len(comments) == 0 {
		return "\n*No comments*\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n### Comments (%d)\n\n", len(comments)))

	for i, c := range comments {
		author := "Unknown"
		if c.Author != nil {
			author = c.Author.DisplayName
		}
		sb.WriteString(fmt.Sprintf("%d. **%s** (%s):\n", i+1, author, c.Created))

		body := c.Body
		if md, err := ADFToMarkdown(body); err == nil && md != "" {
			body = md
		}
		// Indent comment body
		for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
			sb.WriteString(fmt.Sprintf("   %s\n", line))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func FormatAttachments(attachments []*models.IssueAttachmentScheme) string {
	if len(attachments) == 0 {
		return "\n*No attachments*\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n### Attachments (%d)\n\n", len(attachments)))

	for _, a := range attachments {
		size := formatSize(a.Size)
		sb.WriteString(fmt.Sprintf("- **%s** (%s, %s)\n", a.Filename, size, a.MimeType))
		if a.Content != "" {
			sb.WriteString(fmt.Sprintf("  Download: %s\n", a.Content))
		}
	}

	return sb.String()
}

func FormatSearchResults(issues []*models.IssueSchemeV2, total int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Found %d issues** (showing %d)\n\n", total, len(issues)))

	sb.WriteString("| Key | Summary | Status | Assignee |\n")
	sb.WriteString("|-----|---------|--------|----------|\n")

	for _, issue := range issues {
		assignee := "Unassigned"
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}
		status := ""
		if issue.Fields.Status != nil {
			status = issue.Fields.Status.Name
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			issue.Key,
			truncate(issue.Fields.Summary, 60),
			status,
			assignee,
		))
	}

	return sb.String()
}

func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
```

**Step 2: Write jira issue command**

Write `plugins/atlassian-cloud/cli/cmd/jira_issue.go`:
```go
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/output"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var jiraCmd = &cobra.Command{
	Use:   "jira",
	Short: "Jira operations",
}

var jiraIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue operations",
}

var jiraIssueGetCmd = &cobra.Command{
	Use:   "get <issue-key-or-url>",
	Short: "Get issue details",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraIssueGet,
}

var jiraSearchCmd = &cobra.Command{
	Use:   "search <jql>",
	Short: "Search issues with JQL",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraSearch,
}

var (
	issueDescription bool
	issueComments    bool
	issueAttachments bool
	issueAllFields   bool
	issueFields      string
	searchMax        int
	searchDescription bool
	siteName         string
)

func init() {
	rootCmd.AddCommand(jiraCmd)
	jiraCmd.AddCommand(jiraIssueCmd)
	jiraCmd.AddCommand(jiraSearchCmd)
	jiraIssueCmd.AddCommand(jiraIssueGetCmd)

	// Global site flag
	rootCmd.PersistentFlags().StringVar(&siteName, "site", "", "Atlassian site (overrides default)")

	jiraIssueGetCmd.Flags().BoolVar(&issueDescription, "description", false, "Include description")
	jiraIssueGetCmd.Flags().BoolVar(&issueComments, "comments", false, "Include comments")
	jiraIssueGetCmd.Flags().BoolVar(&issueAttachments, "attachments", false, "Include attachments")
	jiraIssueGetCmd.Flags().BoolVar(&issueAllFields, "all-fields", false, "Include all fields")
	jiraIssueGetCmd.Flags().StringVar(&issueFields, "fields", "", "Comma-separated list of fields")

	jiraSearchCmd.Flags().IntVar(&searchMax, "max", 20, "Maximum results")
	jiraSearchCmd.Flags().BoolVar(&searchDescription, "description", false, "Include descriptions in results")
}

func runJiraIssueGet(cmd *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	site := siteName
	if site == "" && ref.Site != "" {
		site = ref.Site
	}

	clients, err := auth.NewClients(site)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Build fields list
	fields := []string{"summary", "status", "issuetype", "priority", "assignee", "reporter", "project", "labels", "created", "updated"}
	if issueDescription || issueAllFields {
		fields = append(fields, "description")
	}
	if issueComments || issueAllFields {
		fields = append(fields, "comment")
	}
	if issueAttachments || issueAllFields {
		fields = append(fields, "attachment")
	}

	issue, response, err := clients.Jira.Issue.Get(ctx, ref.IssueKey, fields, nil)
	if err != nil {
		if response != nil && response.Code == 404 {
			return fmt.Errorf("issue %s not found", ref.IssueKey)
		}
		if response != nil && response.Code == 401 {
			fmt.Fprintln(os.Stderr, "Authentication failed. Run: atlassian-cloud auth login")
			os.Exit(auth.ExitCodeAuthRequired)
		}
		return fmt.Errorf("cannot get issue: %w", err)
	}

	// Output
	fmt.Print(output.FormatIssueSummary(issue))

	if (issueDescription || issueAllFields) && issue.Fields.Description != "" {
		fmt.Print(output.FormatIssueDescription(issue.Fields.Description))
	}

	if (issueComments || issueAllFields) && issue.Fields.Comment != nil {
		fmt.Print(output.FormatComments(issue.Fields.Comment.Comments))
	}

	if issueAttachments || issueAllFields {
		// Attachments come from a separate field in v2
		// The Issue.Get with "attachment" field should include them in RenderedFields
		// For now, we'd need to fetch separately or use the attachment endpoints
		// This will be refined in testing
		fmt.Println("\n*Use --attachments with a follow-up to list attachment details*")
	}

	return nil
}

func runJiraSearch(cmd *cobra.Command, args []string) error {
	clients, err := auth.NewClients(siteName)
	if err != nil {
		return err
	}

	ctx := context.Background()

	fields := []string{"summary", "status", "assignee"}
	if searchDescription {
		fields = append(fields, "description")
	}

	results, _, err := clients.Jira.Issue.Search.Get(ctx, args[0], fields, nil, 0, searchMax, "")
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	fmt.Print(output.FormatSearchResults(results.Issues, results.Total))

	return nil
}
```

**Step 3: Build and verify**

```bash
cd plugins/atlassian-cloud/cli
go build -o bin/atlassian-cloud .
./bin/atlassian-cloud jira issue get --help
./bin/atlassian-cloud jira search --help
```

Expected: Help output for both commands with correct flags.

**Step 4: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/output/jira.go plugins/atlassian-cloud/cli/cmd/jira_issue.go
git commit -m "feat: add jira issue get and search commands"
```

---

## Task 7: Jira Comments & Fields Commands

**Files:**
- Create: `plugins/atlassian-cloud/cli/cmd/jira_comment.go`
- Create: `plugins/atlassian-cloud/cli/cmd/jira_fields.go`

**Step 1: Write jira comment command**

Write `plugins/atlassian-cloud/cli/cmd/jira_comment.go`:
```go
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/output"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var jiraCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Comment operations",
}

var jiraCommentListCmd = &cobra.Command{
	Use:   "list <issue-key-or-url>",
	Short: "List comments on an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraCommentList,
}

var jiraCommentAddCmd = &cobra.Command{
	Use:   "add <issue-key-or-url>",
	Short: "Add a comment to an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraCommentAdd,
}

var (
	commentBody  string
	commentStdin bool
)

func init() {
	jiraCmd.AddCommand(jiraCommentCmd)
	jiraCommentCmd.AddCommand(jiraCommentListCmd)
	jiraCommentCmd.AddCommand(jiraCommentAddCmd)

	jiraCommentAddCmd.Flags().StringVar(&commentBody, "body", "", "Comment text")
	jiraCommentAddCmd.Flags().BoolVar(&commentStdin, "stdin", false, "Read comment body from stdin")
}

func runJiraCommentList(cmd *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	site := siteName
	if site == "" && ref.Site != "" {
		site = ref.Site
	}

	clients, err := auth.NewClients(site)
	if err != nil {
		return err
	}

	ctx := context.Background()
	comments, _, err := clients.Jira.Issue.Comment.Gets(ctx, ref.IssueKey, "", nil, 0, 50)
	if err != nil {
		return fmt.Errorf("cannot list comments: %w", err)
	}

	fmt.Print(output.FormatComments(comments.Comments))
	return nil
}

func runJiraCommentAdd(cmd *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	body := commentBody
	if commentStdin {
		scanner := bufio.NewScanner(os.Stdin)
		var lines string
		for scanner.Scan() {
			lines += scanner.Text() + "\n"
		}
		body = lines
	}

	if body == "" {
		return fmt.Errorf("comment body required (use --body or --stdin)")
	}

	site := siteName
	if site == "" && ref.Site != "" {
		site = ref.Site
	}

	clients, err := auth.NewClients(site)
	if err != nil {
		return err
	}

	ctx := context.Background()
	payload := &models.CommentPayloadSchemeV2{Body: body}
	comment, _, err := clients.Jira.Issue.Comment.Add(ctx, ref.IssueKey, payload, nil)
	if err != nil {
		return fmt.Errorf("cannot add comment: %w", err)
	}

	fmt.Printf("Comment added (ID: %s)\n", comment.ID)
	return nil
}
```

**Step 2: Write jira fields command**

Write `plugins/atlassian-cloud/cli/cmd/jira_fields.go`:
```go
package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
)

var jiraFieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "Field operations",
}

var jiraFieldsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available fields",
	RunE:  runJiraFieldsList,
}

func init() {
	jiraCmd.AddCommand(jiraFieldsCmd)
	jiraFieldsCmd.AddCommand(jiraFieldsListCmd)
}

func runJiraFieldsList(cmd *cobra.Command, args []string) error {
	clients, err := auth.NewClients(siteName)
	if err != nil {
		return err
	}

	ctx := context.Background()
	fields, _, err := clients.Jira.Issue.Field.Gets(ctx)
	if err != nil {
		return fmt.Errorf("cannot list fields: %w", err)
	}

	fmt.Printf("| %-30s | %-25s | %-10s |\n", "ID", "Name", "Custom")
	fmt.Printf("|%s|%s|%s|\n", "--------------------------------", "---------------------------", "------------")

	for _, f := range fields {
		fmt.Printf("| %-30s | %-25s | %-10v |\n", f.ID, f.Name, f.Custom)
	}

	return nil
}
```

Note: The `Field.Gets()` method may need adjustment based on the actual go-atlassian API. The implementer should verify the exact field service method available on the Jira v2 client (`clients.Jira.Issue.Field` or similar) and adjust accordingly.

**Step 3: Build and verify**

```bash
cd plugins/atlassian-cloud/cli
go build -o bin/atlassian-cloud .
./bin/atlassian-cloud jira comment --help
./bin/atlassian-cloud jira fields --help
```

Expected: Help output for all subcommands.

**Step 4: Commit**

```bash
git add plugins/atlassian-cloud/cli/cmd/jira_comment.go plugins/atlassian-cloud/cli/cmd/jira_fields.go
git commit -m "feat: add jira comment and fields commands"
```

---

## Task 8: Confluence Page Command

**Files:**
- Create: `plugins/atlassian-cloud/cli/internal/output/confluence.go`
- Create: `plugins/atlassian-cloud/cli/cmd/confluence_page.go`

**Step 1: Write Confluence markdown formatter**

Write `plugins/atlassian-cloud/cli/internal/output/confluence.go`:
```go
package output

import (
	"fmt"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func FormatPageSummary(page *models.PageScheme) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**%s**\n", page.Title))
	sb.WriteString(fmt.Sprintf("**Page ID**: %s | **Space**: %s\n", page.ID, page.SpaceID))

	if page.Version != nil {
		sb.WriteString(fmt.Sprintf("**Version**: %d\n", page.Version.Number))
	}
	if page.CreatedAt != "" {
		sb.WriteString(fmt.Sprintf("**Created**: %s\n", page.CreatedAt))
	}
	sb.WriteString(fmt.Sprintf("**Status**: %s\n", page.Status))

	return sb.String()
}

func FormatPageBody(page *models.PageScheme) string {
	if page.Body == nil {
		return "\n*No body content*\n"
	}

	var raw string
	if page.Body.AtlasDocFormat != nil && page.Body.AtlasDocFormat.Value != "" {
		raw = page.Body.AtlasDocFormat.Value
	} else if page.Body.Storage != nil && page.Body.Storage.Value != "" {
		raw = page.Body.Storage.Value
	}

	if raw == "" {
		return "\n*Empty page*\n"
	}

	// Try ADF conversion
	md, err := ADFToMarkdown(raw)
	if err == nil && md != "" {
		return fmt.Sprintf("\n### Content\n\n%s", md)
	}

	// Fallback: raw content (may be storage format HTML)
	return fmt.Sprintf("\n### Content\n\n%s\n", raw)
}

func FormatConfluenceAttachments(attachments []*models.AttachmentScheme) string {
	if len(attachments) == 0 {
		return "\n*No attachments*\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n### Attachments (%d)\n\n", len(attachments)))

	for _, a := range attachments {
		size := formatSize(a.FileSize)
		sb.WriteString(fmt.Sprintf("- **%s** (%s, %s)\n", a.Title, size, a.MediaType))
		if a.DownloadLink != "" {
			sb.WriteString(fmt.Sprintf("  Download: %s\n", a.DownloadLink))
		}
	}

	return sb.String()
}

func FormatSearchResultsConfluence(results []*models.SearchResultScheme, total int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Found %d results** (showing %d)\n\n", total, len(results)))

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("- **%s** — %s\n", r.Title, r.Excerpt))
		if r.URL != "" {
			sb.WriteString(fmt.Sprintf("  URL: %s\n", r.URL))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
```

**Step 2: Write confluence page command**

Write `plugins/atlassian-cloud/cli/cmd/confluence_page.go`:
```go
package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/output"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var confluenceCmd = &cobra.Command{
	Use:   "confluence",
	Short: "Confluence operations",
}

var confluencePageCmd = &cobra.Command{
	Use:   "page",
	Short: "Page operations",
}

var confluencePageGetCmd = &cobra.Command{
	Use:   "get <page-id-or-url>",
	Short: "Get page details",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfluencePageGet,
}

var confluenceSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search Confluence pages using CQL",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfluenceSearch,
}

var (
	pageBody        bool
	pageAttachments bool
	searchSpace     string
	searchMaxConf   int
)

func init() {
	rootCmd.AddCommand(confluenceCmd)
	confluenceCmd.AddCommand(confluencePageCmd)
	confluenceCmd.AddCommand(confluenceSearchCmd)
	confluencePageCmd.AddCommand(confluencePageGetCmd)

	confluencePageGetCmd.Flags().BoolVar(&pageBody, "body", false, "Include page body content")
	confluencePageGetCmd.Flags().BoolVar(&pageAttachments, "attachments", false, "Include attachments list")

	confluenceSearchCmd.Flags().StringVar(&searchSpace, "space", "", "Limit search to space key")
	confluenceSearchCmd.Flags().IntVar(&searchMaxConf, "max", 10, "Maximum results")
}

func runConfluencePageGet(cmd *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseConfluenceRef(args[0])
	if !ok {
		return fmt.Errorf("invalid page reference: %s", args[0])
	}

	site := siteName
	if site == "" && ref.Site != "" {
		site = ref.Site
	}

	clients, err := auth.NewClients(site)
	if err != nil {
		return err
	}

	ctx := context.Background()

	pageID, err := strconv.Atoi(ref.PageID)
	if err != nil {
		return fmt.Errorf("invalid page ID: %s", ref.PageID)
	}

	bodyFormat := ""
	if pageBody {
		bodyFormat = "atlas_doc_format"
	}

	page, _, err := clients.ConfluenceV2.Page.Get(ctx, pageID, bodyFormat, false, 0)
	if err != nil {
		return fmt.Errorf("cannot get page: %w", err)
	}

	fmt.Print(output.FormatPageSummary(page))

	if pageBody {
		fmt.Print(output.FormatPageBody(page))
	}

	if pageAttachments {
		attachments, _, err := clients.ConfluenceV2.Attachment.Gets(ctx, pageID, "pages", nil, "", 50)
		if err != nil {
			fmt.Printf("\n*Cannot load attachments: %v*\n", err)
		} else {
			fmt.Print(output.FormatConfluenceAttachments(attachments.Results))
		}
	}

	return nil
}

func runConfluenceSearch(cmd *cobra.Command, args []string) error {
	clients, err := auth.NewClients(siteName)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Build CQL query
	cql := fmt.Sprintf("type = page AND text ~ \"%s\"", args[0])
	if searchSpace != "" {
		cql = fmt.Sprintf("type = page AND space = \"%s\" AND text ~ \"%s\"", searchSpace, args[0])
	}

	results, _, err := clients.ConfluenceV1.Search.Content(ctx, cql, &models.SearchContentOptions{
		Limit: searchMaxConf,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	fmt.Print(output.FormatSearchResultsConfluence(results.Results, results.TotalSize))
	return nil
}
```

**Step 3: Build and verify**

```bash
cd plugins/atlassian-cloud/cli
go build -o bin/atlassian-cloud .
./bin/atlassian-cloud confluence page get --help
./bin/atlassian-cloud confluence search --help
```

Expected: Help output for confluence commands.

**Step 4: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/output/confluence.go plugins/atlassian-cloud/cli/cmd/confluence_page.go
git commit -m "feat: add confluence page get and search commands"
```

---

## Task 9: Ensure-Binary Script & Skills

**Files:**
- Create: `plugins/atlassian-cloud/skills/atlassian-jira/scripts/ensure-binary.sh`
- Create: `plugins/atlassian-cloud/skills/atlassian-jira/SKILL.md`
- Create: `plugins/atlassian-cloud/skills/atlassian-confluence/SKILL.md`

**Step 1: Write ensure-binary.sh**

Write `plugins/atlassian-cloud/skills/atlassian-jira/scripts/ensure-binary.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail

CLI_DIR="${CLAUDE_PLUGIN_ROOT}/cli"
BIN_PATH="${CLI_DIR}/bin/atlassian-cloud"

# Check if binary exists and is executable
if [[ -x "$BIN_PATH" ]]; then
    exit 0
fi

echo "Building atlassian-cloud CLI..." >&2

# Ensure Go is available
if ! command -v go &>/dev/null; then
    if command -v mise &>/dev/null; then
        echo "Installing Go via mise..." >&2
        mise install go@latest
        eval "$(mise env)"
    else
        echo "ERROR: Go is not installed. Install Go or mise first." >&2
        echo "  Option 1: Install mise (https://mise.jdx.dev)" >&2
        echo "  Option 2: Install Go directly (https://go.dev/dl/)" >&2
        exit 1
    fi
fi

mkdir -p "${CLI_DIR}/bin"
cd "$CLI_DIR"
go build -o bin/atlassian-cloud .

echo "atlassian-cloud CLI built successfully" >&2
```

Make executable:
```bash
chmod +x plugins/atlassian-cloud/skills/atlassian-jira/scripts/ensure-binary.sh
```

**Step 2: Write atlassian-jira SKILL.md**

Write `plugins/atlassian-cloud/skills/atlassian-jira/SKILL.md`:
```markdown
---
name: atlassian-jira
description: Use when the user asks to "fetch JIRA issue", "get ticket", "show DEV-123", "look up issue", "search jira", "find tickets", "comment on ticket", "add comment to issue", or pastes a JIRA URL like "https://company.atlassian.net/browse/KEY-123". Also triggers on bare issue keys like "DEV-123" or "PROJ-456" in the user's message.
version: 0.1.0
---

# Jira Issue Operations via atlassian-cloud CLI

Access Jira Cloud issues, comments, and search with progressive disclosure for context efficiency.

## Prerequisites

Ensure the CLI binary is built:

```bash
${CLAUDE_PLUGIN_ROOT}/skills/atlassian-jira/scripts/ensure-binary.sh
```

If this fails, guide the user to install Go (`mise install go@latest` or https://go.dev/dl/).

Set the CLI path:
```bash
ATLASSIAN_CLI="${CLAUDE_PLUGIN_ROOT}/cli/bin/atlassian-cloud"
```

## Authentication

Check auth status first:
```bash
$ATLASSIAN_CLI auth status
```

If not authenticated, guide the user:

**Option A: OAuth2 (recommended)**
Requires `ATLASSIAN_CLIENT_ID` and `ATLASSIAN_CLIENT_SECRET` environment variables.
```bash
$ATLASSIAN_CLI auth login
```

**Option B: API Token**
```bash
$ATLASSIAN_CLI auth token --email user@company.com --token YOUR_TOKEN --site company.atlassian.net
```
Get a token at: https://id.atlassian.com/manage-profile/security/api-tokens

If any command exits with code 2, authentication has expired. Tell the user to run `auth login` again.

## Extracting Issue Keys

| Input | Extracted |
|-------|-----------|
| `DEV-123` | DEV-123 |
| `https://acme-corp.atlassian.net/browse/ACME-5136` | ACME-5136 (site: acme-corp.atlassian.net) |
| `https://co.atlassian.net/browse/PROJ-456?focusedCommentId=123` | PROJ-456 |
| "ticket DEV-789" | DEV-789 |

Pattern: `[A-Z][A-Z0-9]+-\d+`

When a URL includes the site hostname, pass it with `--site`:
```bash
$ATLASSIAN_CLI --site acme-corp.atlassian.net jira issue get ACME-5136
```

## Progressive Disclosure — ALWAYS Start at Level 1

### Level 1: Summary (default — use this first)

```bash
$ATLASSIAN_CLI jira issue get KEY-123
```

Returns: Key, Summary, Status, Type, Priority, Assignee, Reporter, Project, Labels, Created, Updated.

**This is usually sufficient.** Only escalate if the user asks for more detail.

### Level 2: + Description

```bash
$ATLASSIAN_CLI jira issue get KEY-123 --description
```

Adds the full issue description (ADF converted to markdown).

### Level 3: + Comments

```bash
$ATLASSIAN_CLI jira issue get KEY-123 --description --comments
```

Adds all comments with author and timestamp.

### Level 4: + Attachments

```bash
$ATLASSIAN_CLI jira issue get KEY-123 --description --comments --attachments
```

Adds attachment list with filenames, sizes, and download URLs.

### Full context

```bash
$ATLASSIAN_CLI jira issue get KEY-123 --all-fields
```

## Searching Issues

```bash
# Basic JQL search
$ATLASSIAN_CLI jira search "project = DEV AND status = Open" --max 20

# Text search
$ATLASSIAN_CLI jira search "text ~ 'deployment error'" --max 10

# With descriptions
$ATLASSIAN_CLI jira search "assignee = currentUser() AND status != Done" --max 15 --description
```

Returns a markdown table: Key | Summary | Status | Assignee.

## Comments

### List comments
```bash
$ATLASSIAN_CLI jira comment list KEY-123
```

### Add a comment
```bash
$ATLASSIAN_CLI jira comment add KEY-123 --body "Fix deployed in v2.3.1"
```

For longer comments:
```bash
echo "Detailed analysis of the issue..." | $ATLASSIAN_CLI jira comment add KEY-123 --stdin
```

## Custom Fields

### Discover available fields
```bash
$ATLASSIAN_CLI jira fields list
```

### Request specific fields
```bash
$ATLASSIAN_CLI jira issue get KEY-123 --fields "Story Points,Sprint,customfield_10001"
```

## Presenting Results

### For simple lookups

Present the CLI output directly — it's already formatted as markdown.

### For search results

The output is a markdown table. Present it as-is or summarize if the user needs a quick answer.

### For full context requests

Structure clearly with headers:
```
## KEY-123: Issue Summary

**Status**: In Progress | **Type**: Story | **Priority**: High

### Description
[markdown content]

### Comments (3)
1. **Author** (date): text
2. ...

### Attachments (2)
- file.png (245 KB)
```

## Error Handling

| Error | Action |
|-------|--------|
| Exit code 2 | Auth expired → `$ATLASSIAN_CLI auth login` |
| "issue not found" | Check key format, verify permissions |
| "cannot connect" | Check network, verify site URL |
| Build failure | Ensure Go is installed: `go version` |

## Tips

1. **Always start at Level 1** — escalate only when more detail is needed
2. **Parse URLs automatically** — the CLI handles both URLs and bare keys
3. **Use --site for cross-site** — when URL contains a different site than default
4. **Search before fetching** — use JQL to find relevant issues first
5. **Comments are paginated** — last 50 shown by default
```

**Step 3: Write atlassian-confluence SKILL.md**

Write `plugins/atlassian-cloud/skills/atlassian-confluence/SKILL.md`:
```markdown
---
name: atlassian-confluence
description: Use when the user asks to "read confluence page", "get wiki page", "search confluence", "find in confluence", "look up documentation", or pastes a Confluence URL like "https://company.atlassian.net/wiki/spaces/SPACE/pages/123456/Page+Title". Also triggers when the user mentions Confluence page IDs or asks about company documentation on Confluence.
version: 0.1.0
---

# Confluence Page Operations via atlassian-cloud CLI

Read Confluence Cloud pages and search content with progressive disclosure for context efficiency.

## Prerequisites

Ensure the CLI binary is built:

```bash
${CLAUDE_PLUGIN_ROOT}/../atlassian-jira/scripts/ensure-binary.sh
```

Set the CLI path:
```bash
ATLASSIAN_CLI="${CLAUDE_PLUGIN_ROOT}/../atlassian-jira/../../../cli/bin/atlassian-cloud"
```

Note: The binary is shared with the atlassian-jira skill. A more robust path:
```bash
ATLASSIAN_CLI="$(dirname "$(dirname "$CLAUDE_PLUGIN_ROOT")")/cli/bin/atlassian-cloud"
```

## Authentication

Same as Jira — check with `$ATLASSIAN_CLI auth status`. See the atlassian-jira skill for setup instructions.

## Extracting Page References

| Input | Extracted |
|-------|-----------|
| `https://acme-corp.atlassian.net/wiki/spaces/ENG/pages/1234567890/API+Guidelines` | Page ID: 1234567890 (site: acme-corp.atlassian.net) |
| `https://co.atlassian.net/wiki/spaces/DEV/pages/123456` | Page ID: 123456 |
| `1234567890` | Page ID: 1234567890 |

When a URL includes the site hostname, pass it with `--site`:
```bash
$ATLASSIAN_CLI --site acme-corp.atlassian.net confluence page get 1234567890
```

## Progressive Disclosure — ALWAYS Start at Level 1

### Level 1: Summary (default)

```bash
$ATLASSIAN_CLI confluence page get <page-id-or-url>
```

Returns: Title, Page ID, Space, Version, Created date, Status.

### Level 2: + Body

```bash
$ATLASSIAN_CLI confluence page get <page-id-or-url> --body
```

Adds full page content (ADF converted to markdown).

### Level 3: + Attachments

```bash
$ATLASSIAN_CLI confluence page get <page-id-or-url> --body --attachments
```

Adds attachment list.

## Searching Pages

```bash
# Full-text search
$ATLASSIAN_CLI confluence search "deployment runbook" --max 10

# Search within a specific space
$ATLASSIAN_CLI confluence search "API documentation" --space WS --max 10
```

Returns: Title, excerpt, URL for each result.

## Presenting Results

### For page summaries
Present the CLI output directly.

### For full page content
Structure with headers:
```
## Page Title

**Space**: WS | **Version**: 4 | **Updated**: 2026-02-19

### Content
[markdown-converted page body]

### Attachments (3)
- diagram.png (1.2 MB)
- spec.pdf (450 KB)
```

## Error Handling

| Error | Action |
|-------|--------|
| Exit code 2 | Auth expired → `$ATLASSIAN_CLI auth login` |
| "page not found" | Verify page ID, check permissions |
| "invalid page ID" | Ensure numeric ID extracted from URL |

## Tips

1. **Always start at Level 1** — page bodies can be very large
2. **Parse URLs automatically** — the CLI handles both URLs and numeric IDs
3. **Use --space for targeted search** — narrows results significantly
4. **Search returns excerpts** — use page get for full content
```

**Step 4: Commit**

```bash
git add plugins/atlassian-cloud/skills/
git commit -m "feat: add Jira and Confluence skills with ensure-binary script"
```

---

## Task 10: README & Reference Docs

**Files:**
- Create: `plugins/atlassian-cloud/README.md`
- Create: `plugins/atlassian-cloud/references/cli-commands.md`

**Step 1: Write README**

Write `plugins/atlassian-cloud/README.md`:
```markdown
# atlassian-cloud

Context-efficient Jira and Confluence Cloud access for Claude Code via a Go CLI built on [go-atlassian](https://github.com/ctreminiom/go-atlassian).

## Features

- **Jira**: Issue lookup (by key or URL), JQL search, comments (read & write), custom field discovery
- **Confluence**: Page reading (by ID or URL), full-text search with CQL
- **Authentication**: OAuth2 3LO with automatic token renewal, API token fallback
- **Progressive disclosure**: Compact summaries by default, escalate to full details on demand
- **ADF conversion**: Atlassian Document Format converted to clean markdown
- **URL recognition**: Automatically parses `*.atlassian.net` URLs

## Requirements

- Go 1.21+ (or [mise](https://mise.jdx.dev) to install it automatically)
- Atlassian Cloud account

## Setup

### 1. Build

The CLI builds automatically on first skill invocation. Or build manually:

```bash
cd plugins/atlassian-cloud/cli
go build -o bin/atlassian-cloud .
```

### 2. Authenticate

**OAuth2 (recommended):**

1. Create an OAuth app at https://developer.atlassian.com/console/myapps/
2. Set callback URL to `http://localhost:19872/callback`
3. Add required scopes: `read:jira-work`, `write:jira-work`, `read:jira-user`, `read:confluence-content.all`, `read:confluence-space.summary`, `offline_access`
4. Set environment variables:
   ```bash
   export ATLASSIAN_CLIENT_ID=your-client-id
   export ATLASSIAN_CLIENT_SECRET=your-client-secret
   ```
5. Run: `atlassian-cloud auth login`

**API Token (simpler):**

1. Generate token at https://id.atlassian.com/manage-profile/security/api-tokens
2. Run: `atlassian-cloud auth token --email you@company.com --token YOUR_TOKEN --site company.atlassian.net`

## Usage

### Jira

```bash
# Get issue summary
atlassian-cloud jira issue get DEV-123

# Get issue with description and comments
atlassian-cloud jira issue get DEV-123 --description --comments

# Search
atlassian-cloud jira search "project = DEV AND status = Open"

# Add comment
atlassian-cloud jira comment add DEV-123 --body "Fix deployed"
```

### Confluence

```bash
# Get page summary
atlassian-cloud confluence page get https://co.atlassian.net/wiki/spaces/ENG/pages/123456/Page

# Get full page content
atlassian-cloud confluence page get 123456 --body

# Search
atlassian-cloud confluence search "deployment guide" --space WS
```

## Security

- Credentials stored in `~/.config/atlassian-cloud/auth.json` with `0600` permissions
- Config directory has `0700` permissions
- OAuth2 tokens automatically refresh before expiry
- API tokens stored encrypted-at-rest (filesystem permissions)
```

**Step 2: Write CLI reference**

Write `plugins/atlassian-cloud/references/cli-commands.md`:
```markdown
# atlassian-cloud CLI Reference

## Global Flags

| Flag | Description |
|------|-------------|
| `--site` | Atlassian site hostname (overrides default) |

## Auth Commands

| Command | Description |
|---------|-------------|
| `auth login` | OAuth2 browser flow |
| `auth token --email --token --site` | Set API token |
| `auth status` | Show auth status |

## Jira Commands

| Command | Description |
|---------|-------------|
| `jira issue get <key\|url>` | Get issue details |
| `jira search <jql>` | Search with JQL |
| `jira comment list <key\|url>` | List comments |
| `jira comment add <key\|url>` | Add comment |
| `jira fields list` | List available fields |

### Issue Get Flags

| Flag | Description |
|------|-------------|
| `--description` | Include description |
| `--comments` | Include comments |
| `--attachments` | Include attachment list |
| `--all-fields` | Include everything |
| `--fields` | Comma-separated field names |

### Search Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--max` | 20 | Max results |
| `--description` | false | Include descriptions |

### Comment Add Flags

| Flag | Description |
|------|-------------|
| `--body` | Comment text |
| `--stdin` | Read from stdin |

## Confluence Commands

| Command | Description |
|---------|-------------|
| `confluence page get <id\|url>` | Get page details |
| `confluence search <query>` | Search with CQL |

### Page Get Flags

| Flag | Description |
|------|-------------|
| `--body` | Include page body |
| `--attachments` | Include attachments |

### Search Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--space` | (all) | Limit to space key |
| `--max` | 10 | Max results |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication required |
```

**Step 3: Commit**

```bash
git add plugins/atlassian-cloud/README.md plugins/atlassian-cloud/references/
git commit -m "docs: add README and CLI reference for atlassian-cloud"
```

---

## Task 11: Integration Testing

**Files:**
- None new — testing the build and CLI

**Step 1: Full build**

```bash
cd plugins/atlassian-cloud/cli
go build -o bin/atlassian-cloud .
```

Expected: Clean build, no errors.

**Step 2: Run all unit tests**

```bash
cd plugins/atlassian-cloud/cli
go test ./... -v
```

Expected: All tests pass.

**Step 3: Verify CLI help tree**

```bash
./bin/atlassian-cloud --help
./bin/atlassian-cloud auth --help
./bin/atlassian-cloud jira --help
./bin/atlassian-cloud jira issue get --help
./bin/atlassian-cloud jira search --help
./bin/atlassian-cloud jira comment --help
./bin/atlassian-cloud jira fields --help
./bin/atlassian-cloud confluence --help
./bin/atlassian-cloud confluence page get --help
./bin/atlassian-cloud confluence search --help
```

Expected: All commands show proper help with flags.

**Step 4: Test auth status (no auth configured)**

```bash
./bin/atlassian-cloud auth status
```

Expected: "Not authenticated. Run: atlassian-cloud auth login"

**Step 5: Test error handling for unauthenticated access**

```bash
./bin/atlassian-cloud jira issue get DEV-123 2>&1; echo "Exit: $?"
```

Expected: Exit code 2, message about running `auth login`.

**Step 6: Commit any fixes**

If any tests failed or builds broke, fix and commit:
```bash
git add -A plugins/atlassian-cloud/
git commit -m "fix: resolve build/test issues found in integration testing"
```

---

## Task 12: Manual End-to-End Test with Real Credentials

**This task requires real Atlassian credentials. Skip if not available.**

**Step 1: Set up API token auth**

```bash
./bin/atlassian-cloud auth token --email YOUR_EMAIL --token YOUR_TOKEN --site YOUR_SITE.atlassian.net
./bin/atlassian-cloud auth status
```

**Step 2: Test Jira issue get**

```bash
./bin/atlassian-cloud jira issue get KNOWN-ISSUE-KEY
./bin/atlassian-cloud jira issue get KNOWN-ISSUE-KEY --description
./bin/atlassian-cloud jira issue get KNOWN-ISSUE-KEY --description --comments
```

**Step 3: Test Jira search**

```bash
./bin/atlassian-cloud jira search "project = YOUR_PROJECT ORDER BY created DESC" --max 5
```

**Step 4: Test Confluence**

```bash
./bin/atlassian-cloud confluence search "test" --max 5
./bin/atlassian-cloud confluence page get KNOWN_PAGE_ID --body
```

**Step 5: Fix any issues found and commit**

```bash
git add -A plugins/atlassian-cloud/
git commit -m "fix: resolve issues found in end-to-end testing"
```

---

## Summary

| Task | Description | Depends On |
|------|-------------|------------|
| 1 | Scaffold plugin + Go module | — |
| 2 | Config & auth storage | 1 |
| 3 | URL parser | 1 |
| 4 | ADF-to-markdown converter | 1 |
| 5 | Auth commands (OAuth2 + token) | 2 |
| 6 | Jira issue get & search | 3, 4, 5 |
| 7 | Jira comments & fields | 5, 6 |
| 8 | Confluence page get & search | 3, 4, 5 |
| 9 | Skills + ensure-binary script | 6, 7, 8 |
| 10 | README & reference docs | 9 |
| 11 | Integration testing | 10 |
| 12 | End-to-end testing | 11 |

Tasks 2, 3, 4 can be done in parallel after Task 1.
Tasks 6, 7, 8 depend on 5 but 7 and 8 can be parallelized.
