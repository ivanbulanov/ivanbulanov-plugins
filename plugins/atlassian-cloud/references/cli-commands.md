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
| `jira comment update <key\|url> [comment-id]` | Update comment |
| `jira fields list` | List available fields |
| `jira attachment download <key\|url> [filename]` | Download attachments |

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

### Comment Update Flags

| Flag | Description |
|------|-------------|
| `--body` | New comment text |
| `--stdin` | Read from stdin |

### Attachment Download Flags

| Flag | Description |
|------|-------------|
| `--all` | Download all attachments |
| `--output-dir` | Directory to save files (default: OS temp dir) |

## Confluence Commands

| Command | Description |
|---------|-------------|
| `confluence page get <id\|url>` | Get page details |
| `confluence search <query>` | Search with CQL |
| `confluence publish <file.md>` | Publish a Markdown document to an existing page |
| `confluence attachment download <id\|url> [filename]` | Download attachments |
| `confluence attachment upload <id\|url> <file>...` | Upload files as attachments |

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

### Publish Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--page` | (none) | Target page id or URL |
| `--title` | (none) | Set the page title |
| `--assets-dir` | source document's directory | Where rendered diagrams go |
| `--link-refs` | `all` | Link cross-references: `all` or `none` |
| `--no-toc` | false | Do not insert a table of contents (one is inserted by default) |
| `--dry-run` | false | Generate and check, publish nothing |
| `--force` | false | Publish over a page this tool did not last write |

The target page must already exist; this command does not create pages.

Documents containing Mermaid fences also need `mmdc` from
`@mermaid-js/mermaid-cli` on `PATH`, or `MMDC` set to its location. The CLI
never installs it, and diagram source is rendered locally — it is never sent
to a hosted renderer.

Rendered diagrams land in `--assets-dir`, which defaults to the source
document's own directory. Filenames carry a hash of the diagram source, so
republishing an unchanged document re-uses them; editing a diagram writes a new
file and uploads a new attachment, and neither the old file nor the old
attachment is removed. Point `--assets-dir` somewhere disposable if the
document lives in a repository where that clutter matters.

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `ATLASSIAN_CLIENT_ID` | OAuth2 app client ID (required for `auth login`) |
| `ATLASSIAN_CLIENT_SECRET` | OAuth2 app client secret (required for `auth login`) |
| `ATLASSIAN_API_TOKEN` | Fallback for `auth token --token` |
| `MMDC` | Path to a pinned `mmdc` binary, overriding the `PATH` lookup |
| `XDG_CONFIG_HOME` | Config base directory (default `~/.config`) |

### Confluence Attachment Download Flags

| Flag | Description |
|------|-------------|
| `--all` | Download all attachments |
| `--output-dir` | Directory to save files (default: OS temp dir) |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Authentication required |
| 3 | `confluence publish` refused before writing anything |
| 4 | `confluence publish` published, but verification failed |
