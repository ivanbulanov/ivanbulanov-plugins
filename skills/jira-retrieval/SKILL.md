---
name: jira-retrieval
description: This skill should be used when the user asks to "fetch JIRA issue", "get ticket", "show DEV-123", "retrieve issue details", "get comments for ticket", "show attachments", or pastes a JIRA URL like "https://yoursite.atlassian.net/browse/KEY-123". Also triggers when the user asks to "understand the requirements", "what does the ticket say", or needs context from a JIRA issue.
version: 0.1.0
---

# JIRA Issue Retrieval with acli

Retrieve JIRA issues, comments, and attachments using the Atlassian CLI (acli) with token-efficient output and intelligent content parsing.

## Overview

This skill uses acli to fetch JIRA data with an adaptive strategy:
- **Text output** (default): 5x smaller, suitable for simple issues
- **JSON output**: Full Atlassian Document Format (ADF) when structure matters

## Prerequisites

Ensure acli is installed and authenticated:

```bash
# Check installation
acli --version

# Authenticate if needed
acli jira auth
```

## Extracting Issue Keys

When the user provides a JIRA URL or mentions an issue, extract the key:

| Input | Extracted Key |
|-------|---------------|
| `DEV-123` | DEV-123 |
| `https://social.atlassian.net/browse/DEV-123` | DEV-123 |
| `https://yoursite.atlassian.net/browse/PROJ-456?focusedCommentId=123` | PROJ-456 |
| "ticket DEV-789" | DEV-789 |

Pattern: Issue keys are uppercase letters followed by a hyphen and numbers (e.g., `[A-Z]+-\d+`).

## Basic Retrieval Workflow

### Step 1: Fetch Issue (Text Output - Token Efficient)

```bash
acli jira workitem view KEY-123
```

Returns rendered text output with key fields. Use this for most cases.

### Step 2: Assess Content Complexity

Check the text output for indicators of complex content:
- Tables (column headers, data rows)
- Code blocks (syntax, formatting)
- Embedded images or attachments
- Detailed structured data

**If complex content detected**: Re-fetch with JSON output and parse ADF.

### Step 3: Fetch Comments (When Needed)

```bash
acli jira workitem comment list --key KEY-123
```

For full comment details with ADF:

```bash
acli jira workitem comment list --key KEY-123 --json
```

### Step 4: Fetch Attachments (When Needed)

```bash
acli jira workitem attachment list --key KEY-123
```

Returns attachment metadata (filename, size, download URL).

## Adaptive Output Strategy

### When to Use Text Output (Default)

Use text output when:
- Issue contains primarily prose text
- No tables, code blocks, or complex formatting
- Token efficiency is prioritized
- Quick overview is sufficient

```bash
acli jira workitem view KEY-123
```

### When to Use JSON + ADF Parsing

Switch to JSON when the text output reveals:
- Tables (data appears misaligned or flattened)
- Code blocks (formatting lost)
- Structured data (acceptance criteria, specs)
- User explicitly requests "full" or "detailed" content

```bash
acli jira workitem view KEY-123 --json
```

Then parse ADF using the converter script:

```bash
acli jira workitem view KEY-123 --json | ${CLAUDE_PLUGIN_ROOT}/skills/jira-retrieval/scripts/adf-to-markdown.sh
```

## Field Selection

Control which fields to retrieve using `--fields`:

```bash
# Minimal (fastest)
acli jira workitem view KEY-123 --fields "key,summary,status"

# Standard (default)
acli jira workitem view KEY-123 --fields "key,issuetype,summary,status,assignee,description"

# With comments
acli jira workitem view KEY-123 --fields "key,summary,description,status,comment"

# All fields
acli jira workitem view KEY-123 --fields "*all"
```

For field reference details, consult `references/field-mappings.md`.

## Common Patterns

### Pattern 1: Quick Issue Summary

```bash
acli jira workitem view KEY-123 --fields "key,summary,status,assignee"
```

Use for status checks or assignment queries.

### Pattern 2: Full Context Retrieval

```bash
# Issue with description
acli jira workitem view KEY-123

# Add comments
acli jira workitem comment list --key KEY-123

# Add attachments
acli jira workitem attachment list --key KEY-123
```

Use when user needs complete understanding of the issue.

### Pattern 3: Structured Content (Tables, Code)

```bash
# Fetch JSON
acli jira workitem view KEY-123 --json > /tmp/issue.json

# Parse ADF to markdown
cat /tmp/issue.json | ${CLAUDE_PLUGIN_ROOT}/skills/jira-retrieval/scripts/adf-to-markdown.sh
```

Use when text output loses important structure.

## Presenting Results

### For Simple Issues

Present key information directly:

```
**KEY-123**: Issue Summary
**Status**: In Progress
**Assignee**: user@example.com

**Description**:
[Issue description text]
```

### For Complex Issues

Structure the response:

```
## KEY-123: Issue Summary

**Status**: In Progress | **Type**: Story | **Assignee**: user@example.com

### Description
[Parsed markdown content with tables/code preserved]

### Comments (3)
1. **Author** (2024-01-15): Comment text...
2. **Author** (2024-01-16): Comment text...

### Attachments (2)
- screenshot.png (245 KB)
- design-doc.pdf (1.2 MB)
```

## Error Handling

### Authentication Errors

If acli returns authentication errors:

```bash
acli jira auth
```

Then retry the command.

### Issue Not Found

If the issue key is invalid or inaccessible, inform the user:
- Check the issue key format
- Verify access permissions
- Confirm the JIRA instance

### Network Errors

Retry once on network failures. If persistent, inform user of connectivity issues.

## Additional Resources

### Reference Files

For detailed information, consult:
- **`references/adf-format.md`** - Atlassian Document Format structure and parsing
- **`references/acli-commands.md`** - Complete acli command reference
- **`references/field-mappings.md`** - All available JIRA fields

### Scripts

- **`scripts/adf-to-markdown.sh`** - Convert ADF JSON to readable markdown

### Examples

Working examples in `examples/`:
- **`basic-retrieval.md`** - Simple issue fetch workflow
- **`full-context.md`** - Complete issue with comments and attachments

## Tips

1. **Start with text output** - Only switch to JSON when structure is needed
2. **Use field selection** - Reduce output size by requesting only needed fields
3. **Parse URLs automatically** - Extract issue keys from any Atlassian URL format
4. **Cache complex issues** - For long issues, fetch once and reference the result
5. **Combine commands** - Fetch issue, comments, and attachments in parallel when full context needed
