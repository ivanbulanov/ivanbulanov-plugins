---
name: atlassian-jira
description: Use when the user asks to "fetch JIRA issue", "get ticket", "show DEV-123", "look up issue", "search jira", "find tickets", "comment on ticket", "add comment to issue", "update comment", "edit comment", or pastes a JIRA URL like "https://company.atlassian.net/browse/KEY-123". Also triggers on bare issue keys like "DEV-123" or "PROJ-456" in the user's message.
---

# Jira Issue Operations via atlassian-cloud CLI

Access Jira Cloud issues, comments, and search with progressive disclosure for context efficiency.

## Setup

Run once at the start of any operation:

```bash
<base-directory>/../../../scripts/setup.sh
```

This builds the CLI if needed and checks authentication. If it fails with a Go error, guide the user to install Go (`mise install go@latest` or https://go.dev/dl/).

**CLI path**: `<base-directory>/../../../cli/bin/atlassian-cloud` — shown as `atlassian-cloud` in examples below.

If already authenticated, proceed to the operation. Otherwise, walk the user through setup.

### Guided API Token Setup (recommended)

API tokens are the simplest way to authenticate. Walk the user through these steps interactively:

**Step 1: Extract the site from context.**
If the user pasted a URL like `https://acme-corp.atlassian.net/browse/DEV-123`, the site is `acme-corp.atlassian.net`.
If only a bare issue key was given, ask: "What is your Atlassian site URL? (e.g. `yourcompany.atlassian.net`)"

**Step 2: Ask for their Atlassian account email.**
"What email do you use to log into `<site>`?"

**Step 3: Direct them to create an API token.**
Tell the user:
> Go to https://id.atlassian.com/manage-profile/security/api-tokens and click **Create API token**.
> Give it a name (e.g. "CLI") and copy the token value.

Then ask: "Paste your API token here (it will be stored locally in `~/.config/atlassian-cloud/auth.json` with 0600 permissions)."

**Step 4: Run the auth command with the collected values.**
```bash
atlassian-cloud auth token --email <email> --token <token> --site <site>
```

**Step 5: Verify.**
```bash
atlassian-cloud auth status
```

If successful, proceed with the original request. If it fails, check for typos in email/token/site.

### OAuth2 (alternative — for apps or shared environments)

Only suggest this if the user specifically wants browser-based login or is building an integration.
Requires `ATLASSIAN_CLIENT_ID` and `ATLASSIAN_CLIENT_SECRET` environment variables from an OAuth 2.0 app at https://developer.atlassian.com/console/myapps/.
```bash
atlassian-cloud auth login
```

### Re-authentication

If any command exits with code 2, the token is invalid or expired. Run `atlassian-cloud auth status` to diagnose, then repeat the setup if needed. API tokens don't expire unless revoked, so exit code 2 usually means the token was deleted or the email/site is wrong.

**Sandboxing:** If you have enabled Claude Code's Bash sandbox (it is off by default), the build step and the CLI need network access. See the plugin README's **Sandboxing** section for the `settings.json` allowlist.

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
atlassian-cloud --site acme-corp.atlassian.net jira issue get ACME-5136
```

## Progressive Disclosure — ALWAYS Start at Level 1

### Level 1: Summary (default — use this first)

```bash
atlassian-cloud jira issue get KEY-123
```

Returns: Key, Summary, Status, Type, Priority, Assignee, Reporter, Project, Labels, Created, Updated.

**This is usually sufficient.** Only escalate if the user asks for more detail.

### Level 2: + Description

```bash
atlassian-cloud jira issue get KEY-123 --description
```

Adds the full issue description (ADF converted to markdown).

### Level 3: + Comments

```bash
atlassian-cloud jira issue get KEY-123 --description --comments
```

Adds all comments with author and timestamp.

### Level 4: + Attachments

```bash
atlassian-cloud jira issue get KEY-123 --description --comments --attachments
```

Adds attachment list with filenames, sizes, and download URLs.

### Full context

```bash
atlassian-cloud jira issue get KEY-123 --all-fields
```

## Searching Issues

```bash
# Basic JQL search
atlassian-cloud jira search "project = DEV AND status = Open" --max 20

# Text search
atlassian-cloud jira search "text ~ 'deployment error'" --max 10

# With descriptions
atlassian-cloud jira search "assignee = currentUser() AND status != Done" --max 15 --description
```

Returns a markdown table: Key | Summary | Status | Assignee.

## Comments

### List comments
```bash
atlassian-cloud jira comment list KEY-123
```

### Add a comment
```bash
atlassian-cloud jira comment add KEY-123 --body "Fix deployed in v2.3.1"
```

For longer comments:
```bash
echo "Detailed analysis of the issue..." | atlassian-cloud jira comment add KEY-123 --stdin
```

### Update a comment
```bash
atlassian-cloud jira comment update KEY-123 10042 --body "Updated analysis"
```

Using a comment URL:
```bash
atlassian-cloud jira comment update "https://company.atlassian.net/browse/KEY-123?focusedCommentId=10042" --body "Updated text"
```

For longer updates:
```bash
echo "Revised detailed analysis..." | atlassian-cloud jira comment update KEY-123 10042 --stdin
```

## Custom Fields

### Discover available fields
```bash
atlassian-cloud jira fields list
```

### Request specific fields
```bash
atlassian-cloud jira issue get KEY-123 --fields "Story Points,Sprint,customfield_10001"
```

## Downloading Attachments

### Single attachment
```bash
atlassian-cloud jira attachment download KEY-123 report.pdf
```

Downloads to a temp directory and prints the file path.

### All attachments
```bash
atlassian-cloud jira attachment download KEY-123 --all
```

### Save to specific directory
```bash
atlassian-cloud jira attachment download KEY-123 report.pdf --output-dir ./downloads
atlassian-cloud jira attachment download KEY-123 --all --output-dir ./downloads
```

Output: one file path per line. The agent can use the Read tool to view downloaded files.

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
| Exit code 2 | Auth expired → `atlassian-cloud auth login` |
| "issue not found" | Check key format, verify permissions |
| "cannot connect" | Check network, verify site URL |
| Build failure | Ensure Go is installed: `go version` |

## Tips

1. **Always start at Level 1** — escalate only when more detail is needed
2. **Parse URLs automatically** — the CLI handles both URLs and bare keys
3. **Use --site for cross-site** — when URL contains a different site than default
4. **Search before fetching** — use JQL to find relevant issues first
5. **Comments are paginated** — last 50 shown by default
