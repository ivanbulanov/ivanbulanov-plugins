# atlassian-cloud Code Review TODO

Findings from a comprehensive 6-agent parallel code review of the full plugin codebase.
Organized by priority. Each item includes the exact file, line numbers, and a fix description.

---

## Must Fix (Security / Correctness)

### ~~1. CQL injection in Confluence search~~ DONE

**File:** `cli/cmd/confluence_page.go:110-114`
**Category:** Security — injection
**Confidence:** 100

User input from `args[0]` and `--space` flag is interpolated directly into a CQL query
string via `fmt.Sprintf` without escaping double quotes:

```go
cql := `type = page`
if searchSpace != "" {
    cql += fmt.Sprintf(` AND space = "%s"`, searchSpace)
}
cql += fmt.Sprintf(` AND text ~ "%s"`, args[0])
```

A search like `atlassian-cloud confluence search 'foo" OR type = "attachment'` produces
`type = page AND text ~ "foo" OR type = "attachment"`, breaking out of the intended query
and bypassing filters.

**Fix:** Escape double quotes in user input before interpolation. Add a helper:

```go
func escapeCQL(s string) string {
    s = strings.ReplaceAll(s, `\`, `\\`)
    s = strings.ReplaceAll(s, `"`, `\"`)
    return s
}
```

Apply it to both `searchSpace` and `args[0]` before `fmt.Sprintf`.

---

### ~~2. Reflected XSS in OAuth callback error page~~ DONE

**File:** `cli/internal/auth/oauth2.go:63-67`
**Category:** Security — XSS
**Confidence:** 100

The `error` query parameter from the HTTP request is written into an HTML response
without escaping:

```go
errMsg := r.URL.Query().Get("error")
if errMsg == "" {
    errMsg = "no authorization code received"
}
fmt.Fprintf(w, "<html><body><h2>Authorization failed</h2><p>%s</p></body></html>", errMsg)
```

An attacker can craft a URL like
`http://localhost:19872/callback?error=<script>alert(1)</script>` to inject JavaScript.

**Fix:** Use `html.EscapeString(errMsg)` before writing it into the HTML response.
Add `"html"` to the import block.

---

### ~~3. OAuth2 state parameter is static (CSRF)~~ DONE

**File:** `cli/internal/auth/oauth2.go:92`
**Category:** Security — CSRF
**Confidence:** 75

The OAuth2 `state` parameter is hardcoded to `"atlassian-cloud-cli"`:

```go
authURL, err := svc.GetAuthorizationURL(cfg.Scopes, "atlassian-cloud-cli")
```

RFC 6749 Section 10.12 requires the state to be an unpredictable, per-request value.
An attacker who knows the static state can forge the callback with their own authorization
code, binding the victim's CLI to the attacker's Atlassian account.

**Fix:**
1. Generate a cryptographically random state (32 bytes from `crypto/rand`, base64-encoded).
2. Pass it to `GetAuthorizationURL`.
3. In the callback handler, validate `r.URL.Query().Get("state")` matches before accepting
   the code. Reject with an error if it doesn't match.

---

### ~~4. Token refresh silently skipped on malformed expiry~~ DONE

**File:** `cli/internal/auth/clients.go:119-122`
**Category:** Bug — silent failure
**Confidence:** 75

The condition combines a `time.Parse` error with a double-negative time comparison:

```go
expiry, err := time.Parse(time.RFC3339, siteAuth.TokenExpiry)
if err != nil || !time.Now().Add(tokenExpiryBuffer).After(expiry) {
    return siteAuth.AccessToken, nil
}
```

If `time.Parse` fails (corrupted config, manual edit, format change), the function returns
the existing access token without refreshing. If that token is also expired, all API calls
silently fail with 401.

**Fix:** Split the conditions — treat parse error as "expiry unknown, refresh to be safe":

```go
expiry, err := time.Parse(time.RFC3339, siteAuth.TokenExpiry)
if err != nil {
    fmt.Fprintf(os.Stderr, "Warning: cannot parse token expiry %q, attempting refresh\n", siteAuth.TokenExpiry)
    // fall through to refresh logic below
} else if !time.Now().Add(tokenExpiryBuffer).After(expiry) {
    return siteAuth.AccessToken, nil
}
```

---

## Should Fix (Functionality / Architecture)

### ~~5. `--fields` flag declared but never consumed~~ DONE

**File:** `cli/cmd/jira_issue.go:44,59`
**Category:** Dead feature — misleads users
**Confidence:** 100

