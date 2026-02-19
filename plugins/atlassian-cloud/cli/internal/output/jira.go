package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func FormatIssueSummary(issue *models.IssueSchemeV2) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "**%s**: %s\n", issue.Key, issue.Fields.Summary)

	if issue.Fields.Status != nil {
		fmt.Fprintf(&sb, "**Status**: %s", issue.Fields.Status.Name)
	}
	if issue.Fields.IssueType != nil {
		fmt.Fprintf(&sb, " | **Type**: %s", issue.Fields.IssueType.Name)
	}
	if issue.Fields.Priority != nil {
		fmt.Fprintf(&sb, " | **Priority**: %s", issue.Fields.Priority.Name)
	}
	sb.WriteString("\n")

	if issue.Fields.Assignee != nil {
		fmt.Fprintf(&sb, "**Assignee**: %s\n", issue.Fields.Assignee.DisplayName)
	} else {
		sb.WriteString("**Assignee**: Unassigned\n")
	}

	if issue.Fields.Reporter != nil {
		fmt.Fprintf(&sb, "**Reporter**: %s\n", issue.Fields.Reporter.DisplayName)
	}

	if issue.Fields.Project != nil {
		fmt.Fprintf(&sb, "**Project**: %s (%s)\n", issue.Fields.Project.Name, issue.Fields.Project.Key)
	}

	if len(issue.Fields.Labels) > 0 {
		fmt.Fprintf(&sb, "**Labels**: %s\n", strings.Join(issue.Fields.Labels, ", "))
	}

	if issue.Fields.Created != nil {
		fmt.Fprintf(&sb, "**Created**: %s\n", formatDateTime(issue.Fields.Created))
	}
	if issue.Fields.Updated != nil {
		fmt.Fprintf(&sb, "**Updated**: %s\n", formatDateTime(issue.Fields.Updated))
	}

	return sb.String()
}

func FormatIssueDescription(description string) string {
	if description == "" {
		return "\n*No description*\n"
	}
	return renderADF(description, "Description")
}

func FormatComments(comments []*models.IssueCommentSchemeV2) string {
	if len(comments) == 0 {
		return "\n*No comments*\n"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n### Comments (%d)\n\n", len(comments))

	for i, c := range comments {
		author := "Unknown"
		if c.Author != nil {
			author = c.Author.DisplayName
		}
		fmt.Fprintf(&sb, "%d. **%s** (%s):\n", i+1, author, c.Created)

		body := c.Body
		if md, err := ADFToMarkdown(body); err == nil && md != "" {
			body = md
		}
		for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
			fmt.Fprintf(&sb, "   %s\n", line)
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
	fmt.Fprintf(&sb, "\n### Attachments (%d)\n\n", len(attachments))

	for _, a := range attachments {
		size := formatSize(a.Size)
		fmt.Fprintf(&sb, "- **%s** (%s, %s)\n", a.Filename, size, a.MimeType)
		if a.Content != "" {
			fmt.Fprintf(&sb, "  Download: %s\n", a.Content)
		}
	}

	return sb.String()
}

func FormatSearchResults(issues []*models.IssueSchemeV2, total int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**Found %d issues** (showing %d)\n\n", total, len(issues))

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
		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n",
			issue.Key,
			truncate(issue.Fields.Summary, 60),
			status,
			assignee,
		)
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
