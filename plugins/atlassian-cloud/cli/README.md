# atlassian-cloud CLI

Context-efficient Jira and Confluence Cloud CLI built in Go on
[go-atlassian](https://github.com/ctreminiom/go-atlassian).

Accepts issue keys, page IDs, or full `*.atlassian.net` URLs. Outputs
markdown with progressive disclosure (summary by default, details via flags).

## Prerequisites

- Go 1.22+ (matches the `go` directive in `go.mod`)
- `mmdc` from `@mermaid-js/mermaid-cli` — only for `confluence publish`, and
  only for documents containing Mermaid fences

## Build & Test

```bash
go build -o bin/atlassian-cloud .
go test ./...
```

The binary is placed at `bin/atlassian-cloud` (gitignored).

## Authentication

Credentials are stored in `$XDG_CONFIG_HOME/atlassian-cloud/auth.json`
(default `~/.config/atlassian-cloud/auth.json`) with `0600` permissions.

### API Token

1. Generate a token at <https://id.atlassian.com/manage-profile/security/api-tokens>
2. Run:

```bash
atlassian-cloud auth token --email you@company.com --token TOKEN --site company.atlassian.net
```

The `--token` flag can be omitted if `ATLASSIAN_API_TOKEN` is set.

### OAuth2

1. Create an OAuth app at <https://developer.atlassian.com/console/myapps/>
2. Set callback URL to `http://localhost:19872/callback`
3. Add scopes: `read:jira-work`, `write:jira-work`, `read:jira-user`,
   `read:confluence-content.all`, `read:confluence-space.summary`, `offline_access`
4. Export credentials:

```bash
export ATLASSIAN_CLIENT_ID=...
export ATLASSIAN_CLIENT_SECRET=...
```

5. Run:

```bash
atlassian-cloud auth login
```

OAuth2 tokens refresh automatically when `ATLASSIAN_CLIENT_ID` and
`ATLASSIAN_CLIENT_SECRET` are available. Refresh happens silently when a
token is within 5 minutes of expiry.

**`confluence publish` writes to a page**, which needs the
`write:confluence-content` scope, and uploads its diagrams, which needs
`write:confluence-file`. Both are in the requested scope list. Anyone who
authorised before they were added holds a token without them, and the API
answers 403 until they run `auth login` again. API-token users need no
change: the token carries the user's own permissions.

### Checking Status

```bash
atlassian-cloud auth status
```

Shows default site, per-site auth method, token validity, and cloud ID.

### Site Resolution

Commands resolve the target Atlassian site in this order:

1. `--site` flag (global persistent flag)
2. Site extracted from a URL argument
3. `DefaultSite` from `auth.json`

## Command Reference

All commands output markdown-flavored plain text to stdout. Errors go to
stderr. There is no JSON mode.

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Authentication required — run `auth login` or `auth token` |
| `3` | `confluence publish` only: refused before writing anything |
| `4` | `confluence publish` only: published, but verification failed |

### Command Tree

```
atlassian-cloud [--site <site>]
├── auth
│   ├── login
│   ├── token  --email --site [--token]
│   └── status
├── jira
│   ├── issue get <key|url>  [--description] [--comments] [--attachments] [--all-fields] [--fields]
│   ├── search <jql>  [--max] [--description]
│   ├── comment
│   │   ├── list <key|url>
│   │   ├── add <key|url>  [--body] [--stdin]
│   │   └── update <key|url> [comment-id]  [--body] [--stdin]
│   ├── attachment download <key|url> [filename]  [--all] [--output-dir]
│   └── fields list
└── confluence
    ├── page get <id|url>  [--body] [--attachments]
    ├── search <query>  [--space] [--max]
    ├── publish <file.md>  [--page] [--title] [--assets-dir] [--link-refs] [--no-toc] [--dry-run] [--force]
    └── attachment
        ├── download <id|url> [filename]  [--all] [--output-dir]
        └── upload <id|url> <file>...
```

### `jira issue get`

Get issue details by key (`PROJ-123`) or URL.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--description` | bool | `false` | Include description |
| `--comments` | bool | `false` | Include comments |
| `--attachments` | bool | `false` | Include attachment list |
| `--all-fields` | bool | `false` | Include description + comments + attachments |
| `--fields` | string | `""` | Comma-separated additional field IDs |

Default fields fetched: summary, status, issuetype, priority, assignee,
reporter, project, labels, created, updated.

**Output:** Bold markdown field list (`**Key**: ..., **Status**: ...`), then
optional `### Description`, `### Comments (N)`, `### Attachments (N)` sections.
ADF bodies are converted to markdown. Comments are numbered with author,
timestamp, and ID. Attachments show filename, size (human-readable), media
type, and download URL.

### `jira search`

Search issues with a JQL query.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max` | int | `20` | Maximum results |
| `--description` | bool | `false` | Fetch descriptions (not rendered in output) |

**Output:** `**Found N issues** (showing M)` followed by a markdown table
with Key, Summary (truncated to 60 chars), Status, and Assignee columns.

**Note:** `--description` adds the description field to the API request but
the search output formatter does not display it.

### `jira comment list`

List up to 50 comments on an issue. No additional flags.

**Output:** Numbered list with author, timestamp, comment ID, and body
(ADF converted to markdown, indented with 3 spaces).

### `jira comment add`

Add a comment to an issue.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--body` | string | `""` | Comment text |
| `--stdin` | bool | `false` | Read body from stdin |

One of `--body` or `--stdin` is required.

**Output:** `Comment added (ID: <id>)`

### `jira comment update`

Update an existing comment. The comment ID can be provided as a second
positional argument or parsed from a `?focusedCommentId=` query parameter
in the URL.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--body` | string | `""` | Comment text |
| `--stdin` | bool | `false` | Read body from stdin |

One of `--body` or `--stdin` is required.

**Output:** `Comment updated (ID: <id>)`

### `jira attachment download`

Download attachments from a Jira issue.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | `false` | Download all attachments |
| `--output-dir` | string | OS temp dir | Target directory |

Provide a filename as the second argument, or use `--all`. When
`--output-dir` is omitted, files are saved to a temp directory named
`jira-attachments-*` under the OS temp dir (typically `/tmp`).

**Output:** One absolute file path per line to stdout.

### `jira fields list`

List all available Jira fields. No additional flags.

**Output:** Markdown table with fixed-width columns: ID (30 char), Name
(25 char), Custom (10 char).

### `confluence page get`

Get page details by numeric ID or URL.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--body` | bool | `false` | Include page body (ADF converted to markdown) |
| `--attachments` | bool | `false` | Include attachment list |

**Output:** Bold field list (Title, Page ID, Space, Version, Created, Status),
then optional `### Content` and `### Attachments (N)` sections.

### `confluence search`

Search Confluence pages. The query is wrapped in CQL internally
(`type = page AND text ~ "<query>"`). Special characters in the query and
space key are escaped automatically.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--space` | string | `""` | Limit to space key |
| `--max` | int | `10` | Maximum results |

**Output:** `**Found N results** (showing M)` followed by a bullet list
with title, excerpt, and URL per result.

### `confluence publish`

Publish a Markdown file to an existing Confluence page: conversion runs
through Confluence's own `contentbody/convert` endpoint, cross-references
become anchored deep links, Mermaid fenced code blocks are rendered locally
with `mmdc` and uploaded as attachments, and a table of contents is spliced
in by default. **This command does not create pages** — the target page must
already exist.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--page` | string | `""` | Target page id or URL. Falls back to a `<!-- confluence-page: id-or-url -->` comment in the document if omitted |
| `--title` | string | `""` | Set the page title |
| `--assets-dir` | string | source document's directory | Where rendered diagrams and the dry-run storage dump are written |
| `--link-refs` | string | `"all"` | Link cross-references: `all` or `none` |
| `--no-toc` | bool | `false` | Do not insert a table of contents. One is inserted by default; a `<!-- confluence-toc -->` comment only chooses where it goes, not whether it appears |
| `--dry-run` | bool | `false` | Generate and check, publish nothing |
| `--force` | bool | `false` | Publish over a page this tool did not last write |

Mermaid rendering needs `mmdc` from `@mermaid-js/mermaid-cli`, installed
separately by the user — the CLI never installs or downloads it. Set `MMDC`
to point at a specific binary. Diagram source is sent only to the target
Confluence site; no hosted renderer is used.

**Exit codes:** `0` success, `1` error, `2` authentication required, `3`
refused before writing anything (unsupported construct, unresolved
cross-reference, the page changed since the last publish, and similar), `4`
published but a post-publish verification check failed.

**Output:** On `--dry-run`, a summary line (references linked, tables
hoisted, links unwrapped) and the path of the written storage dump. On a
real publish, a summary line (references linked, tables hoisted, attachments
uploaded) followed by a verification line (same-page links checked, broken
count). When nothing would change, `no change; the page already says this`
and nothing is written.

### `confluence attachment download`

Download attachments from a Confluence page. Same interface as
`jira attachment download`.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | `false` | Download all attachments |
| `--output-dir` | string | OS temp dir | Target directory |

When `--output-dir` is omitted, files are saved to
`confluence-attachments-*` under the OS temp dir.

**Output:** One absolute file path per line to stdout.

### `confluence attachment upload`

Upload one or more local files as attachments to a Confluence page. No
flags beyond the global `--site`.

**Output:** One attachment filename per line to stdout, as each upload
completes.

## Architecture

```
cli/
├── main.go                       entry point
├── cmd/
│   ├── root.go                   root command, --site flag, site resolution
│   ├── auth.go                   auth login/token/status
│   ├── jira_issue.go             jira issue get, jira search
│   ├── jira_comment.go           jira comment list/add/update
│   ├── jira_fields.go            jira fields list
│   ├── jira_attachment.go        jira attachment download
│   ├── confluence_page.go        confluence page get, confluence search
│   ├── confluence_attachment.go  confluence attachment download/upload
│   ├── confluence_publish.go     confluence publish
│   └── cmd_test.go               tests for resolveSite, escapeCQL, ExtractAttachments
├── internal/
│   ├── auth/
│   │   ├── clients.go            Clients struct, NewClients, HTTP transports
│   │   ├── oauth.go              OAuth2 browser flow, token refresh
│   │   └── auth_test.go
│   ├── config/
│   │   └── config.go             auth.json load/save, XDG_CONFIG_HOME
│   ├── download/
│   │   ├── download.go           HTTP file download helper, AuthError
│   │   └── download_test.go
│   ├── output/
│   │   └── ...                   ADF-to-markdown, issue/page/comment formatting
│   ├── urlparse/
│   │   └── ...                   ParseJiraRef, ParseConfluenceRef
│   ├── confluence/
│   │   └── ...                   contentbody/convert calls, attachment upload/listing
│   └── mdpublish/
│       └── ...                   anchor derivation, source rewriting, splicing, verification
├── go.mod
└── go.sum
```

### Key Types

- **`auth.Clients`** — Created by `auth.NewClients(cfg, site)`. Exposes
  `Jira`, `ConfluenceV1`, `ConfluenceV2` API clients, `JiraBaseURL`,
  `ConfluenceBaseURL`, and a pre-authenticated `HTTPClient` for direct
  downloads. Handles automatic OAuth2 token refresh (5-minute expiry buffer).

- **`config.Config`** — Loaded from `auth.json`. Contains `DefaultSite`
  and a map of per-site credentials (method, token, email, cloud ID).

- **`download.AuthError`** — Returned on HTTP 401 during file downloads,
  converted to `auth.AuthRequiredError` (exit code 2) by commands.

### Patterns

- **Progressive disclosure**: Commands default to summary fields, expand
  with `--description`, `--comments`, `--attachments`, `--all-fields`.
- **URL-or-key input**: All resource commands accept both
  `*.atlassian.net` URLs and plain keys/IDs via `urlparse`.
- **Site resolution**: `--site` flag > URL-extracted site > `DefaultSite`.
- **ADF conversion**: Atlassian Document Format bodies are converted to
  markdown for readable output. Falls back to raw text on parse failure.
- **Cobra structure**: One file per resource in `cmd/`, package-level
  command vars, `init()` for flag/subcommand registration.

## Environment Variables

| Variable | Required | Purpose |
|----------|----------|---------|
| `ATLASSIAN_CLIENT_ID` | For OAuth2 | OAuth2 app client ID |
| `ATLASSIAN_CLIENT_SECRET` | For OAuth2 | OAuth2 app client secret |
| `ATLASSIAN_API_TOKEN` | No | Fallback for `auth token --token` |
| `MMDC` | No | Path to a pinned `mmdc` binary, overriding the `PATH` lookup |
| `XDG_CONFIG_HOME` | No | Config base dir (default `~/.config`) |

Both `ATLASSIAN_CLIENT_ID` and `ATLASSIAN_CLIENT_SECRET` must be set for
`auth login` and for automatic OAuth2 token refresh at runtime. If either
is missing, OAuth2 operations will fail.
