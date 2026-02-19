# Atlassian Cloud Plugin Design

**Date:** 2026-02-19
**Status:** Approved

## Overview

A new Claude Code plugin (`atlassian-cloud`) that provides context-efficient access to Jira and Confluence Cloud via a Go CLI built on the [go-atlassian](https://github.com/ctreminiom/go-atlassian) library. Exists alongside the existing `acli-jira` plugin as a separate, broader-scoped integration.

## Decisions

| Decision | Choice |
|----------|--------|
| Plugin name | `atlassian-cloud` |
| Relationship to acli-jira | Separate plugin |
| Architecture | Monolithic CLI binary + skill scripts |
| Auth model | OAuth2 primary, API token fallback |
| Output format | Markdown (ADF converted in Go) |
| Distribution | Build on first use (Go required or installed via mise) |
| Confluence scope | Read-only (v1) |
| Jira write scope | Comments only |

## Plugin Structure

```
plugins/atlassian-cloud/
├── .claude-plugin/
│   └── plugin.json
├── README.md
├── cli/                              # Go source code
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── cmd/                          # Cobra command tree
│   │   ├── root.go
│   │   ├── auth.go
│   │   ├── jira_issue.go
│   │   ├── jira_comment.go
│   │   ├── jira_fields.go
│   │   └── confluence_page.go
│   ├── internal/
│   │   ├── auth/                     # OAuth2 + API token management
│   │   ├── output/                   # Markdown formatter, ADF→MD converter
│   │   └── config/                   # Config file handling
│   └── Makefile
├── skills/
│   ├── atlassian-jira/
│   │   ├── SKILL.md
│   │   └── scripts/
│   │       └── ensure-binary.sh
│   └── atlassian-confluence/
│       └── SKILL.md
└── references/
    ├── adf-format.md
    └── cli-commands.md
```

## Skills

### atlassian-jira

**Triggers on:**
- URLs: `https://*.atlassian.net/browse/PROJ-123`
- Ticket IDs: `DEV-123`, `PROJ-456` (uppercase + dash + digits)
- Phrases: "get jira issue", "search jira", "find ticket", "comment on ticket"

**Capabilities:**
- Retrieve issue details (progressive disclosure)
- JQL search
- List and add comments
- List attachments
- Discover custom fields

### atlassian-confluence

**Triggers on:**
- URLs: `https://*.atlassian.net/wiki/spaces/*/pages/*`
- Phrases: "read confluence page", "search confluence", "find in confluence"

**Capabilities:**
- Retrieve page content (progressive disclosure)
- Search pages within spaces
- List attachments

## CLI Commands

### Authentication

```bash
# OAuth2 browser flow
atlassian-cloud auth login

# API token fallback
atlassian-cloud auth token --email user@co.com --token XXXX --site co.atlassian.net

# Check status
atlassian-cloud auth status
```

### Jira

```bash
# Issue retrieval (progressive disclosure)
atlassian-cloud jira issue get DEV-123                          # Level 1: summary
atlassian-cloud jira issue get DEV-123 --description            # Level 2: + description
atlassian-cloud jira issue get DEV-123 --description --comments # Level 3: + comments
atlassian-cloud jira issue get DEV-123 --all-fields             # Everything

# URL input works too
atlassian-cloud jira issue get https://co.atlassian.net/browse/DEV-123

# Search
atlassian-cloud jira search "project = DEV AND status = Open" --max 20
atlassian-cloud jira search "text ~ 'deployment'" --max 10 --description

# Comments
atlassian-cloud jira comment list DEV-123
atlassian-cloud jira comment add DEV-123 --body "Fix deployed"
echo "Long comment" | atlassian-cloud jira comment add DEV-123 --stdin

# Custom fields
atlassian-cloud jira fields list --project DEV
atlassian-cloud jira issue get DEV-123 --fields "Story Points,Sprint"
```

### Confluence

```bash
# Page retrieval (progressive disclosure)
atlassian-cloud confluence page get <url-or-id>                 # Summary
atlassian-cloud confluence page get <url-or-id> --body          # + full body
atlassian-cloud confluence page get <url-or-id> --body --attachments

# Search
atlassian-cloud confluence search "deployment runbook" --space WS --max 10
```

## Progressive Disclosure

| Level | Flags | Typical tokens |
|-------|-------|---------------|
| 1 (default) | none | 200-500 |
| 2 | `--description` | 700-2500 |
| 3 | `--comments` | 1500-5000 |
| 4 | `--attachments` | +200-500 |
| Full | `--all-fields` | 3000-10000 |

Skills instruct Claude to always start at Level 1 and only escalate when more detail is needed.

## Authentication

### Token Storage

Location: `~/.config/atlassian-cloud/auth.json` (permissions: `0600`)
Directory: `~/.config/atlassian-cloud/` (permissions: `0700`)

```json
{
  "default_site": "company.atlassian.net",
  "sites": {
    "company.atlassian.net": {
      "method": "oauth2",
      "access_token": "...",
      "refresh_token": "...",
      "token_expiry": "2026-02-19T15:30:00Z",
      "cloud_id": "abc-123-def",
      "scopes": ["read:jira-work", "write:jira-work", "read:confluence-content.all"]
    }
  }
}
```

### Token Renewal Flow

Every CLI command:
1. Load auth config
2. Check `token_expiry`
3. If expiry < now + 5 min:
   - Use `refresh_token` to get new `access_token`
   - Update `auth.json` with new token + expiry
   - If refresh fails (revoked/expired): exit with code 2
4. Proceed with API call

Exit code 2 signals "re-authentication needed" — skills instruct Claude to guide the user through `auth login`.

### OAuth2 Login Flow

1. Read `client_id` / `client_secret` from env or config
2. Start temporary HTTP server on `localhost:19872`
3. Generate authorization URL with PKCE
4. Open browser to Atlassian consent page
5. User grants access → redirect to `localhost:19872/callback`
6. Exchange auth code for tokens
7. Fetch accessible resources (cloud_id)
8. Store tokens in `auth.json`
9. Print success + site name

### Permission Enforcement

- CLI verifies file permissions on startup
- Warns and offers to fix if `auth.json` is world-readable
- Config directory created with `0700`
- All credential files written with `0600`

## Custom Field Discovery

```bash
atlassian-cloud jira fields list --project DEV
```

Field resolution:
1. If `--fields` value matches `customfield_*` → use directly
2. Otherwise, look up friendly name in local field cache (`~/.config/atlassian-cloud/fields-{site}.json`, TTL: 24h)
3. If cache miss, fetch from API and cache
4. If no match → error with suggestions

## ADF → Markdown Conversion

Done in Go, in-process. Key mappings:

| ADF Node | Markdown |
|----------|----------|
| `heading` (1-6) | `#` through `######` |
| `paragraph` | Plain text + newline |
| `bulletList` / `orderedList` | `- ` / `1. ` |
| `codeBlock` | Fenced code block |
| `blockquote` | `> ` |
| `table` | `\| cell \|` table |
| `mediaSingle` | `![alt](url)` |
| `mention` | `@display-name` |
| `emoji` | Unicode emoji |
| `inlineCard` | `[title](url)` |
| `status` | `[STATUS_TEXT]` |
| Marks: `strong`, `em`, `code`, `link` | `**`, `*`, `` ` ``, `[text](url)` |

## URL Parsing

The CLI parses Atlassian URLs to extract context:

| URL Pattern | Extracted |
|------------|-----------|
| `https://{site}.atlassian.net/browse/{KEY}` | site, issue key |
| `https://{site}.atlassian.net/wiki/spaces/{SPACE}/pages/{ID}/{Title}` | site, space, page ID |

Both URL and direct ID input are supported for all commands.

## Build & Distribution

- Go source in `plugins/atlassian-cloud/cli/`
- `ensure-binary.sh` script checks for compiled binary
- If missing: checks for `go` in PATH, falls back to `mise install go`, then builds
- Binary placed in `plugins/atlassian-cloud/cli/bin/atlassian-cloud`
- `.gitignore` excludes the binary
- Uses `cobra` for CLI framework, `go-atlassian` for API access

## Technology Stack

- **Language:** Go (1.21+)
- **CLI framework:** [cobra](https://github.com/spf13/cobra)
- **Atlassian API:** [go-atlassian](https://github.com/ctreminiom/go-atlassian)
- **Config:** JSON files in `~/.config/atlassian-cloud/`
- **Build:** Standard `go build`, automated via `ensure-binary.sh`
