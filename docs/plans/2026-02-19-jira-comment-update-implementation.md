# Jira Comment Update Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `jira comment update` command that updates existing Jira issue comments, identified by comment URL or issue key + comment ID.

**Architecture:** Extend `JiraRef` to carry an optional `CommentID` extracted from URL query param `focusedCommentId`. Add a new cobra subcommand under `jira comment` that accepts 1-2 positional args and reuses `--body`/`--stdin` flags. Update `FormatComments` to display comment IDs.

**Tech Stack:** Go 1.25, cobra CLI, go-atlassian v2 (`CommentRichTextConnector.Update`)

---

### Task 1: Extend JiraRef to extract comment ID from URLs

**Files:**
- Modify: `plugins/atlassian-cloud/cli/internal/urlparse/urlparse.go`
- Modify: `plugins/atlassian-cloud/cli/internal/urlparse/urlparse_test.go`

**Step 1: Write the failing tests**

Add a `wantCommentID` field to the existing `TestParseJiraURL` test table and add new test cases. Replace the entire test function in `urlparse_test.go`:

```go
func TestParseJiraURL(t *testing.T) {
	tests := []struct {
		input         string
		wantKey       string
		wantSite      string
		wantCommentID string
		wantOk        bool
	}{
		{"DEV-123", "DEV-123", "", "", true},
		{"PROJ-1", "PROJ-1", "", "", true},
		{"https://acme-corp.atlassian.net/browse/ACME-5136", "ACME-5136", "acme-corp.atlassian.net", "", true},
		{"https://company.atlassian.net/browse/PROJ-456?focusedCommentId=123", "PROJ-456", "company.atlassian.net", "123", true},
		{"https://company.atlassian.net/browse/PROJ-456?focusedId=999", "PROJ-456", "company.atlassian.net", "999", true},
		{"https://company.atlassian.net/browse/PROJ-456?focusedCommentId=123&other=val", "PROJ-456", "company.atlassian.net", "123", true},
		{"not-a-ticket", "", "", "", false},
		{"", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, ok := ParseJiraRef(tt.input)
			if ok != tt.wantOk {
				t.Fatalf("ParseJiraRef(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if result.IssueKey != tt.wantKey {
				t.Errorf("IssueKey = %q, want %q", result.IssueKey, tt.wantKey)
			}
			if result.Site != tt.wantSite {
				t.Errorf("Site = %q, want %q", result.Site, tt.wantSite)
			}
			if result.CommentID != tt.wantCommentID {
				t.Errorf("CommentID = %q, want %q", result.CommentID, tt.wantCommentID)
			}
		})
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd plugins/atlassian-cloud/cli && go test ./internal/urlparse/ -run TestParseJiraURL -v`
Expected: FAIL — `result.CommentID undefined`

**Step 3: Implement the changes**

In `urlparse.go`, add `CommentID` to `JiraRef`:

```go
type JiraRef struct {
	IssueKey  string
	Site      string
	CommentID string
}
```

Update `ParseJiraRef` to extract `focusedCommentId` (or `focusedId`) from URL query params. Replace the URL branch (the `return` statement at the end) with:

```go
	commentID := u.Query().Get("focusedCommentId")
	if commentID == "" {
		commentID = u.Query().Get("focusedId")
	}

	return JiraRef{
		IssueKey:  matches[1],
		Site:      u.Host,
		CommentID: commentID,
	}, true
```

**Step 4: Run tests to verify they pass**

Run: `cd plugins/atlassian-cloud/cli && go test ./internal/urlparse/ -run TestParseJiraURL -v`
Expected: PASS

**Step 5: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/urlparse/urlparse.go plugins/atlassian-cloud/cli/internal/urlparse/urlparse_test.go
git commit -m "feat: extract comment ID from Jira URLs in ParseJiraRef"
```

---

### Task 2: Update FormatComments to display comment IDs

**Files:**
- Modify: `plugins/atlassian-cloud/cli/internal/output/jira.go:62-88` (FormatComments function)
- Modify: `plugins/atlassian-cloud/cli/internal/output/jira_test.go` (TestJira_FormatComments)

**Step 1: Update the failing tests**

In `jira_test.go`, update the existing test cases in `TestJira_FormatComments` to expect comment IDs. Specifically:

In the `"nil Author"` subtest, add `ID: "10001"` to the comment and check for `[ID: 10001]`:

```go
	t.Run("nil Author", func(t *testing.T) {
		comments := []*models.IssueCommentSchemeV2{
			{
				ID:      "10001",
				Author:  nil,
				Created: "2024-06-15T10:30:00.000+0000",
				Body:    "Some comment",
			},
		}
		got := FormatComments(comments)
		if !strings.Contains(got, "**Unknown**") {
			t.Errorf("expected Unknown author, got:\n%s", got)
		}
		if !strings.Contains(got, "[ID: 10001]") {
			t.Errorf("expected comment ID in output, got:\n%s", got)
		}
	})
