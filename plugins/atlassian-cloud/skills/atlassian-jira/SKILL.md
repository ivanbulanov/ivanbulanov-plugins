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
