---
name: atlassian-confluence
description: Use when the user asks to "read confluence page", "get wiki page", "search confluence", "find in confluence", "look up documentation", or pastes a Confluence URL like "https://company.atlassian.net/wiki/spaces/SPACE/pages/123456/Page+Title". Also triggers when the user mentions Confluence page IDs or asks about company documentation on Confluence.
version: 0.1.1
---

# Confluence Page Operations via atlassian-cloud CLI

Read Confluence Cloud pages and search content with progressive disclosure for context efficiency.

## Prerequisites

Use the **base directory** from the skill metadata header above to derive paths.

Ensure the CLI binary is built (the script auto-detects paths):

```bash
<base-directory>/../atlassian-jira/scripts/ensure-binary.sh
```

Set the CLI path (3 levels up from skill base to plugin root, then into cli/bin):
```bash
ATLASSIAN_CLI="<base-directory>/../../../cli/bin/atlassian-cloud"
```

## Authentication

Check auth first:
```bash
$ATLASSIAN_CLI auth status
```

If not authenticated, follow the **Guided API Token Setup** in the atlassian-jira skill — the same auth config covers both Jira and Confluence.

## Extracting Page References

| Input | Extracted |
|-------|-----------|
| `https://acme-corp.atlassian.net/wiki/spaces/ENG/pages/1234567890/API+Guidelines` | Page ID: 1234567890 (site: acme-corp.atlassian.net) |
| `https://co.atlassian.net/wiki/spaces/DEV/pages/123456` | Page ID: 123456 |
| `1234567890` | Page ID: 1234567890 |

When a URL includes the site hostname, pass it with `--site`:
```bash
$ATLASSIAN_CLI --site acme-corp.atlassian.net confluence page get 1234567890
```

## Progressive Disclosure — ALWAYS Start at Level 1

### Level 1: Summary (default)

```bash
$ATLASSIAN_CLI confluence page get <page-id-or-url>
```

Returns: Title, Page ID, Space, Version, Created date, Status.

### Level 2: + Body

```bash
$ATLASSIAN_CLI confluence page get <page-id-or-url> --body
```

Adds full page content (ADF converted to markdown).

### Level 3: + Attachments

```bash
$ATLASSIAN_CLI confluence page get <page-id-or-url> --body --attachments
```

Adds attachment list.

## Searching Pages

```bash
# Full-text search
$ATLASSIAN_CLI confluence search "deployment runbook" --max 10

# Search within a specific space
$ATLASSIAN_CLI confluence search "API documentation" --space WS --max 10
```

Returns: Title, excerpt, URL for each result.

## Presenting Results

### For page summaries
Present the CLI output directly.

### For full page content
Structure with headers:
```
## Page Title

**Space**: WS | **Version**: 4 | **Updated**: 2026-02-19

### Content
[markdown-converted page body]

### Attachments (3)
- diagram.png (1.2 MB)
- spec.pdf (450 KB)
```

## Error Handling

| Error | Action |
|-------|--------|
| Exit code 2 | Auth expired → `$ATLASSIAN_CLI auth login` |
| "page not found" | Verify page ID, check permissions |
| "invalid page ID" | Ensure numeric ID extracted from URL |

## Tips

1. **Always start at Level 1** — page bodies can be very large
2. **Parse URLs automatically** — the CLI handles both URLs and numeric IDs
3. **Use --space for targeted search** — narrows results significantly
4. **Search returns excerpts** — use page get for full content
