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
| `confluence attachment download <id\|url> [filename]` | Download attachments |

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
