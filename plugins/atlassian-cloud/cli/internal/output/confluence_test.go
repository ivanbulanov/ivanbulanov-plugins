package output

import (
	"testing"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func TestConfluenceFormatPageSummary(t *testing.T) {
	tests := []struct {
		name string
		page *models.PageScheme
		want string
	}{
		{
			name: "all fields populated",
			page: &models.PageScheme{
				ID:        "12345",
				Title:     "My Page Title",
				SpaceID:   "SPACE1",
				Status:    "current",
				CreatedAt: "2025-01-15T10:30:00Z",
				Version: &models.PageVersionScheme{
					Number: 5,
				},
			},
			want: "**My Page Title**\n" +
				"**Page ID**: 12345 | **Space**: SPACE1\n" +
				"**Version**: 5\n" +
				"**Created**: 2025-01-15T10:30:00Z\n" +
				"**Status**: current\n",
		},
		{
			name: "nil version",
			page: &models.PageScheme{
				ID:        "99",
				Title:     "Draft Page",
				SpaceID:   "SP2",
				Status:    "draft",
				CreatedAt: "2025-06-01T00:00:00Z",
			},
			want: "**Draft Page**\n" +
				"**Page ID**: 99 | **Space**: SP2\n" +
				"**Created**: 2025-06-01T00:00:00Z\n" +
				"**Status**: draft\n",
		},
		{
			name: "empty created at",
			page: &models.PageScheme{
				ID:      "42",
				Title:   "No Date",
				SpaceID: "SP3",
				Status:  "current",
				Version: &models.PageVersionScheme{
					Number: 1,
				},
			},
			want: "**No Date**\n" +
				"**Page ID**: 42 | **Space**: SP3\n" +
				"**Version**: 1\n" +
				"**Status**: current\n",
		},
		{
			name: "minimal fields only",
			page: &models.PageScheme{
				Title: "Bare",
			},
			want: "**Bare**\n" +
				"**Page ID**:  | **Space**: \n" +
				"**Status**: \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPageSummary(tt.page)
			if got != tt.want {
				t.Errorf("FormatPageSummary():\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestConfluenceFormatPageBody(t *testing.T) {
	tests := []struct {
		name string
		page *models.PageScheme
		want string
	}{
		{
			name: "nil body",
			page: &models.PageScheme{
				Title: "No Body",
			},
			want: "\n*No body content*\n",
		},
		{
			name: "body with nil sub-fields",
			page: &models.PageScheme{
				Title: "Empty Body",
				Body:  &models.PageBodyScheme{},
			},
			want: "\n*Empty page*\n",
		},
		{
			name: "body with empty atlas doc format value",
			page: &models.PageScheme{
				Title: "Empty ADF",
				Body: &models.PageBodyScheme{
					AtlasDocFormat: &models.PageBodyRepresentationScheme{
						Value: "",
					},
				},
			},
			want: "\n*Empty page*\n",
		},
		{
			name: "body with valid atlas doc format",
			page: &models.PageScheme{
				Title: "ADF Page",
				Body: &models.PageBodyScheme{
					AtlasDocFormat: &models.PageBodyRepresentationScheme{
						Value: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Hello from ADF"}]}]}`,
					},
				},
			},
			want: "\n### Content\n\nHello from ADF\n",
		},
		{
			name: "body with storage format only",
			page: &models.PageScheme{
				Title: "Storage Page",
				Body: &models.PageBodyScheme{
					Storage: &models.PageBodyRepresentationScheme{
						Value: "<p>HTML storage content</p>",
					},
				},
			},
			// Not valid ADF JSON, so renderADF falls back to raw
			want: "\n### Content\n\n<p>HTML storage content</p>\n",
		},
		{
			name: "atlas doc format takes precedence over storage",
			page: &models.PageScheme{
				Title: "Both Formats",
				Body: &models.PageBodyScheme{
					AtlasDocFormat: &models.PageBodyRepresentationScheme{
						Value: `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"ADF wins"}]}]}`,
					},
					Storage: &models.PageBodyRepresentationScheme{
						Value: "<p>Storage loses</p>",
					},
				},
			},
			want: "\n### Content\n\nADF wins\n",
		},
		{
			name: "empty atlas doc format falls back to storage",
			page: &models.PageScheme{
				Title: "Fallback",
				Body: &models.PageBodyScheme{
					AtlasDocFormat: &models.PageBodyRepresentationScheme{
						Value: "",
					},
					Storage: &models.PageBodyRepresentationScheme{
						Value: "plain storage text",
					},
				},
			},
			// Not valid ADF JSON, so renderADF falls back to raw
			want: "\n### Content\n\nplain storage text\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatPageBody(tt.page)
			if got != tt.want {
				t.Errorf("FormatPageBody():\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestConfluenceFormatSearchResultsConfluence(t *testing.T) {
	tests := []struct {
		name    string
		results []*models.SearchResultScheme
		total   int
		want    string
	}{
		{
			name:    "empty results",
			results: nil,
			total:   0,
			want:    "**Found 0 results** (showing 0)\n\n",
		},
		{
			name:    "empty slice",
			results: []*models.SearchResultScheme{},
			total:   0,
			want:    "**Found 0 results** (showing 0)\n\n",
		},
		{
			name: "single result with URL",
			results: []*models.SearchResultScheme{
				{
					Title:   "Getting Started",
					Excerpt: "A guide to getting started with the platform.",
					URL:     "https://wiki.example.com/pages/123",
				},
			},
			total: 1,
			want: "**Found 1 results** (showing 1)\n\n" +
				"- **Getting Started** \u2014 A guide to getting started with the platform.\n" +
				"  URL: https://wiki.example.com/pages/123\n" +
				"\n",
		},
		{
			name: "multiple results mixed URLs",
			results: []*models.SearchResultScheme{
				{
					Title:   "Page A",
					Excerpt: "First result excerpt",
					URL:     "https://wiki.example.com/a",
				},
				{
					Title:   "Page B",
					Excerpt: "Second result excerpt",
					URL:     "",
				},
				{
					Title:   "Page C",
					Excerpt: "Third result excerpt",
					URL:     "https://wiki.example.com/c",
				},
			},
			total: 50,
			want: "**Found 50 results** (showing 3)\n\n" +
				"- **Page A** \u2014 First result excerpt\n" +
				"  URL: https://wiki.example.com/a\n" +
				"\n" +
				"- **Page B** \u2014 Second result excerpt\n" +
				"\n" +
				"- **Page C** \u2014 Third result excerpt\n" +
				"  URL: https://wiki.example.com/c\n" +
				"\n",
		},
		{
			name: "total greater than shown",
			results: []*models.SearchResultScheme{
				{
					Title:   "Only One",
					Excerpt: "Excerpt",
				},
			},
			total: 100,
			want: "**Found 100 results** (showing 1)\n\n" +
				"- **Only One** \u2014 Excerpt\n" +
				"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSearchResultsConfluence(tt.results, tt.total)
			if got != tt.want {
				t.Errorf("FormatSearchResultsConfluence():\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestConfluenceFormatConfluenceAttachments(t *testing.T) {
	tests := []struct {
		name        string
		attachments []*models.AttachmentScheme
		want        string
	}{
		{
			name:        "nil slice",
			attachments: nil,
			want:        "\n*No attachments*\n",
		},
		{
			name:        "empty slice",
			attachments: []*models.AttachmentScheme{},
			want:        "\n*No attachments*\n",
		},
		{
			name: "single attachment with download link",
			attachments: []*models.AttachmentScheme{
				{
					Title:        "report.pdf",
					FileSize:     2048576,
					MediaType:    "application/pdf",
					DownloadLink: "/download/attachments/123/report.pdf",
				},
			},
			want: "\n### Attachments (1)\n\n" +
				"- **report.pdf** (2.0 MB, application/pdf)\n" +
				"  Download: /download/attachments/123/report.pdf\n",
		},
		{
			name: "single attachment without download link",
			attachments: []*models.AttachmentScheme{
				{
					Title:     "logo.png",
					FileSize:  5120,
					MediaType: "image/png",
				},
			},
			want: "\n### Attachments (1)\n\n" +
				"- **logo.png** (5.0 KB, image/png)\n",
		},
		{
			name: "multiple attachments",
			attachments: []*models.AttachmentScheme{
				{
					Title:        "doc.txt",
					FileSize:     256,
					MediaType:    "text/plain",
					DownloadLink: "/download/doc.txt",
				},
				{
					Title:     "image.jpg",
					FileSize:  512000,
					MediaType: "image/jpeg",
				},
				{
					Title:        "data.csv",
					FileSize:     10240,
					MediaType:    "text/csv",
					DownloadLink: "/download/data.csv",
				},
			},
			want: "\n### Attachments (3)\n\n" +
				"- **doc.txt** (256 B, text/plain)\n" +
				"  Download: /download/doc.txt\n" +
				"- **image.jpg** (500.0 KB, image/jpeg)\n" +
				"- **data.csv** (10.0 KB, text/csv)\n" +
				"  Download: /download/data.csv\n",
		},
		{
			name: "zero byte attachment",
			attachments: []*models.AttachmentScheme{
				{
					Title:     "empty.txt",
					FileSize:  0,
					MediaType: "text/plain",
				},
			},
			want: "\n### Attachments (1)\n\n" +
				"- **empty.txt** (0 B, text/plain)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatConfluenceAttachments(tt.attachments)
			if got != tt.want {
				t.Errorf("FormatConfluenceAttachments():\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestConfluenceFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int
		want  string
	}{
		{name: "zero bytes", bytes: 0, want: "0 B"},
		{name: "one byte", bytes: 1, want: "1 B"},
		{name: "under 1KB", bytes: 512, want: "512 B"},
		{name: "exactly 1023 bytes", bytes: 1023, want: "1023 B"},
		{name: "exactly 1KB", bytes: 1024, want: "1.0 KB"},
		{name: "1.5KB", bytes: 1536, want: "1.5 KB"},
		{name: "just under 1MB", bytes: 1024*1024 - 1, want: "1024.0 KB"},
		{name: "exactly 1MB", bytes: 1024 * 1024, want: "1.0 MB"},
		{name: "10MB", bytes: 10 * 1024 * 1024, want: "10.0 MB"},
		{name: "1.5MB", bytes: 1536 * 1024, want: "1.5 MB"},
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