```

In the `"multiple comments"` subtest, add IDs and check them:

```go
	t.Run("multiple comments", func(t *testing.T) {
		comments := []*models.IssueCommentSchemeV2{
			{
				ID:      "10001",
				Author:  &models.UserScheme{DisplayName: "Alice"},
				Created: "2024-06-15T10:30:00.000+0000",
				Body:    "First comment",
			},
			{
				ID:      "10002",
				Author:  &models.UserScheme{DisplayName: "Bob"},
				Created: "2024-06-16T11:00:00.000+0000",
				Body:    "Second comment",
			},
		}
		got := FormatComments(comments)

		if !strings.Contains(got, "### Comments (2)") {
			t.Errorf("expected Comments (2), got:\n%s", got)
		}
		if !strings.Contains(got, "1. **Alice**") {
			t.Errorf("expected comment 1 by Alice, got:\n%s", got)
		}
		if !strings.Contains(got, "[ID: 10001]") {
			t.Errorf("expected comment ID 10001, got:\n%s", got)
		}
		if !strings.Contains(got, "2. **Bob**") {
			t.Errorf("expected comment 2 by Bob, got:\n%s", got)
		}
		if !strings.Contains(got, "[ID: 10002]") {
			t.Errorf("expected comment ID 10002, got:\n%s", got)
		}
		if !strings.Contains(got, "First comment") {
			t.Errorf("expected first comment body, got:\n%s", got)
		}
		if !strings.Contains(got, "Second comment") {
			t.Errorf("expected second comment body, got:\n%s", got)
		}
	})
```

**Step 2: Run tests to verify they fail**

Run: `cd plugins/atlassian-cloud/cli && go test ./internal/output/ -run TestJira_FormatComments -v`
Expected: FAIL — missing `[ID: 10001]`

**Step 3: Implement the change**

In `jira.go`, update the `FormatComments` format line (line 75) from:

```go
		fmt.Fprintf(&sb, "%d. **%s** (%s):\n", i+1, author, c.Created)
```

to:

```go
		fmt.Fprintf(&sb, "%d. **%s** (%s) [ID: %s]:\n", i+1, author, c.Created, c.ID)
