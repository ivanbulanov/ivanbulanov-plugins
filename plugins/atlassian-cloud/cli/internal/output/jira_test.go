package output

import (
	"strings"
	"testing"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

// helper to create a *models.DateTimeScheme from a time.Time.
func dateTimePtr(t time.Time) *models.DateTimeScheme {
	dt := models.DateTimeScheme(t)
	return &dt
}

func TestJira_FormatIssueSummary(t *testing.T) {
	fixedTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	t.Run("full fields", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "PROJ-42",
			Fields: &models.IssueFieldsSchemeV2{
				Summary:   "Implement feature X",
				Status:    &models.StatusScheme{Name: "In Progress"},
				IssueType: &models.IssueTypeScheme{Name: "Story"},
				Priority:  &models.PriorityScheme{Name: "High"},
				Assignee:  &models.UserScheme{DisplayName: "Alice"},
				Reporter:  &models.UserScheme{DisplayName: "Bob"},
				Project:   &models.ProjectScheme{Name: "My Project", Key: "PROJ"},
				Labels:    []string{"backend", "urgent"},
				Created:   dateTimePtr(fixedTime),
				Updated:   dateTimePtr(fixedTime.Add(24 * time.Hour)),
			},
		}

		got := FormatIssueSummary(issue)

		expects := []string{
			"**PROJ-42**: Implement feature X",
			"**Status**: In Progress",
			"**Type**: Story",
			"**Priority**: High",
			"**Assignee**: Alice",
			"**Reporter**: Bob",
			"**Project**: My Project (PROJ)",
			"**Labels**: backend, urgent",
			"**Created**: 2024-06-15T10:30:00Z",
			"**Updated**: 2024-06-16T10:30:00Z",
		}

		for _, want := range expects {
			if !strings.Contains(got, want) {
				t.Errorf("expected output to contain %q, got:\n%s", want, got)
			}
		}
	})

	t.Run("nil Status", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "X-1",
			Fields: &models.IssueFieldsSchemeV2{
				Summary: "test",
			},
		}
		got := FormatIssueSummary(issue)
		if strings.Contains(got, "**Status**") {
			t.Errorf("should not contain Status when nil, got:\n%s", got)
		}
	})

	t.Run("nil IssueType", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "X-1",
			Fields: &models.IssueFieldsSchemeV2{
				Summary: "test",
				Status:  &models.StatusScheme{Name: "Open"},
			},
		}
		got := FormatIssueSummary(issue)
		if strings.Contains(got, "**Type**") {
			t.Errorf("should not contain Type when nil, got:\n%s", got)
		}
	})

	t.Run("nil Priority", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "X-1",
			Fields: &models.IssueFieldsSchemeV2{
				Summary: "test",
			},
		}
		got := FormatIssueSummary(issue)
		if strings.Contains(got, "**Priority**") {
			t.Errorf("should not contain Priority when nil, got:\n%s", got)
		}
	})

	t.Run("nil Assignee shows Unassigned", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "X-1",
			Fields: &models.IssueFieldsSchemeV2{
				Summary: "test",
			},
		}
		got := FormatIssueSummary(issue)
		if !strings.Contains(got, "**Assignee**: Unassigned") {
			t.Errorf("expected Unassigned when Assignee is nil, got:\n%s", got)
		}
	})

	t.Run("nil Reporter", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "X-1",
			Fields: &models.IssueFieldsSchemeV2{
				Summary: "test",
			},
		}
		got := FormatIssueSummary(issue)
		if strings.Contains(got, "**Reporter**") {
			t.Errorf("should not contain Reporter when nil, got:\n%s", got)
		}
	})

	t.Run("nil Project", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "X-1",
			Fields: &models.IssueFieldsSchemeV2{
				Summary: "test",
			},
		}
		got := FormatIssueSummary(issue)
		if strings.Contains(got, "**Project**") {
			t.Errorf("should not contain Project when nil, got:\n%s", got)
		}
	})

	t.Run("empty Labels", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "X-1",
			Fields: &models.IssueFieldsSchemeV2{
				Summary: "test",
				Labels:  []string{},
			},
		}
		got := FormatIssueSummary(issue)
		if strings.Contains(got, "**Labels**") {
			t.Errorf("should not contain Labels when empty, got:\n%s", got)
		}
	})

	t.Run("nil Created", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "X-1",
			Fields: &models.IssueFieldsSchemeV2{
				Summary: "test",
			},
		}
		got := FormatIssueSummary(issue)
		if strings.Contains(got, "**Created**") {
			t.Errorf("should not contain Created when nil, got:\n%s", got)
		}
	})

	t.Run("nil Updated", func(t *testing.T) {
		issue := &models.IssueSchemeV2{
			Key: "X-1",
			Fields: &models.IssueFieldsSchemeV2{
				Summary: "test",
			},
		}
		got := FormatIssueSummary(issue)
		if strings.Contains(got, "**Updated**") {
			t.Errorf("should not contain Updated when nil, got:\n%s", got)
		}
	})
}

