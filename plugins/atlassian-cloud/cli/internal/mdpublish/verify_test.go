package mdpublish

import (
	"fmt"
	"strings"
	"testing"
)

func adfParagraphs(texts ...string) []byte {
	parts := make([]string, 0, len(texts))
	for _, s := range texts {
		parts = append(parts, fmt.Sprintf(`{"type":"paragraph","content":[{"type":"text","text":%q}]}`, s))
	}
	return []byte(`{"type":"doc","version":1,"content":[` + strings.Join(parts, ",") + `]}`)
}

func TestCheckPreflightPassesWhenTextMatches(t *testing.T) {
	before := adfParagraphs("hello", "@@ph0@@")
	after := adfParagraphs("hello")
	got, err := CheckPreflight(before, after, []Placeholder{{Key: "@@ph0@@", Kind: "mermaid"}})
	if err != nil {
		t.Fatalf("CheckPreflight: %v", err)
	}
	if !got.OK() {
		t.Errorf("want OK, got %+v", got)
	}
}

// SpliceTOC and SpliceImages both replace a marker with a node that carries no
// text, so neither may be reported as a text difference.
func TestCheckPreflightIgnoresSplicedMarkers(t *testing.T) {
	before := adfDoc(TOCMarker+"Intro", "@@ph0@@Body")
	after := adfDoc("Intro", "Body")

	got, err := CheckPreflight(before, after, []Placeholder{{Key: "@@ph0@@", Kind: "mermaid"}})
	if err != nil {
		t.Fatalf("CheckPreflight: %v", err)
	}
	if got.TextDiff != "" {
		t.Errorf("TextDiff = %q, want none", got.TextDiff)
	}
}

func TestCheckPreflightCatchesTextCorruption(t *testing.T) {
	// This is the failure the prototype shipped: a heading mangled by a
	// string rewrite, invisible to link checking.
	before := adfDoc("D4 — Reuse widget_tokens rather than a new store")
	after := adfDoc("D4 — Reuse �M120� rather than a new store")
	got, err := CheckPreflight(before, after, nil)
	if err != nil {
		t.Fatalf("CheckPreflight: %v", err)
	}
	if got.OK() {
		t.Fatal("want failure — the text changed")
	}
	if len(got.HeadingDrift) == 0 {
		t.Error("want the heading drift reported")
	}
	if got.TextDiff == "" {
		t.Error("want a text difference reported")
	}
}

func TestCheckPreflightCatchesUnsupportedContent(t *testing.T) {
	after := []byte(`{"type":"doc","version":1,"content":[{"type":"inlineExtension","attrs":` +
		`{"extensionKey":"__confluenceADFMigrationUnsupportedContentInternalExtension__"}}]}`)
	got, err := CheckPreflight(adfParagraphs(), after, nil)
	if err != nil {
		t.Fatalf("CheckPreflight: %v", err)
	}
	if len(got.Unsupported) == 0 {
		t.Error("want unsupported content reported")
	}
}

func TestCheckPublished(t *testing.T) {
	adf := []byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"heading","attrs":{"level":2},"content":[{"type":"text","text":"5. API"}]},` +
		`{"type":"paragraph","content":[{"type":"text","text":"a","marks":[{"type":"link","attrs":` +
		`{"href":"https://x.atlassian.net/wiki/spaces/DOCS/pages/123/T#5.-API"}}]},` +
		`{"type":"text","text":"b","marks":[{"type":"link","attrs":` +
		`{"href":"https://x.atlassian.net/wiki/spaces/DOCS/pages/123/T#Missing"}}]},` +
		`{"type":"text","text":"c","marks":[{"type":"link","attrs":` +
		`{"href":"https://example.com/other#Elsewhere"}}]}]}]}`)
	links, broken, err := CheckPublished(adf, "123")
	if err != nil {
		t.Fatalf("CheckPublished: %v", err)
	}
	if links != 2 {
		t.Errorf("links = %d, want 2 — the external link is not ours", links)
	}
	if len(broken) != 1 || broken[0] != "Missing" {
		t.Errorf("broken = %v", broken)
	}
}

func TestCheckPublishedDecodesFragments(t *testing.T) {
	adf := []byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"heading","attrs":{"level":3},"content":[{"type":"text","text":"D6 — Bypass"}]},` +
		`{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","attrs":` +
		`{"href":"https://x.atlassian.net/wiki/spaces/DOCS/pages/123/T#D6-%E2%80%94-Bypass"}}]}]}]}`)
	links, broken, err := CheckPublished(adf, "123")
	if err != nil {
		t.Fatalf("CheckPublished: %v", err)
	}
	if links != 1 || len(broken) != 0 {
		t.Errorf("links = %d, broken = %v — an encoded fragment must resolve", links, broken)
	}
}
