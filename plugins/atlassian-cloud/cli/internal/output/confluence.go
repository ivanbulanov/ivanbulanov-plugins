package output

import (
	"fmt"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func FormatPageSummary(page *models.PageScheme) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "**%s**\n", page.Title)
	fmt.Fprintf(&sb, "**Page ID**: %s | **Space**: %s\n", page.ID, page.SpaceID)

	if page.Version != nil {
		fmt.Fprintf(&sb, "**Version**: %d\n", page.Version.Number)
	}
	if page.CreatedAt != "" {
		fmt.Fprintf(&sb, "**Created**: %s\n", page.CreatedAt)
	}
	fmt.Fprintf(&sb, "**Status**: %s\n", page.Status)

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

	return renderADF(raw, "Content")
}

func FormatConfluenceAttachments(attachments []*models.AttachmentScheme) string {
	if len(attachments) == 0 {
		return "\n*No attachments*\n"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n### Attachments (%d)\n\n", len(attachments))

	for _, a := range attachments {
		size := formatSize(a.FileSize)
		fmt.Fprintf(&sb, "- **%s** (%s, %s)\n", a.Title, size, a.MediaType)
		if a.DownloadLink != "" {
			fmt.Fprintf(&sb, "  Download: %s\n", a.DownloadLink)
		}
	}

	return sb.String()
}

func FormatSearchResultsConfluence(results []*models.SearchResultScheme, total int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "**Found %d results** (showing %d)\n\n", total, len(results))

	for _, r := range results {
		fmt.Fprintf(&sb, "- **%s** — %s\n", r.Title, r.Excerpt)
		if r.URL != "" {
			fmt.Fprintf(&sb, "  URL: %s\n", r.URL)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