func TestJira_FormatIssueDescription(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		got := FormatIssueDescription("")
		if got != "\n*No description*\n" {
			t.Errorf("expected no description marker, got: %q", got)
		}
	})

	t.Run("valid ADF JSON", func(t *testing.T) {
		adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello world"}]}]}`
		got := FormatIssueDescription(adf)
		if !strings.Contains(got, "### Description") {
			t.Errorf("expected Description heading, got: %q", got)
		}
		if !strings.Contains(got, "Hello world") {
			t.Errorf("expected converted ADF content, got: %q", got)
		}
	})

	t.Run("plain text falls back to raw", func(t *testing.T) {
		plain := "Just a plain description"
		got := FormatIssueDescription(plain)
		if !strings.Contains(got, "### Description") {
			t.Errorf("expected Description heading, got: %q", got)
		}
		if !strings.Contains(got, plain) {
			t.Errorf("expected plain text in output, got: %q", got)
		}
	})
}

func TestJira_FormatComments(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		got := FormatComments(nil)
		if got != "\n*No comments*\n" {
			t.Errorf("expected no comments marker, got: %q", got)
		}

		got2 := FormatComments([]*models.IssueCommentSchemeV2{})
		if got2 != "\n*No comments*\n" {
			t.Errorf("expected no comments marker for empty slice, got: %q", got2)
		}
	})

	t.Run("nil Author", func(t *testing.T) {
		comments := []*models.IssueCommentSchemeV2{
			{
				Author:  nil,
				Created: "2024-06-15T10:30:00.000+0000",
				Body:    "Some comment",
			},
		}
		got := FormatComments(comments)
		if !strings.Contains(got, "**Unknown**") {
			t.Errorf("expected Unknown author, got:\n%s", got)
		}
	})

	t.Run("multiple comments", func(t *testing.T) {
		comments := []*models.IssueCommentSchemeV2{
			{
				Author:  &models.UserScheme{DisplayName: "Alice"},
				Created: "2024-06-15T10:30:00.000+0000",
				Body:    "First comment",
			},
			{
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
		if !strings.Contains(got, "2. **Bob**") {
			t.Errorf("expected comment 2 by Bob, got:\n%s", got)
		}
		if !strings.Contains(got, "First comment") {
			t.Errorf("expected first comment body, got:\n%s", got)
		}
		if !strings.Contains(got, "Second comment") {
			t.Errorf("expected second comment body, got:\n%s", got)
		}
	})

	t.Run("ADF comment body", func(t *testing.T) {
		adf := `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"ADF body"}]}]}`
		comments := []*models.IssueCommentSchemeV2{
			{
				Author:  &models.UserScheme{DisplayName: "Charlie"},
				Created: "2024-01-01T00:00:00.000+0000",
				Body:    adf,
			},
		}
		got := FormatComments(comments)
		if !strings.Contains(got, "ADF body") {
			t.Errorf("expected ADF-converted body, got:\n%s", got)
		}
	})
}

func TestJira_FormatAttachments(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		got := FormatAttachments(nil)
		if got != "\n*No attachments*\n" {
			t.Errorf("expected no attachments marker, got: %q", got)
		}

		got2 := FormatAttachments([]*models.IssueAttachmentScheme{})
		if got2 != "\n*No attachments*\n" {
			t.Errorf("expected no attachments marker for empty slice, got: %q", got2)
		}
	})

	t.Run("multiple attachments with and without Content URL", func(t *testing.T) {
		attachments := []*models.IssueAttachmentScheme{
			{
				Filename: "screenshot.png",
				Size:     204800,
				MimeType: "image/png",
				Content:  "https://example.com/file/screenshot.png",
			},
			{
				Filename: "notes.txt",
				Size:     512,
				MimeType: "text/plain",
				// No Content URL
			},
		}
		got := FormatAttachments(attachments)

		if !strings.Contains(got, "### Attachments (2)") {
			t.Errorf("expected Attachments (2), got:\n%s", got)
		}
		if !strings.Contains(got, "**screenshot.png**") {
			t.Errorf("expected screenshot.png, got:\n%s", got)
		}
		if !strings.Contains(got, "200.0 KB") {
			t.Errorf("expected 200.0 KB size, got:\n%s", got)
		}
		if !strings.Contains(got, "Download: https://example.com/file/screenshot.png") {
			t.Errorf("expected download URL for screenshot, got:\n%s", got)
		}
		if !strings.Contains(got, "**notes.txt**") {
			t.Errorf("expected notes.txt, got:\n%s", got)
		}
		if !strings.Contains(got, "512 B") {
			t.Errorf("expected 512 B size, got:\n%s", got)
		}
		// notes.txt has no Content URL, so no Download line for it
		lines := strings.Split(got, "\n")
		notesDownloadFound := false
		for i, line := range lines {
			if strings.Contains(line, "notes.txt") {
				// Check that the next line is not a Download line for this file
				if i+1 < len(lines) && strings.Contains(lines[i+1], "Download:") {
					notesDownloadFound = true
				}
			}
		}
		if notesDownloadFound {
			t.Errorf("should not have Download line for notes.txt (no Content URL), got:\n%s", got)
		}
	})
}

