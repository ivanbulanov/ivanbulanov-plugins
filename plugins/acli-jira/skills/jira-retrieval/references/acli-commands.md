# acli JIRA Commands Reference

Complete reference for acli JIRA commands used in issue retrieval.

## Authentication

```bash
# Authenticate with JIRA Cloud
acli jira auth

# Check current auth status
acli jira auth status
```

## Workitem Commands

### View Issue

```bash
acli jira workitem view <key> [flags]
```

**Flags:**

| Flag | Description | Example |
|------|-------------|---------|
| `--fields` | Comma-separated field list | `--fields "key,summary,status"` |
| `--json` | Output as JSON (includes ADF) | `--json` |
| `--web` | Open in browser | `--web` |

**Field Specifiers:**

| Specifier | Description |
|-----------|-------------|
| `*all` | All fields |
| `*navigable` | Navigable fields only |
| `-fieldname` | Exclude specific field |
| `field1,field2` | Specific fields only |

**Examples:**

```bash
# Default fields
acli jira workitem view DEV-123

# Specific fields
acli jira workitem view DEV-123 --fields "key,summary,description,status"

# All fields as JSON
acli jira workitem view DEV-123 --fields "*all" --json

# Exclude description
acli jira workitem view DEV-123 --fields "*navigable,-description"
```

### List Comments

```bash
acli jira workitem comment list [flags]
```

**Flags:**

| Flag | Description | Default |
|------|-------------|---------|
| `--key` | Issue key (required) | - |
| `--json` | Output as JSON | false |
| `--limit` | Max comments per page | 50 |
| `--order` | Sort order | `+created` |
| `--paginate` | Fetch all pages | false |

**Order Options:**
- `+created` - Oldest first
- `-created` - Newest first
- `+updated` - Oldest updated first
- `-updated` - Newest updated first

**Examples:**

```bash
# List comments (default order)
acli jira workitem comment list --key DEV-123

# Latest 10 comments
acli jira workitem comment list --key DEV-123 --limit 10 --order "-created"

# All comments as JSON
acli jira workitem comment list --key DEV-123 --json --paginate

# JSON with limit
acli jira workitem comment list --key DEV-123 --json --limit 5
```

### List Attachments

```bash
acli jira workitem attachment list [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--key` | Issue key (required) |
| `--json` | Output as JSON |

**Examples:**

```bash
# List attachments
acli jira workitem attachment list --key DEV-123

# As JSON
acli jira workitem attachment list --key DEV-123 --json
```

### Search Issues

```bash
acli jira workitem search [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--jql` | JQL query string |
| `--fields` | Fields to return |
| `--json` | Output as JSON |
| `--limit` | Max results |

**Examples:**

```bash
# Search by JQL
acli jira workitem search --jql "project = DEV AND status = Open"

# With specific fields
acli jira workitem search --jql "assignee = currentUser()" --fields "key,summary,status"
```

## Output Formats

### Text Output (Default)

Human-readable format:

```
Key: DEV-123
Type: Story
Summary: Implement user authentication
Status: In Progress
Assignee: user@example.com
Description: [Rendered text content]
```

**Characteristics:**
- Compact (~5x smaller than JSON)
- Tables rendered as text (may lose structure)
- Code blocks rendered inline
- Suitable for simple issues

### JSON Output

Full JIRA API response:

```json
{
  "expand": "...",
  "id": "12345",
  "self": "https://...",
  "key": "DEV-123",
  "fields": {
    "summary": "...",
    "description": { /* ADF content */ },
    "status": { "name": "In Progress" },
    ...
  }
}
```

**Characteristics:**
- Complete field data
- ADF preserved for rich content
- Larger output size
- Required for structured content parsing

## Error Codes

| Error | Meaning | Resolution |
|-------|---------|------------|
| `401 Unauthorized` | Auth expired | Run `acli jira auth` |
| `404 Not Found` | Issue doesn't exist | Check key format |
| `403 Forbidden` | No access | Verify permissions |
| `429 Too Many Requests` | Rate limited | Wait and retry |

## Tips

1. **Use `--fields` aggressively** - Reduces response size and improves speed
2. **Default to text output** - Only use `--json` when parsing ADF
3. **Combine with `jq`** - Parse JSON output: `acli ... --json | jq '.fields.summary'`
4. **Check auth first** - Run `acli jira auth status` if errors occur
5. **Use `--paginate`** - For issues with many comments
