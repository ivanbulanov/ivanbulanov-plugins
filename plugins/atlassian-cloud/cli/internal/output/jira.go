package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func FormatIssueSummary(issue *models.IssueSchemeV2) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**%s**: %s\n", issue.Key, issue.Fields.Summary))

	if issue.Fields.Status != nil {
		sb.WriteString(fmt.Sprintf("**Status**: %s", issue.Fields.Status.Name))
	}
	if issue.Fields.IssueType != nil {
		sb.WriteString(fmt.Sprintf(" | **Type**: %s", issue.Fields.IssueType.Name))
	}
	if issue.Fields.Priority != nil {
		sb.WriteString(fmt.Sprintf(" | **Priority**: %s", issue.Fields.Priority.Name))
	}
	sb.WriteString("\n")

	if issue.Fields.Assignee != nil {
		sb.WriteString(fmt.Sprintf("**Assignee**: %s\n", issue.Fields.Assignee.DisplayName))
	} else {
		sb.WriteString("**Assignee**: Unassigned\n")
	}

	if issue.Fields.Reporter != nil {
		sb.WriteString(fmt.Sprintf("**Reporter**: %s\n", issue.Fields.Reporter.DisplayName))
	}

	if issue.Fields.Project != nil {
		sb.WriteString(fmt.Sprintf("**Project**: %s (%s)\n", issue.Fields.Project.Name, issue.Fields.Project.Key))
	}

	if len(issue.Fields.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("**Labels**: %s\n", strings.Join(issue.Fields.Labels, ", ")))
	}

	if issue.Fields.Created != nil {
		sb.WriteString(fmt.Sprintf("**Created**: %s\n", formatDateTime(issue.Fields.Created)))
	}
	if issue.Fields.Updated != nil {
		sb.WriteString(fmt.Sprintf("**Updated**: %s\n", formatDateTime(issue.Fields.Updated)))
	}

	return sb.String()
}

func FormatIssueDescription(description string) string {
	if description == "" {
		return "\n*No description*\n"
	}

	// Try ADF parsing first
	md, err := ADFToMarkdown(description)
	if err == nil && md != "" {
		return fmt.Sprintf("\n### Description\n\n%s", md)
	}

	// Fallback: plain text
	return fmt.Sprintf("\n### Description\n\n%s\n", description)
}

func FormatComments(comments []*models.IssueCommentSchemeV2) string {
	if len(comments) == 0 {
		return "\n*No comments*\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n### Comments (%d)\n\n", len(comments)))

	for i, c := range comments {
		author := "Unknown"
		if c.Author != nil {
			author = c.Author.DisplayName
		}
		sb.WriteString(fmt.Sprintf("%d. **%s** (%s):\n", i+1, author, c.Created))

		body := c.Body
		if md, err := ADFToMarkdown(body); err == nil && md != "" {
			body = md
		}
		// Indent comment body
		for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
			sb.WriteString(fmt.Sprintf("   %s\n", line))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func FormatAttachments(attachments []*models.IssueAttachmentScheme) string {
	if len(attachments) == 0 {
		return "\n*No attachments*\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n### Attachments (%d)\n\n", len(attachments)))

	for _, a := range attachments {
		size := formatSize(a.Size)
		sb.WriteString(fmt.Sprintf("- **%s** (%s, %s)\n", a.Filename, size, a.MimeType))
		if a.Content != "" {
			sb.WriteString(fmt.Sprintf("  Download: %s\n", a.Content))
		}
	}

	return sb.String()
}

func FormatSearchResults(issues []*models.IssueSchemeV2, total int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Found %d issues** (showing %d)\n\n", total, len(issues)))

	sb.WriteString("| Key | Summary | Status | Assignee |\n")
	sb.WriteString("|-----|---------|--------|----------|\n")

	for _, issue := range issues {
		assignee := "Unassigned"
		if issue.Fields.Assignee != nil {
			assignee = issue.Fields.Assignee.DisplayName
		}
		status := ""
		if issue.Fields.Status != nil {
			status = issue.Fields.Status.Name
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			issue.Key,
			truncate(issue.Fields.Summary, 60),
			status,
			assignee,
		))
	}

	return sb.String()
}

func formatDateTime(dt *models.DateTimeScheme) string {
	if dt == nil {
		return ""
	}
	return time.Time(*dt).Format(time.RFC3339)
}

func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
