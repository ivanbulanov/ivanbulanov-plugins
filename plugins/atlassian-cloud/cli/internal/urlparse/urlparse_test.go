package urlparse

import "testing"

func TestParseJiraURL(t *testing.T) {
	tests := []struct {
		input   string
		wantKey string
		wantSite string
		wantOk  bool
	}{
		{"DEV-123", "DEV-123", "", true},
		{"PROJ-1", "PROJ-1", "", true},
		{"https://acme-corp.atlassian.net/browse/ACME-5136", "ACME-5136", "acme-corp.atlassian.net", true},
		{"https://company.atlassian.net/browse/PROJ-456?focusedCommentId=123", "PROJ-456", "company.atlassian.net", true},
		{"not-a-ticket", "", "", false},
		{"", "", "", false},
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
		})
	}
}

func TestParseConfluenceURL(t *testing.T) {
	tests := []struct {
		input    string
		wantID   string
		wantSite string
		wantOk   bool
	}{
		{"https://acme-corp.atlassian.net/wiki/spaces/ENG/pages/1234567890/API+Guidelines", "1234567890", "acme-corp.atlassian.net", true},
		{"https://co.atlassian.net/wiki/spaces/DEV/pages/123456", "123456", "co.atlassian.net", true},
		{"1234567890", "1234567890", "", true},
		{"not-a-page", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, ok := ParseConfluenceRef(tt.input)
			if ok != tt.wantOk {
				t.Fatalf("ParseConfluenceRef(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if result.PageID != tt.wantID {
				t.Errorf("PageID = %q, want %q", result.PageID, tt.wantID)
			}
			if result.Site != tt.wantSite {
				t.Errorf("Site = %q, want %q", result.Site, tt.wantSite)
			}
		})
	}
}
