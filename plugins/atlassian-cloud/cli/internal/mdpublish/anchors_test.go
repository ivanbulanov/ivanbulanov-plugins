package mdpublish

import (
	"encoding/json"
	"testing"
)

func adfDoc(headings ...string) []byte {
	// each argument is a heading's plain text
	out := `{"type":"doc","version":1,"content":[`
	for i, h := range headings {
		if i > 0 {
			out += ","
		}
		// json.Marshal escapes control characters and quotes so the
		// heading text can never corrupt the surrounding JSON structure.
		text, _ := json.Marshal(h)
		out += `{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":` + string(text) + `}]}`
	}
	return []byte(out + `]}`)
}

func TestHeadings(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"plain", "5. API", "5.-API"},
		{"em dash and comma", "D1 — Cache the result, not the request", "D1-—-Cache-the-result,-not-the-request"},
		{"slashes kept", "POST /items/bulk/refresh", "POST-/items/bulk/refresh"},
		{"parentheses kept", "7. Retry budget (max 20, backoff 3)", "7.-Retry-budget-(max-20,-backoff-3)"},
		{"leading and trailing space trimmed", "  Spaced  ", "Spaced"},
		{"double space becomes two hyphens", "A  B", "A--B"},
		{"tab is whitespace", "A\tB", "A-B"},
		{"no case folding", "What Has To Be Built", "What-Has-To-Be-Built"},
		{"not truncated", "very long heading very long heading very long heading very long heading",
			"very-long-heading-very-long-heading-very-long-heading-very-long-heading"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Headings(adfDoc(tt.text))
			if err != nil {
				t.Fatalf("Headings: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("got %d headings, want 1", len(got))
			}
			if got[0].ID != tt.want {
				t.Errorf("ID = %q, want %q", got[0].ID, tt.want)
			}
		})
	}
}

func TestHeadingsDuplicates(t *testing.T) {
	got, err := Headings(adfDoc("Same", "Other", "Same", "Same"))
	if err != nil {
		t.Fatalf("Headings: %v", err)
	}
	want := []string{"Same", "Other", "Same.1", "Same.2"}
	for i, w := range want {
		if got[i].ID != w {
			t.Errorf("heading %d: ID = %q, want %q", i, got[i].ID, w)
		}
	}
}

func TestHeadingsInlineNodes(t *testing.T) {
	// inline code contributes its text; emoji contributes shortName; a date contributes "[date]"
	adf := []byte(`{"type":"doc","version":1,"content":[{"type":"heading","attrs":{"level":3},` +
		`"content":[{"type":"text","text":"D4 — Reuse ","marks":[]},` +
		`{"type":"text","text":"widget_tokens","marks":[{"type":"code"}]},` +
		`{"type":"text","text":" now "},` +
		`{"type":"emoji","attrs":{"shortName":":rocket:"}},` +
		`{"type":"date","attrs":{"timestamp":"1"}}]}]}`)
	got, err := Headings(adf)
	if err != nil {
		t.Fatalf("Headings: %v", err)
	}
	want := "D4-—-Reuse-widget_tokens-now-:rocket:[date]"
	if got[0].ID != want {
		t.Errorf("ID = %q, want %q", got[0].ID, want)
	}
}

func TestHeadingsEmptyGetsNoID(t *testing.T) {
	adf := []byte(`{"type":"doc","version":1,"content":[{"type":"heading","attrs":{"level":2},"content":[]}]}`)
	got, err := Headings(adf)
	if err != nil {
		t.Fatalf("Headings: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %d headings, want 0 — an empty heading has no id", len(got))
	}
}

func TestHeadingInsideExpand(t *testing.T) {
	adf := []byte(`{"type":"doc","version":1,"content":[{"type":"expand","attrs":{},"content":[` +
		`{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"Hidden"}]}]}]}`)
	found, err := HeadingInsideExpand(adf)
	if err != nil {
		t.Fatalf("HeadingInsideExpand: %v", err)
	}
	if !found {
		t.Error("want true — headings inside expands use a separate id namespace")
	}
}

func TestText(t *testing.T) {
	got, err := Text(adfDoc("One", "Two"))
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	if got != "OneTwo" {
		t.Errorf("Text = %q, want %q", got, "OneTwo")
	}
}

func TestHeadingsGoldenCases(t *testing.T) {
	// One case per punctuation class the anchor rule has to survive: em dash,
	// double quotes, semicolon, slashes, a colon path parameter, parentheses
	// with a comma, and a two-digit section number. Conformance to Atlassian's
	// live renderer is what the integration test checks, by comparing every
	// generated id against the ids in the published page's DOM; these pairs
	// guard the Go port against regressing on its own.
	pairs := [][2]string{
		{"1. Summary", "1.-Summary"},
		{"What has to be built", "What-has-to-be-built"},
		{"D1 — Cache the result, not the request", "D1-—-Cache-the-result,-not-the-request"},
		{`D2 — "Draft", not "pending", in code and data`, `D2-—-"Draft",-not-"pending",-in-code-and-data`},
		{"D4 — Reuse widget_tokens rather than a new store", "D4-—-Reuse-widget_tokens-rather-than-a-new-store"},
		{"D8 — State is keyed by id; the log stays per-run", "D8-—-State-is-keyed-by-id;-the-log-stays-per-run"},
		{"5. API", "5.-API"},
		{"POST /items/bulk/refresh", "POST-/items/bulk/refresh"},
		{"POST /items/bulk-fetch (v2)", "POST-/items/bulk-fetch-(v2)"},
		{"PUT /admin/items/:itemId/tags (v2)", "PUT-/admin/items/:itemId/tags-(v2)"},
		{"7. Retry budget (max 20, backoff 3)", "7.-Retry-budget-(max-20,-backoff-3)"},
		{"10. Provider subscription state (v3)", "10.-Provider-subscription-state-(v3)"},
	}
	for _, p := range pairs {
		t.Run(p[0], func(t *testing.T) {
			got, err := Headings(adfDoc(p[0]))
			if err != nil {
				t.Fatalf("Headings: %v", err)
			}
			if got[0].ID != p[1] {
				t.Errorf("ID = %q, want %q", got[0].ID, p[1])
			}
		})
	}
}
