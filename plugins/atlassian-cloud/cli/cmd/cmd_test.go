package cmd

import "testing"

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
