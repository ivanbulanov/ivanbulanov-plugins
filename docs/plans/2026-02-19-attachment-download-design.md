# Attachment Download Feature — atlassian-cloud Plugin

## Goal

Add CLI commands to download Jira and Confluence attachments to local files, supporting both single-file and bulk download modes.

## CLI Interface

```
# Jira — single attachment
jira attachment download KEY-123 report.pdf
jira attachment download KEY-123 report.pdf --output-dir ./downloads

# Jira — all attachments
jira attachment download KEY-123 --all
jira attachment download KEY-123 --all --output-dir ./downloads

# Confluence — single attachment
confluence attachment download 12345 diagram.png
confluence attachment download 12345 diagram.png --output-dir ./downloads

# Confluence — all attachments
confluence attachment download 12345 --all
confluence attachment download 12345 --all --output-dir ./downloads
```

**Flags:**
- `--all` — download all attachments from the issue/page
- `--output-dir` — save files to this directory (created if needed). Without this flag, files go to OS temp dir.

**Output:** One file path per line, printed to stdout.

## Architecture

### New files

| File | Purpose |
|------|---------|
| `cmd/jira_attachment.go` | `jira attachment download` command |
| `cmd/confluence_attachment.go` | `confluence attachment download` command |
| `internal/download/download.go` | Shared download helper |

### Download helper

```go
// internal/download/download.go
package download

func ToFile(ctx context.Context, httpClient *http.Client, url, destPath string) error
```

Downloads from a URL using an authenticated HTTP client and writes to `destPath`. Returns error on non-2xx responses.

### Clients struct change

Add `HTTPClient *http.Client` to `auth.Clients` so the download helper can reuse the same authenticated transport (Bearer token or Basic auth headers).

### Jira flow

1. Parse issue key via `urlparse.ParseJiraRef`
2. Fetch issue with `attachment` field via `clients.Jira.Issue.Get()`
3. Extract attachments via existing `extractAttachments()` (uses raw JSON/gjson)
4. Match by filename arg, or take all with `--all`
5. For each attachment: download `Content` URL → write to temp or output dir
6. Print file paths to stdout

### Confluence flow

1. Parse page ref via `urlparse.ParseConfluenceRef`
2. Fetch attachments via `clients.ConfluenceV2.Attachment.Gets()`
3. Match by filename arg, or take all with `--all`
4. Construct full download URL from `DownloadLink` (relative path needs site base URL prepended)
5. Download → write to temp or output dir
6. Print file paths to stdout

## Error handling

| Scenario | Behavior |
|----------|----------|
| Attachment not found by name | Error listing available attachment names |
| Auth failure (401) | Wrapped as `AuthRequiredError` → exit code 2 |
| Download HTTP error | Error with filename and HTTP status code |
| No attachments on issue/page | Clear message: "no attachments found" |

## SKILL.md updates

- Add download examples to `atlassian-jira/SKILL.md` as a new progressive disclosure level
- Add download examples to `atlassian-confluence/SKILL.md`
