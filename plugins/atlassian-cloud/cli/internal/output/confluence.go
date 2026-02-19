package output

import (
	"fmt"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func FormatPageSummary(page *models.PageScheme) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**%s**\n", page.Title))
	sb.WriteString(fmt.Sprintf("**Page ID**: %s | **Space**: %s\n", page.ID, page.SpaceID))

	if page.Version != nil {
		sb.WriteString(fmt.Sprintf("**Version**: %d\n", page.Version.Number))
	}
	if page.CreatedAt != "" {
		sb.WriteString(fmt.Sprintf("**Created**: %s\n", page.CreatedAt))
	}
	sb.WriteString(fmt.Sprintf("**Status**: %s\n", page.Status))

	return sb.String()
}

func FormatPageBody(page *models.PageScheme) string {
	if page.Body == nil {
		return "\n*No body content*\n"
	}

	var raw string
	if page.Body.AtlasDocFormat != nil && page.Body.AtlasDocFormat.Value != "" {
		raw = page.Body.AtlasDocFormat.Value
	} else if page.Body.Storage != nil && page.Body.Storage.Value != "" {
		raw = page.Body.Storage.Value
	}

	if raw == "" {
		return "\n*Empty page*\n"
	}

	// Try ADF conversion
	md, err := ADFToMarkdown(raw)
	if err == nil && md != "" {
		return fmt.Sprintf("\n### Content\n\n%s", md)
	}

	// Fallback: raw content (may be storage format HTML)
	return fmt.Sprintf("\n### Content\n\n%s\n", raw)
}

func FormatConfluenceAttachments(attachments []*models.AttachmentScheme) string {
	if len(attachments) == 0 {
		return "\n*No attachments*\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n### Attachments (%d)\n\n", len(attachments)))

	for _, a := range attachments {
		size := formatSize(a.FileSize)
		sb.WriteString(fmt.Sprintf("- **%s** (%s, %s)\n", a.Title, size, a.MediaType))
		if a.DownloadLink != "" {
			sb.WriteString(fmt.Sprintf("  Download: %s\n", a.DownloadLink))
		}
	}

	return sb.String()
}

func FormatSearchResultsConfluence(results []*models.SearchResultScheme, total int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("**Found %d results** (showing %d)\n\n", total, len(results)))

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("- **%s** — %s\n", r.Title, r.Excerpt))
		if r.URL != "" {
			sb.WriteString(fmt.Sprintf("  URL: %s\n", r.URL))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