The `issueFields` variable is declared and bound to the `--fields` flag but never read
in `runJiraIssueGet`. The field list at lines 76-85 is built entirely from boolean flags.
Users who pass `--fields customfield_10001` get no error and no custom fields.

**Fix:** After line 85, add:

```go
if issueFields != "" {
    for _, f := range strings.Split(issueFields, ",") {
        fields = append(fields, strings.TrimSpace(f))
    }
}
```

Add `"strings"` to the import block if not already present.

---

### ~~6. `--attachments` flag shows placeholder instead of calling `FormatAttachments`~~ PARTIAL (V2 model lacks Attachment field; placeholder removed)

**File:** `cli/cmd/jira_issue.go:109-111` and `cli/internal/output/jira.go:90-107`
**Category:** Dead code / incomplete feature
**Confidence:** 100

The `--attachments` flag fetches the `attachment` field from the API (line 83-85) but
discards the data and prints a confusing placeholder:

```go
if issueAttachments || issueAllFields {
    fmt.Println("\n*Use --attachments with a follow-up to list attachment details*")
}
```

Meanwhile a fully implemented `FormatAttachments()` exists at `jira.go:90`.

**Fix:** Replace lines 109-111 with:

```go
if (issueAttachments || issueAllFields) && issue.Fields.Attachment != nil {
    fmt.Print(output.FormatAttachments(issue.Fields.Attachment))
}
```

---

### ~~7. `os.Exit()` in library code — `NewClients`~~ DONE

**File:** `cli/internal/auth/clients.go:44-53`
**Category:** Architecture — untestable, violates Go convention
**Confidence:** 75

`NewClients` has signature `(*Clients, error)` but calls `os.Exit(2)` on two paths:

```go
if site == "" {
    fmt.Fprintln(os.Stderr, "Not authenticated. Run: atlassian-cloud auth login")
    os.Exit(ExitCodeAuthRequired)
}
siteAuth, ok := cfg.Sites[site]
if !ok {
    fmt.Fprintf(os.Stderr, "No credentials for site %q. Run: atlassian-cloud auth login\n", site)
    os.Exit(ExitCodeAuthRequired)
}
```

This makes the function untestable and lies about its contract (it says it returns an
error but actually terminates the process).

**Fix:** Return errors instead. Define a sentinel error type if you need the special exit
code:

```go
type AuthRequiredError struct{ Message string }
func (e *AuthRequiredError) Error() string { return e.Message }
```

Then in `cmd/root.go` `Execute()`, check for `*AuthRequiredError` and use exit code 2.

---

### ~~8. `os.Exit()` in `RunE` handler — `runJiraIssueGet`~~ DONE

**File:** `cli/cmd/jira_issue.go:92-95`
**Category:** Architecture — bypasses cobra error handling
**Confidence:** 75

```go
if response != nil && response.Code == 401 {
    fmt.Fprintln(os.Stderr, "Authentication failed. Run: atlassian-cloud auth login")
    os.Exit(auth.ExitCodeAuthRequired)
}
```

A `RunE` handler should return errors, not call `os.Exit`. Cobra handles the error return.

**Fix:** Return a (possibly sentinel) error:

```go
if response != nil && response.Code == 401 {
    return &auth.AuthRequiredError{Message: "authentication failed; run: atlassian-cloud auth login"}
}
```

---

### ~~9. Inconsistent HTTP 401 handling across commands~~ DONE

**File:** Only `cli/cmd/jira_issue.go:92-96` checks for 401.
**Category:** Contract inconsistency
**Confidence:** 50

The other 6 command handlers (`runJiraSearch`, `runJiraCommentList`, `runJiraCommentAdd`,
`runJiraFieldsList`, `runConfluencePageGet`, `runConfluenceSearch`) do not check for 401.
The documented exit code 2 is only produced by one command.

**Fix:** After fixing items 7 and 8 (returning errors from `NewClients`), handle 401 in a
shared helper or HTTP transport middleware so all commands benefit uniformly. Or at minimum,
add 401 checks to each command's error handling block.

---

### ~~10. API token exposed in process listing~~ DONE

**File:** `cli/cmd/auth.go:47-51`
**Category:** Security — credential exposure
**Confidence:** 75

The `--token` flag requires the API token as a CLI argument, making it visible via `ps`,
`/proc/<pid>/cmdline`, and shell history:

```go
authTokenCmd.Flags().StringVar(&tokenToken, "token", "", "Atlassian API token")
_ = authTokenCmd.MarkFlagRequired("token")
```

**Fix:** Support reading the token from the `ATLASSIAN_API_TOKEN` environment variable as
the primary method. Make `--token` optional and fall back to the env var:

