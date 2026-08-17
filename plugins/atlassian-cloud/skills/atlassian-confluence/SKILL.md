---
name: atlassian-confluence
description: Use when the user asks to "read confluence page", "get wiki page", "search confluence", "find in confluence", "look up documentation", or pastes a Confluence URL like "https://company.atlassian.net/wiki/spaces/SPACE/pages/123456/Page+Title". Also triggers when the user mentions Confluence page IDs or asks about company documentation on Confluence. This skill only reads — for writing a Markdown document onto a page, use the confluence-publish skill instead, even when the request also pastes a page URL.
---

# Confluence Page Operations via atlassian-cloud CLI

Read Confluence Cloud pages and search content with progressive disclosure for context efficiency.

**This skill reads; it does not write.** If the user wants to put a Markdown
document onto a page — "publish this design", "update the page from this doc",
"sync the design doc" — stop and use the **confluence-publish** skill, which
handles cross-reference linking, diagram rendering, and the drift check. Use
this skill first only to find the page id, then hand over.

## Setup

Run once at the start of any operation:

```bash
${CLAUDE_PLUGIN_ROOT}/scripts/setup.sh
```

This builds the CLI if needed and checks authentication. If it fails with a Go error, guide the user to install Go (`mise install go@latest` or https://go.dev/dl/).

**CLI path**: `${CLAUDE_PLUGIN_ROOT}/cli/bin/atlassian-cloud` — shown as `atlassian-cloud` in examples below.

If not authenticated, follow the **Guided API Token Setup** in the atlassian-jira skill — the same auth config covers both Jira and Confluence.

**Sandboxing:** If you have enabled Claude Code's Bash sandbox (it is off by default), the build step and the CLI need network access. See the plugin README's **Sandboxing** section for the `settings.json` allowlist.

## Extracting Page References

| Input | Extracted |
|-------|-----------|
| `https://acme-corp.atlassian.net/wiki/spaces/ENG/pages/1234567890/API+Guidelines` | Page ID: 1234567890 (site: acme-corp.atlassian.net) |
| `https://co.atlassian.net/wiki/spaces/DEV/pages/123456` | Page ID: 123456 |
| `1234567890` | Page ID: 1234567890 |

When a URL includes the site hostname, pass it with `--site`:
```bash
atlassian-cloud --site acme-corp.atlassian.net confluence page get 1234567890
```

## Progressive Disclosure — ALWAYS Start at Level 1

### Level 1: Summary (default)

```bash
atlassian-cloud confluence page get <page-id-or-url>
```

Returns: Title, Page ID, Space, Version, Created date, Status.

### Level 2: + Body

```bash
atlassian-cloud confluence page get <page-id-or-url> --body
```

Adds full page content (ADF converted to markdown).

### Level 3: + Attachments

```bash
atlassian-cloud confluence page get <page-id-or-url> --body --attachments
```

Adds attachment list.

## Searching Pages

```bash
# Full-text search
atlassian-cloud confluence search "deployment runbook" --max 10

# Search within a specific space
atlassian-cloud confluence search "API documentation" --space ENG --max 10
```

Returns: Title, excerpt, URL for each result.

## Downloading Attachments

### Single attachment
```bash
atlassian-cloud confluence attachment download <page-id-or-url> diagram.png
```

Downloads to a temp directory and prints the file path.

### All attachments
```bash
atlassian-cloud confluence attachment download <page-id-or-url> --all
```

### Save to specific directory
```bash
atlassian-cloud confluence attachment download <page-id-or-url> diagram.png --output-dir ./downloads
atlassian-cloud confluence attachment download <page-id-or-url> --all --output-dir ./downloads
```

Output: one file path per line. The agent can use the Read tool to view downloaded files.

### View page images with their placement

To see a page's images and where they appear in the document:

1. Fetch body and attachments:
   ```bash
   atlassian-cloud confluence page get <page-id-or-url> --body --attachments
   ```
   In the rendered body, images appear in document order as `![<filename>](attachment:<id>)`. The alt text is the attachment filename; surrounding headings and table cells show where each image sits on the page.

2. Download the image files:
   ```bash
   atlassian-cloud confluence attachment download <page-id-or-url> --all --output-dir <dir>
   ```

3. Use the Read tool on each downloaded file to view it, and match it to its placement using the filename from step 1.

**Note:** Works with scoped API tokens — downloads use the REST content endpoint and follow the media redirect without leaking credentials.

## Presenting Results

### For page summaries
Present the CLI output directly.

### For full page content
Structure with headers:
```
## Page Title

**Space**: ENG | **Version**: 4 | **Updated**: 2026-02-19

### Content
[markdown-converted page body]

### Attachments (3)
- diagram.png (1.2 MB)
- spec.pdf (450 KB)
```

## Error Handling

| Error | Action |
|-------|--------|
| Exit code 2 | Auth expired; run `atlassian-cloud auth login` |
| "page not found" | Verify page ID, check permissions |
| "invalid page ID" | Ensure numeric ID extracted from URL |
| Build failure | Ensure Go is installed: `go version`. If the sandbox blocked it, relay the `settings.json` entry `setup.sh` printed; never edit the user's settings yourself |

## Tips

1. **Always start at Level 1** — page bodies can be very large
2. **Parse URLs automatically** — the CLI handles both URLs and numeric IDs
3. **Use --space for targeted search** — narrows results significantly
4. **Search returns excerpts** — use page get for full content