func TestJira_FormatSearchResults(t *testing.T) {
	t.Run("empty results", func(t *testing.T) {
		got := FormatSearchResults(nil, 0)
		if !strings.Contains(got, "**Found 0 issues** (showing 0)") {
			t.Errorf("expected found 0 issues, got:\n%s", got)
		}
		// Table header should still be present
		if !strings.Contains(got, "| Key | Summary | Status | Assignee |") {
			t.Errorf("expected table header, got:\n%s", got)
		}
	})

	t.Run("multiple results with nil assignee and status", func(t *testing.T) {
		issues := []*models.IssueSchemeV2{
			{
				Key: "PROJ-1",
				Fields: &models.IssueFieldsSchemeV2{
					Summary:  "First issue",
					Status:   &models.StatusScheme{Name: "Open"},
					Assignee: &models.UserScheme{DisplayName: "Alice"},
				},
			},
			{
				Key: "PROJ-2",
				Fields: &models.IssueFieldsSchemeV2{
					Summary: "Second issue",
					// nil Status and nil Assignee
				},
			},
		}

		got := FormatSearchResults(issues, 50)

		if !strings.Contains(got, "**Found 50 issues** (showing 2)") {
			t.Errorf("expected found 50 issues showing 2, got:\n%s", got)
		}
		if !strings.Contains(got, "| PROJ-1 | First issue | Open | Alice |") {
			t.Errorf("expected PROJ-1 row, got:\n%s", got)
		}
		if !strings.Contains(got, "| PROJ-2 | Second issue |  | Unassigned |") {
			t.Errorf("expected PROJ-2 row with empty status and Unassigned, got:\n%s", got)
		}
	})

	t.Run("long summary gets truncated", func(t *testing.T) {
		longSummary := strings.Repeat("A", 100)
		issues := []*models.IssueSchemeV2{
			{
				Key: "PROJ-3",
				Fields: &models.IssueFieldsSchemeV2{
					Summary: longSummary,
				},
			},
		}
		got := FormatSearchResults(issues, 1)
		// The truncated version should be 57 chars + "..."
		truncated := strings.Repeat("A", 57) + "..."
		if !strings.Contains(got, truncated) {
			t.Errorf("expected truncated summary, got:\n%s", got)
		}
	})
}

func TestJira_formatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int
		want  string
	}{
		{name: "zero bytes", bytes: 0, want: "0 B"},
		{name: "below KB boundary", bytes: 1023, want: "1023 B"},
		{name: "exactly 1 KB", bytes: 1024, want: "1.0 KB"},
		{name: "below MB boundary", bytes: 1048575, want: "1024.0 KB"},
		{name: "exactly 1 MB", bytes: 1048576, want: "1.0 MB"},
		{name: "large MB value", bytes: 10485760, want: "10.0 MB"},
		{name: "500 bytes", bytes: 500, want: "500 B"},
		{name: "1536 bytes", bytes: 1536, want: "1.5 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestJira_truncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{
			name: "short string under max",
			s:    "hello",
			max:  10,
			want: "hello",
		},
		{
			name: "exactly at max",
			s:    "hello",
			max:  5,
			want: "hello",
		},
		{
			name: "longer than max",
			s:    "hello world!",
			max:  8,
			want: "hello...",
		},
		{
			name: "max less than 4 returns original",
			s:    "hello world",
			max:  3,
			want: "hello world",
		},
		{
			name: "max equals 4",
			s:    "hello world",
			max:  4,
			want: "h...",
		},
		{
			name: "UTF-8 CJK characters",
			s:    "你好世界测试字符串",
			max:  6,
			want: "你好世...",
		},
		{
			name: "CJK exactly at max",
			s:    "你好世",
			max:  3,
			want: "你好世",
		},
		{
			name: "empty string",
			s:    "",
			max:  5,
			want: "",
		},
		{
			name: "max equals 0 returns original (less than 4)",
			s:    "abc",
			max:  0,
			want: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}
