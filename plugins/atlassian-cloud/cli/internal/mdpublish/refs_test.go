package mdpublish

import (
	"strings"
	"testing"
)

var testHeadings = []Heading{
	{Level: 2, Text: "5. API", ID: "5.-API"},
	{Level: 2, Text: "7. Retry budget (max 20, backoff 3)", ID: "7.-Retry-budget-(max-20,-backoff-3)"},
	{Level: 3, Text: "D6 — Bypass by feature flag", ID: "D6-—-Bypass-by-feature-flag"},
}

const testPage = "https://example.atlassian.net/wiki/spaces/DOCS/pages/123/T"

func TestFragmentEncoding(t *testing.T) {
	tests := map[string]string{
		"5.-API":                              "5.-API",
		"7.-Retry-budget-(max-20,-backoff-3)": "7.-Retry-budget-%28max-20%2C-backoff-3%29",
		"D6-—-Bypass":                         "D6-%E2%80%94-Bypass",
		"POST-/items/bulk/refresh":            "POST-%2Fitems%2Fbulk%2Frefresh",
	}
	for in, want := range tests {
		if got := Fragment(in); got != want {
			t.Errorf("Fragment(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLinkReferences(t *testing.T) {
	got, err := LinkReferences("the gate in §5 and D6 apply\n", testHeadings, DefaultPatterns(), testPage)
	if err != nil {
		t.Fatalf("LinkReferences: %v", err)
	}
	if got.Linked != 2 {
		t.Errorf("Linked = %d, want 2", got.Linked)
	}
	if !strings.Contains(got.Markdown, "[§5]("+testPage+"#5.-API)") {
		t.Errorf("section link missing: %q", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "[D6]("+testPage+"#D6-%E2%80%94-Bypass-by-feature-flag)") {
		t.Errorf("decision link missing: %q", got.Markdown)
	}
}

func TestLinkReferencesLinksEveryOccurrence(t *testing.T) {
	got, err := LinkReferences("§5 then §5 then §5\n", testHeadings, DefaultPatterns(), testPage)
	if err != nil {
		t.Fatalf("LinkReferences: %v", err)
	}
	if got.Linked != 3 {
		t.Errorf("Linked = %d, want 3", got.Linked)
	}
}

func TestLinkReferencesSkipsCodeAndHeadings(t *testing.T) {
	src := "## 5. API\n\nprose §5 here\n\n`see §5`\n\n```\nliteral §5\n```\n"
	got, err := LinkReferences(src, testHeadings, DefaultPatterns(), testPage)
	if err != nil {
		t.Fatalf("LinkReferences: %v", err)
	}
	if got.Linked != 1 {
		t.Errorf("Linked = %d, want 1 — only the prose occurrence", got.Linked)
	}
	if strings.Contains(got.Markdown, "## [§5]") || strings.Contains(got.Markdown, "`see [§5]") {
		t.Errorf("rewrote a protected region: %q", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "literal §5") {
		t.Errorf("fenced code changed: %q", got.Markdown)
	}
}

func TestLinkReferencesSkipsExistingLinks(t *testing.T) {
	got, err := LinkReferences("[§5](https://elsewhere.example/x)\n", testHeadings, DefaultPatterns(), testPage)
	if err != nil {
		t.Fatalf("LinkReferences: %v", err)
	}
	if got.Linked != 0 {
		t.Errorf("Linked = %d, want 0", got.Linked)
	}
}

func TestLinkReferencesReportsUnresolved(t *testing.T) {
	got, err := LinkReferences("see §99 which does not exist\n", testHeadings, DefaultPatterns(), testPage)
	if err != nil {
		t.Fatalf("LinkReferences: %v", err)
	}
	if len(got.Unresolved) != 1 || got.Unresolved[0] != "§99" {
		t.Errorf("Unresolved = %v", got.Unresolved)
	}
}

func TestLinkReferencesReportsAmbiguous(t *testing.T) {
	dupes := append([]Heading{}, testHeadings...)
	dupes = append(dupes, Heading{Level: 2, Text: "5. API", ID: "5.-API.1"})
	got, err := LinkReferences("see §5\n", dupes, DefaultPatterns(), testPage)
	if err != nil {
		t.Fatalf("LinkReferences: %v", err)
	}
	if len(got.Ambiguous) != 1 || got.Ambiguous[0] != "§5" {
		t.Errorf("Ambiguous = %v — a duplicated target must not be linked silently", got.Ambiguous)
	}
}
