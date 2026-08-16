package mdpublish

import "testing"

func TestScanReadsConfiguration(t *testing.T) {
	src := []byte("<!-- confluence-page: https://x.atlassian.net/wiki/spaces/DOCS/pages/123/T -->\n" +
		"<!-- confluence-toc -->\n\n# Title\n\ntext\n")
	cfg, problems, err := Scan(src)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems = %v, want none", problems)
	}
	if cfg.PageURL != "https://x.atlassian.net/wiki/spaces/DOCS/pages/123/T" {
		t.Errorf("PageURL = %q", cfg.PageURL)
	}
	if !cfg.HasTOCMarker {
		t.Error("HasTOCMarker = false, want true")
	}
}

func TestScanRejectsUnsupportedConstructs(t *testing.T) {
	tests := []struct {
		name string
		src  string
		kind string
	}{
		{"raw html block", "text\n\n<div>x</div>\n", "raw-html"},
		{"raw inline html", "a <span>x</span> b\n", "raw-html"},
		{"footnote reference", "text[^1]\n\n[^1]: note\n", "footnote"},
		{"nested blockquote", "> outer\n>\n> > inner\n", "nested-blockquote"},
		{"definition list", "Term\n: definition\n", "definition-list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, problems, err := Scan([]byte(tt.src))
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if len(problems) == 0 {
				t.Fatalf("want a %s problem, got none", tt.kind)
			}
			if problems[0].Kind != tt.kind {
				t.Errorf("Kind = %q, want %q", problems[0].Kind, tt.kind)
			}
			if problems[0].Line == 0 {
				t.Error("Line = 0, want a real line number")
			}
		})
	}
}

func TestScanIgnoresConstructsInsideCode(t *testing.T) {
	// The first consumer's Mermaid fences contain <br/>. A line-based scan
	// would reject the very document this tool exists to publish.
	src := []byte("```mermaid\nA->>B: one<br/>two\n```\n\nprose with `<span>` in code\n")
	_, problems, err := Scan(src)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(problems) != 0 {
		t.Errorf("problems = %v, want none — fenced and inline code are not HTML", problems)
	}
}

func TestDefaultPatternsMatchTokens(t *testing.T) {
	patterns := DefaultPatterns()
	if len(patterns) != 2 {
		t.Fatalf("got %d patterns, want 2", len(patterns))
	}
	if !patterns[0].Token.MatchString("see §5 now") {
		t.Error("section pattern should match §5")
	}
	if !patterns[1].Token.MatchString("per D6 above") {
		t.Error("decision pattern should match D6")
	}
	if patterns[1].Token.MatchString("D66") {
		t.Error("decision pattern must not match D66")
	}
}