```go
token := tokenToken
if token == "" {
    token = os.Getenv("ATLASSIAN_API_TOKEN")
}
if token == "" {
    return fmt.Errorf("API token required: use --token flag or ATLASSIAN_API_TOKEN env var")
}
```

Remove `MarkFlagRequired("token")` so the flag becomes optional.

---

### ~~11. `truncate` corrupts multi-byte UTF-8 characters~~ DONE

**File:** `cli/internal/output/jira.go:153-158`
**Category:** Bug — data corruption
**Confidence:** 75

`len(s)` and `s[:max-3]` operate on bytes, not runes. For issue summaries containing CJK,
emoji, or accented characters, truncation can split a multi-byte sequence:

```go
func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max-3] + "..."
}
```

**Fix:** Use `[]rune` conversion:

```go
func truncate(s string, max int) string {
    runes := []rune(s)
    if len(runes) <= max {
        return s
    }
    if max < 4 {
        return s
    }
    return string(runes[:max-3]) + "..."
}
```

---

### ~~12. Table cell block content silently discarded in ADF converter~~ DONE

**File:** `cli/internal/output/adf.go:219-222`
**Category:** Bug — data loss
**Confidence:** 50

Table cells call `convertInlineContent` which only handles inline node types. Block-level
content (headings, code blocks, lists, nested paragraphs) inside table cells is silently
lost:

```go
for _, child := range cell.Content {
    convertInlineContent(&cellBuf, child.Content)
}
```

Real Confluence tables often contain code blocks, lists, or headings in cells.

**Fix:** Use `convertNode` for each child and flatten the result into a single line for
table cell compatibility:

```go
for _, child := range cell.Content {
    convertNode(&cellBuf, child, "")
}
```

Then strip newlines from `cellBuf.String()` when building the table row (replace `\n`
with space, then trim).

---

### ~~13. Nested list rendering produces malformed markdown~~ DONE

**File:** `cli/internal/output/adf.go:93-100`
**Category:** Bug — incorrect output
**Confidence:** 50

When a list item contains a nested sub-list, each nested item receives the outer prefix
prepended. A nested bullet list renders as `- - inner item` instead of properly indented
`  - inner item`:

```go
case "bulletList":
    for _, item := range node.Content {
        if item.Type == "listItem" {
            for _, child := range item.Content {
                convertNode(sb, child, "- ")
            }
        }
    }
```

**Fix:** Track nesting depth or pass an indentation prefix. For nested lists, prepend
indentation spaces instead of repeating the marker. Distinguish between the first child
(which gets the marker) and subsequent children (which get indentation only).

---

### ~~14. `openBrowser` silently fails on Windows~~ DONE

**File:** `cli/internal/auth/oauth2.go:141-147`
**Category:** Bug — platform support
**Confidence:** 75

Missing `runtime.GOOS == "windows"` case. The error from `exec.Command` is discarded:

```go
func openBrowser(url string) {
    name := "xdg-open"
    if runtime.GOOS == "darwin" {
        name = "open"
    }
    _ = exec.Command(name, url).Start()
}
```

**Fix:**

```go
func openBrowser(url string) {
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "darwin":
        cmd = exec.Command("open", url)
    case "windows":
        cmd = exec.Command("cmd", "/c", "start", url)
    default:
        cmd = exec.Command("xdg-open", url)
    }
    _ = cmd.Start()
}
```

---

## Nice to Have (Code Quality)

### ~~15. Define auth method constants~~ DONE

**Files:** `cli/cmd/auth.go:84,114,146` · `cli/internal/auth/clients.go:56,58`
**Category:** Code quality — magic strings
**Confidence:** 75

The strings `"oauth2"` and `"token"` appear in 5+ locations across 2 packages. A typo
causes silent authentication failure.

**Fix:** Add constants to the `config` package:

```go
const (
    AuthMethodOAuth2 = "oauth2"
    AuthMethodToken  = "token"
)
```

Replace all bare string literals across `auth.go`, `clients.go`, and `config_test.go`.

---

### ~~16. `SiteAuth` flat struct allows invalid field combinations~~ DONE (Validate() method)

**File:** `cli/internal/config/config.go:17-26`
**Category:** Type design
**Confidence:** 50

`SiteAuth` uses a flat struct with a `Method` discriminator. OAuth2 fields (`AccessToken`,
`RefreshToken`, `TokenExpiry`, `CloudID`, `Scopes`) and token fields (`Email`, `APIToken`)
are all on the same struct. Nothing prevents creating
`SiteAuth{Method: "token", AccessToken: "leaked_oauth_token"}`.

