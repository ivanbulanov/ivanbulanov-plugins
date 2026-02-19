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
