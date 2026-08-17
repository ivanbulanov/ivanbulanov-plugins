# atlassian-cloud

Context-efficient Jira and Confluence Cloud access for Claude Code via a Go CLI built on [go-atlassian](https://github.com/ctreminiom/go-atlassian).

## Features

- **Jira**: Issue lookup (by key or URL), JQL search, comments (read & write), custom field discovery
- **Confluence**: Page reading (by ID or URL), full-text search with CQL, publishing a Markdown document to an existing page (cross-references become deep links, Mermaid diagrams are rendered and embedded, attachment upload)
- **Authentication**: OAuth2 3LO with automatic token renewal, API token fallback
- **Progressive disclosure**: Compact summaries by default, escalate to full details on demand
- **ADF conversion**: Atlassian Document Format converted to clean markdown
- **URL recognition**: Automatically parses `*.atlassian.net` URLs

## Requirements

- Go 1.22+ (or [mise](https://mise.jdx.dev) to install it automatically)
- Atlassian Cloud account
- `mmdc` from [`@mermaid-js/mermaid-cli`](https://github.com/mermaid-js/mermaid-cli)
  — only for `confluence publish`, and only if the document contains Mermaid
  diagrams. Install it yourself with `npm install -g @mermaid-js/mermaid-cli`;
  the plugin never installs or downloads it.

## Setup

The CLI builds automatically on first skill invocation, and the skills guide you through authentication interactively — just start using a Jira or Confluence skill and follow the prompts.

To build or authenticate manually, see below.

<details>
<summary>Manual build</summary>

```bash
cd plugins/atlassian-cloud/cli
go build -o bin/atlassian-cloud .
```
</details>

<details>
<summary>Manual authentication</summary>

**API Token (recommended):**

1. Generate token at https://id.atlassian.com/manage-profile/security/api-tokens
2. Run: `atlassian-cloud auth token --email you@company.com --token YOUR_TOKEN --site company.atlassian.net`

**OAuth2 (for apps or shared environments):**

1. Create an OAuth app at https://developer.atlassian.com/console/myapps/
2. Set callback URL to `http://localhost:19872/callback`
3. Add required scopes: `read:jira-work`, `write:jira-work`, `read:jira-user`, `read:confluence-content.all`, `read:confluence-space.summary`, `write:confluence-content`, `write:confluence-file`, `offline_access`
4. Set environment variables:
   ```bash
   export ATLASSIAN_CLIENT_ID=your-client-id
   export ATLASSIAN_CLIENT_SECRET=your-client-secret
   ```
5. Run: `atlassian-cloud auth login`
</details>

## Sandboxing

Claude Code's Bash sandbox is **off by default**, so no configuration is needed in the common case. With it enabled, the plugin needs two separate things, and the build needs both of them:

- **`scripts/setup.sh` runs `go build`**, which writes the binary into the plugin's own installation directory under `~/.claude/plugins/`, writes the Go build and module caches under `~/.cache` and `~/go`, and fetches modules from the module proxy. Sandboxed commands may write only to the working directory and the session temp directory, so all three are blocked.
- **The CLI itself** makes outbound HTTPS calls to your Atlassian site and the media CDN.

**A plugin cannot exempt itself.** Sandbox settings are user-side only, in `settings.json`; there is no field in `plugin.json` or in skill frontmatter that relaxes them, and `allowed-tools` governs permission prompts rather than the sandbox. So the configuration below has to be yours.

The build is easiest to handle by running just that one script outside the sandbox. Widening `filesystem.allowWrite` instead means naming three separate locations, and much of `~/.claude` is protected from writes in a way `allowWrite` does not lift:

```json
{
  "sandbox": {
    "excludedCommands": [
      "~/.claude/plugins/cache/ivanbulanov-plugins/atlassian-cloud/*/scripts/setup.sh*"
    ],
    "network": {
      "allowedDomains": [
        "*.atlassian.net", "api.atlassian.com", "api.media.atlassian.com"
      ]
    }
  }
}
```

Only the build escapes the sandbox; the CLI keeps running inside it, which is why its hosts still have to be allowed. If the script fails, run it once by hand — it prints the exact `excludedCommands` entry with your real installation path already filled in, which avoids guessing at the version-number wildcard above.

## Usage

Just talk to Claude naturally — the skills trigger automatically.

### Jira

- Paste a Jira URL: `https://acme.atlassian.net/browse/ACME-123`
- Mention an issue key: "What's the status of FE-42?" (any project prefix works)
- Search: "Find open bugs in the INFRA project"
- Comment: "Add a comment to K8S-7 saying the fix is deployed"

### Confluence

- Paste a Confluence URL: `https://acme.atlassian.net/wiki/spaces/ENG/pages/123456/Page`
- Search: "Search Confluence for the deployment runbook"
- Read: "Get the full content of Confluence page 123456"
- Publish: "Publish design.md to Confluence page 123456" (dry-run first; the
  target page must already exist)

### Progressive disclosure

The skills fetch minimal data first (summary only) and escalate to full details only when needed, keeping context usage efficient.

## Security

- Credentials stored in `~/.config/atlassian-cloud/auth.json` with `0600` permissions
- Config directory has `0700` permissions
- OAuth2 tokens automatically refresh before expiry
- API tokens stored encrypted-at-rest (filesystem permissions)