**Fix:** At minimum, add a `Validate()` method called from `LoadAuthConfig`. For stronger
guarantees, consider separating the auth data into method-specific structs or using a
tagged union pattern.

---

### ~~17. `TokenExpiry` stored as raw string instead of `time.Time`~~ DONE (ExpiryTime() helper)

**File:** `cli/internal/config/config.go:21`
**Category:** Type design — primitive obsession
**Confidence:** 50

`TokenExpiry` is a `string`, forcing `time.Parse(time.RFC3339, ...)` calls in 4 locations:
- `cli/cmd/auth.go:87` (writing)
- `cli/cmd/auth.go:147` (reading for status)
- `cli/internal/auth/clients.go:119` (reading for refresh)
- `cli/internal/auth/clients.go:138` (writing after refresh)

**Fix:** Change to `time.Time` with custom JSON marshaling, or keep as string but add a
helper method `(s *SiteAuth) ExpiryTime() (time.Time, error)` to centralize the parsing.

---

### ~~18. `Clients.SiteURL` is misleadingly named~~ DONE (renamed to JiraBaseURL)

**File:** `cli/internal/auth/clients.go:30,110`
**Category:** Naming
**Confidence:** 50

`SiteURL` is set to the Jira API URL (`https://api.atlassian.com/ex/jira/{cloudID}` for
OAuth2), not the actual user-facing site URL (`https://{site}.atlassian.net`). The field
is currently unused by any command but is exported.

**Fix:** Rename to `JiraBaseURL`, or set it to the actual site URL, or remove it if unused.

---

### ~~19. No config schema version for future migration~~ DONE

**File:** `cli/internal/config/config.go:12-15`
**Category:** Forward compatibility
**Confidence:** 25

`AuthConfig` has no version field. If the config format changes, there's no way to detect
and migrate old configs.

**Fix:** Add `Version int` to `AuthConfig`:

```go
type AuthConfig struct {
    Version     int                 `json:"version"`
    DefaultSite string              `json:"default_site"`
    Sites       map[string]SiteAuth `json:"sites"`
}
```

---

### ~~20. Standardize `resolveSite` usage across all commands~~ DONE

**File:** `cli/cmd/jira_issue.go:117` · `cli/cmd/jira_fields.go:29` · `cli/cmd/confluence_page.go:105`
**Category:** Consistency
**Confidence:** 25

Three search/list commands pass `siteName` directly to `auth.NewClients` instead of routing
through `resolveSite("")`. Functionally equivalent today but diverges if site resolution
logic changes.

**Fix:** Replace `auth.NewClients(siteName)` with `auth.NewClients(resolveSite(""))` in
the three call sites.

---

## Test Coverage Gaps

### 21. `internal/auth/` — zero tests (remaining)

The entire auth package (OAuth2 flow, token refresh, client factory) has no tests.
Priority scenarios:
- `LoadOAuthConfigFromEnv()` with missing/present env vars
- `refreshTokenIfNeeded()` with expired, fresh, and malformed expiry strings
- `NewClients()` with unknown auth method, missing site, valid config

### ~~22. `internal/output/jira.go` and `confluence.go` — zero tests~~ DONE

All formatting functions that produce user-visible output are completely untested:
- `FormatIssueSummary` with nil `Fields`, nil nested fields
- `FormatComments` with nil `Author`, ADF comment body
- `FormatSearchResults` table rendering
- `FormatPageSummary`, `FormatPageBody`, `FormatConfluenceAttachments`
- `formatSize` boundary values (exactly 1024, 1048576)
- `truncate` with UTF-8, short strings, edge cases

### ~~23. `internal/output/adf.go` — missing node type tests~~ DONE

Existing `TestADFToMarkdown` covers 10 cases but only single-node documents. Untested
implemented features:
- `orderedList`, `table`, `mediaSingle`/`mediaGroup`, `rule`, `panel`, `expand`
- `emoji`, `inlineCard`, `status`, `hardBreak` inline types
- `em`, `strike`, `underline` marks; multiple marks on one node
- Multi-node documents (heading + paragraph + list)
- Invalid JSON input (error path)
- `renderADF()` fallback behavior

### ~~24. `internal/config/` — missing error path tests~~ DONE

- `LoadAuthConfig` when no auth file exists (first-run experience)
- `LoadAuthConfig` with malformed JSON
- `Dir()` when `XDG_CONFIG_HOME` is unset (most common macOS path)

### 25. `cmd/` — zero tests

Business logic in command layer is untested:
- CQL construction in `runConfluenceSearch`
- Field selection logic in `runJiraIssueGet`
- `resolveSite` function
