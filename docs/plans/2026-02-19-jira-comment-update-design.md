# Jira Comment Update Feature Design

## Overview

Add a `jira comment update` command to the atlassian-cloud CLI plugin, allowing users to update existing Jira issue comments.

## CLI Interface

```
jira comment update <issue-key-or-url> [comment-id] [flags]

Flags:
  --body   string   New comment text
  --stdin  bool     Read new comment body from stdin
```

### Usage Examples

```bash
# By comment URL (extracts issue key + comment ID from focusedId query param)
jira comment update "https://company.atlassian.net/browse/KEY-123?focusedId=10042" --body "Updated text"

# By issue key + comment ID
jira comment update KEY-123 10042 --body "Updated text"

# Via stdin
echo "Updated text" | jira comment update KEY-123 10042 --stdin
```

## Changes

### 1. `internal/urlparse/urlparse.go`

- Add `CommentID string` field to `JiraRef`
- In `ParseJiraRef`, extract `focusedId` query parameter from URLs and populate `CommentID`

### 2. `cmd/jira_comment.go`

- Add `jiraCommentUpdateCmd` cobra command accepting 1-2 positional args
- Add `runJiraCommentUpdate` function:
  - Parse first arg as issue reference (may include comment ID from URL)
  - If second arg provided, use it as comment ID (overrides URL-extracted ID)
  - Validate comment ID is present
  - Read body from `--body` or `--stdin`
  - Call `clients.Jira.Issue.Comment.Update()`
  - Print `Comment updated (ID: <id>)`

### 3. `internal/output/jira.go`

- Update `FormatComments` to include comment ID in output
- Format: `1. **Author** (date) [ID: 10042]:`

### 4. Documentation

- `references/cli-commands.md` — Add update command to table
- `skills/atlassian-jira/SKILL.md` — Add update examples to Comments section
- `README.md` — Add update to feature list if applicable

### 5. Tests

- `internal/urlparse/urlparse_test.go` — Test `ParseJiraRef` with `focusedId` query param
- `cmd/cmd_test.go` — Test command registration for the new subcommand

## API

Uses the existing `CommentRichTextConnector.Update` method from go-atlassian:

```go
Update(ctx context.Context, issueKeyOrID, commentID string, payload *CommentPayloadSchemeV2, expand []string) (*IssueCommentSchemeV2, *ResponseScheme, error)
```

## Output

Success: `Comment updated (ID: 10042)`
Error: Standard error wrapping with auth error handling, consistent with other commands.