```

**Step 4: Run tests to verify they pass**

Run: `cd plugins/atlassian-cloud/cli && go test ./internal/output/ -run TestJira_FormatComments -v`
Expected: PASS

**Step 5: Commit**

```bash
git add plugins/atlassian-cloud/cli/internal/output/jira.go plugins/atlassian-cloud/cli/internal/output/jira_test.go
git commit -m "feat: display comment IDs in FormatComments output"
```

---

### Task 3: Add jira comment update command

**Files:**
- Modify: `plugins/atlassian-cloud/cli/cmd/jira_comment.go`

**Step 1: Add the cobra command and handler**

Add the following after the `jiraCommentAddCmd` variable declaration (after line 34):

```go
var jiraCommentUpdateCmd = &cobra.Command{
	Use:   "update <issue-key-or-url> [comment-id]",
	Short: "Update a comment on an issue",
	Long: `Update an existing comment. Provide the comment ID as a second argument,
or use a Jira URL with focusedCommentId query parameter.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runJiraCommentUpdate,
}
```

Register it in `init()` — add after line 44 (`jiraCommentCmd.AddCommand(jiraCommentAddCmd)`):

```go
	jiraCommentCmd.AddCommand(jiraCommentUpdateCmd)
	jiraCommentUpdateCmd.Flags().StringVar(&commentBody, "body", "", "Comment text")
	jiraCommentUpdateCmd.Flags().BoolVar(&commentStdin, "stdin", false, "Read comment body from stdin")
```

Add the handler function after `runJiraCommentAdd`:

```go
func runJiraCommentUpdate(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	commentID := ref.CommentID
	if len(args) > 1 {
		commentID = args[1]
	}
	if commentID == "" {
		return fmt.Errorf("comment ID required (provide as second argument or use a URL with focusedCommentId)")
	}

	body := commentBody
	if commentStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		body = string(data)
	}

	if body == "" {
		return fmt.Errorf("comment body required (use --body or --stdin)")
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	payload := &models.CommentPayloadSchemeV2{Body: body}
	comment, response, err := clients.Jira.Issue.Comment.Update(context.Background(), ref.IssueKey, commentID, payload, nil)
	if err != nil {
		code := 0
		if response != nil {
			code = response.Code
		}
		return auth.WrapAPIError(code, fmt.Errorf("cannot update comment: %w", err))
	}

	fmt.Printf("Comment updated (ID: %s)\n", comment.ID)
	return nil
}
```

**Step 2: Build to verify compilation**

Run: `cd plugins/atlassian-cloud/cli && go build -o /dev/null .`
Expected: success (exit 0)

**Step 3: Run all existing tests to verify nothing is broken**

Run: `cd plugins/atlassian-cloud/cli && go test ./...`
Expected: all PASS

**Step 4: Commit**

```bash
git add plugins/atlassian-cloud/cli/cmd/jira_comment.go
git commit -m "feat: add jira comment update command"
```

---

### Task 4: Update documentation

**Files:**
- Modify: `plugins/atlassian-cloud/references/cli-commands.md`
- Modify: `plugins/atlassian-cloud/skills/atlassian-jira/SKILL.md`

**Step 1: Update cli-commands.md**

In the Jira Commands table, add after the `jira comment add` row:

```
| `jira comment update <key\|url> [comment-id]` | Update comment |
```

Add a new section after "### Comment Add Flags":

```markdown
### Comment Update Flags

| Flag | Description |
|------|-------------|
| `--body` | New comment text |
| `--stdin` | Read from stdin |
```

**Step 2: Update SKILL.md**

In the skill description on line 3, add "update comment" to the trigger list. Change:

```
description: Use when the user asks to "fetch JIRA issue", "get ticket", "show DEV-123", "look up issue", "search jira", "find tickets", "comment on ticket", "add comment to issue", or pastes a JIRA URL like "https://company.atlassian.net/browse/KEY-123". Also triggers on bare issue keys like "DEV-123" or "PROJ-456" in the user's message.
```

to:

```
description: Use when the user asks to "fetch JIRA issue", "get ticket", "show DEV-123", "look up issue", "search jira", "find tickets", "comment on ticket", "add comment to issue", "update comment", "edit comment", or pastes a JIRA URL like "https://company.atlassian.net/browse/KEY-123". Also triggers on bare issue keys like "DEV-123" or "PROJ-456" in the user's message.
```

In the Comments section (after line 167), add:

```markdown

### Update a comment
```bash
$ATLASSIAN_CLI jira comment update KEY-123 10042 --body "Updated analysis"
```

Using a comment URL:
```bash
$ATLASSIAN_CLI jira comment update "https://company.atlassian.net/browse/KEY-123?focusedCommentId=10042" --body "Updated text"
```

For longer updates:
```bash
echo "Revised detailed analysis..." | $ATLASSIAN_CLI jira comment update KEY-123 10042 --stdin
```
```

**Step 3: Commit**

```bash
git add plugins/atlassian-cloud/references/cli-commands.md plugins/atlassian-cloud/skills/atlassian-jira/SKILL.md
git commit -m "docs: add jira comment update to CLI reference and skill docs"
```

---

### Task 5: Run full test suite and verify build

**Step 1: Build the binary**

Run: `cd plugins/atlassian-cloud/cli && go build -o bin/atlassian-cloud .`
Expected: success

**Step 2: Run full test suite**

Run: `cd plugins/atlassian-cloud/cli && go test ./... -v`
Expected: all PASS

**Step 3: Verify CLI help shows the new command**

Run: `cd plugins/atlassian-cloud/cli && ./bin/atlassian-cloud jira comment --help`
Expected: output includes `update` in the Available Commands list

Run: `cd plugins/atlassian-cloud/cli && ./bin/atlassian-cloud jira comment update --help`
Expected: shows usage `update <issue-key-or-url> [comment-id]` with `--body` and `--stdin` flags
