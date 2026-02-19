# Attachment Download Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `jira attachment download` and `confluence attachment download` CLI commands to download attachments to local files.

**Architecture:** Shared download helper in `internal/download/` uses an authenticated `*http.Client` (exposed via `auth.Clients`) to GET attachment URLs and write to disk. Each command fetches attachment metadata first, matches by filename or `--all`, then downloads. Default output is OS temp dir; `--output-dir` saves with original filenames.

**Tech Stack:** Go 1.25.6, Cobra CLI, go-atlassian/v2 API library, net/http

---

### Task 1: Expose authenticated HTTP client from auth.Clients

**Files:**
- Modify: `plugins/atlassian-cloud/cli/internal/auth/clients.go`

**Step 1: Add HTTPClient field to Clients struct**

In `plugins/atlassian-cloud/cli/internal/auth/clients.go`, add a new field to the `Clients` struct and a `ConfluenceBaseURL` field (needed for constructing full Confluence download URLs):

```go
type Clients struct {
	Jira             *jira.Client
	ConfluenceV1     *confluence.Client
	ConfluenceV2     *confluencev2.Client
	JiraBaseURL      string
	ConfluenceBaseURL string
	HTTPClient       *http.Client
}
```

**Step 2: Create authenticated HTTP client in buildClients**

Modify `buildClients` to create an `*http.Client` with an auth transport and populate both new fields:

```go
func buildClients(jiraURL, confluenceURL string, configureAuth func(common.Authentication)) (*Clients, error) {
	jiraClient, err := jira.New(http.DefaultClient, jiraURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Jira client: %w", err)
	}
	configureAuth(jiraClient.Auth)

	confV1, err := confluence.New(http.DefaultClient, confluenceURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v1 client: %w", err)
	}
	configureAuth(confV1.Auth)

	confV2, err := confluencev2.New(http.DefaultClient, confluenceURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v2 client: %w", err)
	}
	configureAuth(confV2.Auth)

	// Build an authenticated HTTP client that mirrors the auth config.
	// Extract auth headers from the Jira client (same credentials for all services).
	authHTTP := &http.Client{
		Transport: jiraClient.Auth.GetTransport(),
	}

	return &Clients{
		Jira:              jiraClient,
		ConfluenceV1:      confV1,
		ConfluenceV2:      confV2,
		JiraBaseURL:       jiraURL,
		ConfluenceBaseURL: confluenceURL,
		HTTPClient:        authHTTP,
	}, nil
}
```

**Important:** The `go-atlassian` library's `Authentication` interface may not expose `GetTransport()` directly. Check what's available. If not, we need a different approach — see step 2b.

**Step 2b (alternative): Use a custom roundTripper**

If `GetTransport()` isn't available, pass the auth config into `buildClients` and build the HTTP client from the raw credentials:

Change `buildClients` signature to also accept auth details:

```go
func buildClients(jiraURL, confluenceURL string, configureAuth func(common.Authentication), httpClient *http.Client) (*Clients, error) {
```

Then in `newOAuth2Clients`, create:
```go
httpClient := &http.Client{
	Transport: &bearerTransport{token: accessToken},
}
```

And in `newTokenClients`:
```go
httpClient := &http.Client{
	Transport: &basicAuthTransport{email: siteAuth.Email, token: siteAuth.APIToken},
}
```

Define the transports in the same file:

```go
type bearerTransport struct {
	token string
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}

type basicAuthTransport struct {
	email string
	token string
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.SetBasicAuth(t.email, t.token)
	return http.DefaultTransport.RoundTrip(req)
}
```

**Step 3: Verify build**

Run:
```bash
cd plugins/atlassian-cloud/cli && go build ./...
```
Expected: successful build

**Step 4: Run existing tests**

Run:
```bash
cd plugins/atlassian-cloud/cli && go test ./...
```
Expected: all tests pass (no behavior change)

