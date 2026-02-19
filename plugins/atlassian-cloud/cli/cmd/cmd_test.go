package cmd

import (
	"testing"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func TestResolveSite(t *testing.T) {
	tests := []struct {
		name     string
		global   string
		refSite  string
		expected string
	}{
		{
			name:     "empty global uses refSite",
			global:   "",
			refSite:  "site-a",
			expected: "site-a",
		},
		{
			name:     "global overrides refSite",
			global:   "override",
			refSite:  "site-a",
			expected: "override",
		},
		{
			name:     "both empty returns empty",
			global:   "",
			refSite:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev := siteName
			defer func() { siteName = prev }()

			siteName = tt.global
			got := resolveSite(tt.refSite)
			if got != tt.expected {
				t.Errorf("resolveSite(%q) with siteName=%q: got %q, want %q",
					tt.refSite, tt.global, got, tt.expected)
			}
		})
	}
}

func TestEscapeCQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "double quotes are escaped",
			input:    `foo"bar`,
			expected: `foo\"bar`,
		},
		{
			name:     "backslashes are escaped",
			input:    `foo\bar`,
			expected: `foo\\bar`,
		},
		{
			name:     "both backslash and quote are escaped",
			input:    `foo\"bar`,
			expected: `foo\\\"bar`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeCQL(tt.input)
			if got != tt.expected {
				t.Errorf("escapeCQL(%q): got %q, want %q",
					tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractAttachments(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected []*models.IssueAttachmentScheme
	}{
		{
			name:     "valid attachments",
			data:     `{"fields":{"attachment":[{"id":"1","filename":"doc.pdf","size":1024,"mimeType":"application/pdf","content":"https://example.com/doc.pdf"}]}}`,
			expected: []*models.IssueAttachmentScheme{{ID: "1", Filename: "doc.pdf", Size: 1024, MimeType: "application/pdf", Content: "https://example.com/doc.pdf"}},
		},
		{
			name:     "no attachments field",
			data:     `{"fields":{"summary":"test"}}`,
			expected: nil,
		},
		{
			name:     "empty attachments",
			data:     `{"fields":{"attachment":[]}}`,
			expected: []*models.IssueAttachmentScheme{},
		},
		{
			name:     "invalid JSON",
			data:     `not json`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractAttachments([]byte(tt.data))
			if len(got) != len(tt.expected) {
				t.Fatalf("ExtractAttachments: got %d attachments, want %d", len(got), len(tt.expected))
			}
			for i := range got {
				if got[i].Filename != tt.expected[i].Filename {
					t.Errorf("attachment[%d].Filename = %q, want %q", i, got[i].Filename, tt.expected[i].Filename)
				}
				if got[i].Size != tt.expected[i].Size {
					t.Errorf("attachment[%d].Size = %d, want %d", i, got[i].Size, tt.expected[i].Size)
				}
			}
		})
	}
}