**Step 5: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/auth/clients.go
git commit -m "feat: expose authenticated HTTP client and Confluence base URL from auth.Clients"
```

---

### Task 2: Create the download helper package

**Files:**
- Create: `plugins/atlassian-cloud/cli/internal/download/download.go`
- Create: `plugins/atlassian-cloud/cli/internal/download/download_test.go`

**Step 1: Write the test**

Create `plugins/atlassian-cloud/cli/internal/download/download_test.go`:

```go
package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestToFile_success(t *testing.T) {
	content := "hello world"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(content))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.txt")
	err := ToFile(context.Background(), srv.Client(), srv.URL, dest)
	if err != nil {
		t.Fatalf("ToFile: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestToFile_httpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.txt")
	err := ToFile(context.Background(), srv.Client(), srv.URL, dest)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestToFile_401returnsAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.txt")
	err := ToFile(context.Background(), srv.Client(), srv.URL, dest)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	// Check that it's an AuthRequiredError
	if _, ok := err.(*AuthError); !ok {
		t.Errorf("expected *AuthError, got %T", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd plugins/atlassian-cloud/cli && go test ./internal/download/ -v
```
Expected: compilation failure — package/types don't exist yet

**Step 3: Write the implementation**

Create `plugins/atlassian-cloud/cli/internal/download/download.go`:

```go
package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

// AuthError indicates an authentication failure during download.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// ToFile downloads the content at url using httpClient and writes it to destPath.
// The parent directory of destPath must exist.
// Returns an *AuthError on 401 responses, or a generic error on other non-2xx status codes.
func ToFile(ctx context.Context, httpClient *http.Client, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return &AuthError{Message: "authentication failed; run: atlassian-cloud auth login"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
```

**Step 4: Run tests**

Run:
```bash
cd plugins/atlassian-cloud/cli && go test ./internal/download/ -v
```
Expected: all 3 tests pass

**Step 5: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/download/
git commit -m "feat: add download helper package for attachment file downloads"
```

---

### Task 3: Add `jira attachment download` command

**Files:**
- Create: `plugins/atlassian-cloud/cli/cmd/jira_attachment.go`
- Modify: `plugins/atlassian-cloud/cli/cmd/jira_issue.go` (export `extractAttachments` → `ExtractAttachments`)

**Step 1: Export extractAttachments**

In `plugins/atlassian-cloud/cli/cmd/jira_issue.go`, rename `extractAttachments` to `ExtractAttachments` (it's used in tests too — update `cmd_test.go` references).

Update the call site in `runJiraIssueGet` (same file):
```go
attachments := ExtractAttachments(response.Bytes.Bytes())
```

Update in `cmd_test.go`:
```go
got := ExtractAttachments([]byte(tt.data))
```

**Step 2: Verify rename doesn't break anything**

Run:
```bash
cd plugins/atlassian-cloud/cli && go test ./cmd/ -v
```
Expected: all pass

**Step 3: Create the jira attachment command**

Create `plugins/atlassian-cloud/cli/cmd/jira_attachment.go`:

```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/download"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var jiraAttachmentCmd = &cobra.Command{
	Use:   "attachment",
	Short: "Attachment operations",
}

var jiraAttachmentDownloadCmd = &cobra.Command{
	Use:   "download <issue-key-or-url> [filename]",
	Short: "Download attachments from a Jira issue",
	Long:  "Download a single attachment by filename, or all attachments with --all. Files are saved to a temp directory by default; use --output-dir to specify a target directory.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runJiraAttachmentDownload,
}

var (
	jiraDownloadAll       bool
	jiraDownloadOutputDir string
)

func init() {
	jiraCmd.AddCommand(jiraAttachmentCmd)
	jiraAttachmentCmd.AddCommand(jiraAttachmentDownloadCmd)

	jiraAttachmentDownloadCmd.Flags().BoolVar(&jiraDownloadAll, "all", false, "Download all attachments")
	jiraAttachmentDownloadCmd.Flags().StringVar(&jiraDownloadOutputDir, "output-dir", "", "Directory to save files (default: OS temp dir)")
}

func runJiraAttachmentDownload(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	var filename string
	if len(args) > 1 {
		filename = args[1]
	}

	if filename == "" && !jiraDownloadAll {
		return fmt.Errorf("specify a filename or use --all to download all attachments")
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	// Fetch issue with attachment field
	fields := []string{"attachment"}
	issue, response, err := clients.Jira.Issue.Get(context.Background(), ref.IssueKey, fields, nil)
	if err != nil {
		if response != nil && response.Code == 404 {
			return fmt.Errorf("issue %s not found", ref.IssueKey)
		}
		code := 0
		if response != nil {
			code = response.Code
		}
		return auth.WrapAPIError(code, fmt.Errorf("cannot get issue: %w", err))
	}
	_ = issue // metadata used via raw JSON

	attachments := ExtractAttachments(response.Bytes.Bytes())
	if len(attachments) == 0 {
		return fmt.Errorf("no attachments found on %s", ref.IssueKey)
	}

	// Filter attachments
	type dlTarget struct {
		filename string
		url      string
	}
	var targets []dlTarget

	if jiraDownloadAll {
		for _, a := range attachments {
			targets = append(targets, dlTarget{filename: a.Filename, url: a.Content})
		}
	} else {
		found := false
		for _, a := range attachments {
			if a.Filename == filename {
				targets = append(targets, dlTarget{filename: a.Filename, url: a.Content})
				found = true
				break
			}
		}
		if !found {
			var names []string
			for _, a := range attachments {
				names = append(names, a.Filename)
			}
			return fmt.Errorf("attachment %q not found on %s; available: %v", filename, ref.IssueKey, names)
		}
	}

	// Determine output directory
	outputDir := jiraDownloadOutputDir
	if outputDir == "" {
		outputDir, err = os.MkdirTemp("", "jira-attachments-*")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
	} else {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	// Download each attachment
	ctx := context.Background()
	for _, t := range targets {
		destPath := filepath.Join(outputDir, t.filename)
		if err := download.ToFile(ctx, clients.HTTPClient, t.url, destPath); err != nil {
			if authErr, ok := err.(*download.AuthError); ok {
				return &auth.AuthRequiredError{Message: authErr.Message}
			}
			return fmt.Errorf("download %s: %w", t.filename, err)
		}
		fmt.Println(destPath)
	}

	return nil
}
```

**Step 4: Verify build**

Run:
```bash
cd plugins/atlassian-cloud/cli && go build ./...
```
Expected: successful build

**Step 5: Run all tests**

Run:
```bash
cd plugins/atlassian-cloud/cli && go test ./...
```
Expected: all pass

**Step 6: Commit**

```bash
git add plugins/atlassian-cloud/cli/cmd/jira_attachment.go plugins/atlassian-cloud/cli/cmd/jira_issue.go plugins/atlassian-cloud/cli/cmd/cmd_test.go
git commit -m "feat: add jira attachment download command"
```

---

### Task 4: Add `confluence attachment download` command

**Files:**
- Create: `plugins/atlassian-cloud/cli/cmd/confluence_attachment.go`

**Step 1: Create the confluence attachment command**

Create `plugins/atlassian-cloud/cli/cmd/confluence_attachment.go`:

```go
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/download"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var confluenceAttachmentCmd = &cobra.Command{
	Use:   "attachment",
	Short: "Attachment operations",
}

var confluenceAttachmentDownloadCmd = &cobra.Command{
	Use:   "download <page-id-or-url> [filename]",
	Short: "Download attachments from a Confluence page",
	Long:  "Download a single attachment by filename, or all attachments with --all. Files are saved to a temp directory by default; use --output-dir to specify a target directory.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runConfluenceAttachmentDownload,
}

var (
	confDownloadAll       bool
	confDownloadOutputDir string
)

func init() {
	confluenceCmd.AddCommand(confluenceAttachmentCmd)
	confluenceAttachmentCmd.AddCommand(confluenceAttachmentDownloadCmd)

	confluenceAttachmentDownloadCmd.Flags().BoolVar(&confDownloadAll, "all", false, "Download all attachments")
	confluenceAttachmentDownloadCmd.Flags().StringVar(&confDownloadOutputDir, "output-dir", "", "Directory to save files (default: OS temp dir)")
}

func runConfluenceAttachmentDownload(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseConfluenceRef(args[0])
	if !ok {
		return fmt.Errorf("invalid page reference: %s", args[0])
	}

	var filename string
	if len(args) > 1 {
		filename = args[1]
	}

	if filename == "" && !confDownloadAll {
		return fmt.Errorf("specify a filename or use --all to download all attachments")
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	pageID, err := strconv.Atoi(ref.PageID)
	if err != nil {
		return fmt.Errorf("invalid page ID: %s", ref.PageID)
	}

	attachments, _, err := clients.ConfluenceV2.Attachment.Gets(context.Background(), pageID, "pages", nil, "", 50)
	if err != nil {
		return fmt.Errorf("cannot get attachments: %w", err)
	}

	if attachments == nil || len(attachments.Results) == 0 {
		return fmt.Errorf("no attachments found on page %d", pageID)
	}

	// Filter attachments
	type dlTarget struct {
		filename string
		url      string
	}
	var targets []dlTarget

	if confDownloadAll {
		for _, a := range attachments.Results {
			dlURL := clients.ConfluenceBaseURL + a.DownloadLink
			targets = append(targets, dlTarget{filename: a.Title, url: dlURL})
		}
	} else {
		found := false
		for _, a := range attachments.Results {
			if a.Title == filename {
				dlURL := clients.ConfluenceBaseURL + a.DownloadLink
				targets = append(targets, dlTarget{filename: a.Title, url: dlURL})
				found = true
				break
			}
		}
		if !found {
			var names []string
			for _, a := range attachments.Results {
				names = append(names, a.Title)
			}
			return fmt.Errorf("attachment %q not found on page %d; available: %v", filename, pageID, names)
		}
	}

	// Determine output directory
	outputDir := confDownloadOutputDir
	if outputDir == "" {
		outputDir, err = os.MkdirTemp("", "confluence-attachments-*")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
	} else {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	// Download each attachment
	ctx := context.Background()
	for _, t := range targets {
		destPath := filepath.Join(outputDir, t.filename)
		if err := download.ToFile(ctx, clients.HTTPClient, t.url, destPath); err != nil {
			if authErr, ok := err.(*download.AuthError); ok {
				return &auth.AuthRequiredError{Message: authErr.Message}
			}
			return fmt.Errorf("download %s: %w", t.filename, err)
		}
		fmt.Println(destPath)
	}

	return nil
}
```

**Step 2: Verify build**

Run:
```bash
cd plugins/atlassian-cloud/cli && go build ./...
```
Expected: successful build

**Step 3: Run all tests**

Run:
```bash
cd plugins/atlassian-cloud/cli && go test ./...
```
Expected: all pass

**Step 4: Commit**

```bash
git add plugins/atlassian-cloud/cli/cmd/confluence_attachment.go
git commit -m "feat: add confluence attachment download command"
```

---

### Task 5: Update SKILL.md files with download documentation

**Files:**
- Modify: `plugins/atlassian-cloud/skills/atlassian-jira/SKILL.md`
- Modify: `plugins/atlassian-cloud/skills/atlassian-confluence/SKILL.md`

**Step 1: Add download section to Jira SKILL.md**

In `plugins/atlassian-cloud/skills/atlassian-jira/SKILL.md`, add after the "Custom Fields" section (before "Presenting Results"):

```markdown
## Downloading Attachments

### Single attachment
```bash
$ATLASSIAN_CLI jira attachment download KEY-123 report.pdf
```

Downloads to a temp directory and prints the file path.

### All attachments
```bash
$ATLASSIAN_CLI jira attachment download KEY-123 --all
```

### Save to specific directory
```bash
$ATLASSIAN_CLI jira attachment download KEY-123 report.pdf --output-dir ./downloads
$ATLASSIAN_CLI jira attachment download KEY-123 --all --output-dir ./downloads
```

Output: one file path per line. The agent can use the Read tool to view downloaded files.
```

**Step 2: Add download section to Confluence SKILL.md**

In `plugins/atlassian-cloud/skills/atlassian-confluence/SKILL.md`, add after the "Searching Pages" section (before "Presenting Results"):

```markdown
## Downloading Attachments

### Single attachment
```bash
$ATLASSIAN_CLI confluence attachment download <page-id-or-url> diagram.png
```

Downloads to a temp directory and prints the file path.

### All attachments
```bash
$ATLASSIAN_CLI confluence attachment download <page-id-or-url> --all
```

### Save to specific directory
```bash
$ATLASSIAN_CLI confluence attachment download <page-id-or-url> diagram.png --output-dir ./downloads
$ATLASSIAN_CLI confluence attachment download <page-id-or-url> --all --output-dir ./downloads
```

Output: one file path per line. The agent can use the Read tool to view downloaded files.
```

**Step 3: Commit**

```bash
git add plugins/atlassian-cloud/skills/atlassian-jira/SKILL.md plugins/atlassian-cloud/skills/atlassian-confluence/SKILL.md
git commit -m "docs: add attachment download examples to SKILL.md files"
```

---

### Task 6: Update CLI reference docs and memory

**Files:**
- Modify: `plugins/atlassian-cloud/references/` (if CLI reference doc exists)

**Step 1: Check for existing CLI reference doc**

Look in `plugins/atlassian-cloud/references/` for any CLI reference file. If it exists, add the new commands. If not, skip this.

**Step 2: Update Serena memory**

Update the `atlassian-cloud-cli.md` memory (in both locations — Claude auto-memory and Serena memory) to include the new commands:

Add to CLI Commands section:
```
jira attachment download <key> [filename] [--all --output-dir]
confluence attachment download <id> [filename] [--all --output-dir]
```

Add to Structure section:
```
│   ├── jira_attachment.go    jira attachment download
│   ├── confluence_attachment.go confluence attachment download
```

Add under internal:
```
    ├── download/
    │   ├── download.go       HTTP download to file helper
    │   └── download_test.go
```

**Step 3: Commit any reference doc changes**

```bash
git add -A plugins/atlassian-cloud/references/
git commit -m "docs: update CLI reference with attachment download commands"
```

---

### Task 7: Final build and test verification

**Step 1: Full build**

Run:
```bash
cd plugins/atlassian-cloud/cli && go build -o bin/atlassian-cloud .
```
Expected: successful binary

**Step 2: Full test suite**

Run:
```bash
cd plugins/atlassian-cloud/cli && go test ./... -v
```
Expected: all tests pass

**Step 3: Verify CLI help**

Run:
```bash
cd plugins/atlassian-cloud/cli && ./bin/atlassian-cloud jira attachment download --help
cd plugins/atlassian-cloud/cli && ./bin/atlassian-cloud confluence attachment download --help
```
Expected: both show usage with `--all` and `--output-dir` flags

**Step 4: Go vet**

Run:
```bash
cd plugins/atlassian-cloud/cli && go vet ./...
```
Expected: no issues
