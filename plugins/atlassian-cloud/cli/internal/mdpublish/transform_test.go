package mdpublish

import (
	"strings"
	"testing"
)

func transform(t *testing.T, src string) Transformed {
	t.Helper()
	cfg, problems, err := Scan([]byte(src))
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	out, err := Transform([]byte(src), cfg)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	return out
}

func TestTransformStripsH1AndKeepsTitle(t *testing.T) {
	got := transform(t, "# The Title\n\nbody\n")
	if got.Title != "The Title" {
		t.Errorf("Title = %q", got.Title)
	}
	if strings.Contains(got.Markdown, "# The Title") {
		t.Errorf("H1 still present: %q", got.Markdown)
	}
}

func TestTransformStripsEveryHTMLComment(t *testing.T) {
	// Confluence renders an HTML comment as visible text, so all of them go.
	got := transform(t, "<!-- confluence-toc -->\n\nbefore\n\n<!-- a note -->\n\nafter\n")
	if strings.Contains(got.Markdown, "<!--") {
		t.Errorf("comment survived: %q", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "@@TOC@@") {
		t.Errorf("TOC marker missing: %q", got.Markdown)
	}
}

func TestTransformAddsTOCAtTopWhenUnmarked(t *testing.T) {
	got := transform(t, "# T\n\nfirst paragraph\n")
	if !strings.HasPrefix(strings.TrimSpace(got.Markdown), "@@TOC@@") {
		t.Errorf("want TOC marker first, got %q", got.Markdown)
	}
}

// A marked document must get exactly one marker, in the author's spot — not a
// second one prepended at the top.
func TestTransformDoesNotDuplicateAMarkedTOC(t *testing.T) {
	got := transform(t, "intro\n\n<!-- confluence-toc -->\n\nrest\n")
	if n := strings.Count(got.Markdown, TOCMarker); n != 1 {
		t.Errorf("got %d TOC markers, want 1: %q", n, got.Markdown)
	}
	if strings.HasPrefix(strings.TrimSpace(got.Markdown), TOCMarker) {
		t.Errorf("marker was hoisted to the top, ignoring the chosen spot: %q", got.Markdown)
	}
}

func TestTransformSuppressTOC(t *testing.T) {
	// Both shapes must end up with no marker at all: the silent document that
	// would otherwise get one at the top, and the one that asked for a spot.
	for name, src := range map[string]string{
		"unmarked": "# T\n\nfirst paragraph\n",
		"marked":   "intro\n\n<!-- confluence-toc -->\n\nrest\n",
	} {
		t.Run(name, func(t *testing.T) {
			cfg, problems, err := Scan([]byte(src))
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if len(problems) != 0 {
				t.Fatalf("problems: %v", problems)
			}
			cfg.SuppressTOC = true
			got, err := Transform([]byte(src), cfg)
			if err != nil {
				t.Fatalf("Transform: %v", err)
			}
			if strings.Contains(got.Markdown, TOCMarker) {
				t.Errorf("TOC marker present despite SuppressTOC: %q", got.Markdown)
			}
			if strings.Contains(got.Markdown, "<!--") {
				t.Errorf("comment survived: %q", got.Markdown)
			}
		})
	}
}

func TestTransformReplacesMermaidFences(t *testing.T) {
	got := transform(t, "text\n\n```mermaid\nsequenceDiagram\n  A->>B: hi<br/>there\n```\n\nmore\n")
	if len(got.Placeholders) != 1 {
		t.Fatalf("got %d placeholders, want 1", len(got.Placeholders))
	}
	if got.Placeholders[0].Kind != "mermaid" {
		t.Errorf("Kind = %q", got.Placeholders[0].Kind)
	}
	if !strings.Contains(got.Placeholders[0].Source, "sequenceDiagram") {
		t.Errorf("Source = %q", got.Placeholders[0].Source)
	}
	if !strings.Contains(got.Markdown, got.Placeholders[0].Key) {
		t.Errorf("placeholder key missing from output")
	}
	if strings.Contains(got.Markdown, "sequenceDiagram") {
		t.Errorf("fence body survived into the markdown")
	}
	// The fence markers must go too. A leftover "```mermaid" is not a valid
	// closing fence, so it opens a code block that swallows the rest of the
	// document — three whole sections vanished from a real page this way.
	if strings.Contains(got.Markdown, "```") {
		t.Errorf("fence markers survived into the markdown: %q", got.Markdown)
	}
}

// Two adjacent diagrams are the case that broke a real publish: the leftover
// opener of the first fence ran on until the *second* fence line, so every
// heading in between was absorbed into a code block.
func TestTransformReplacesConsecutiveMermaidFences(t *testing.T) {
	src := "```mermaid\ngraph TD\n  A-->B\n```\n\n## 6. Middle\n\n```mermaid\ngraph TD\n  C-->D\n```\n\n## 7. After\n"
	got := transform(t, src)

	if len(got.Placeholders) != 2 {
		t.Fatalf("got %d placeholders, want 2", len(got.Placeholders))
	}
	if strings.Contains(got.Markdown, "```") {
		t.Errorf("fence markers survived: %q", got.Markdown)
	}
	for _, heading := range []string{"## 6. Middle", "## 7. After"} {
		if !strings.Contains(got.Markdown, heading) {
			t.Errorf("heading %q was swallowed: %q", heading, got.Markdown)
		}
	}
	// Each placeholder must stand alone as a paragraph, or it will not convert
	// to its own <p> and the image splice will not find it.
	for _, ph := range got.Placeholders {
		if !strings.Contains(got.Markdown, "\n\n"+ph.Key+"\n\n") {
			t.Errorf("placeholder %s is not its own paragraph: %q", ph.Key, got.Markdown)
		}
	}
}

func TestTransformReplacesEmptyMermaidFence(t *testing.T) {
	// An empty fence has no body lines at all; reading line zero of an empty
	// span panics.
	got := transform(t, "before\n\n```mermaid\n```\n\nafter\n")
	if len(got.Placeholders) != 1 {
		t.Fatalf("got %d placeholders, want 1", len(got.Placeholders))
	}
	if strings.Contains(got.Markdown, "```") {
		t.Errorf("fence markers survived: %q", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "before") || !strings.Contains(got.Markdown, "after") {
		t.Errorf("surrounding text lost: %q", got.Markdown)
	}
}

func TestTransformReplacesLocalImages(t *testing.T) {
	got := transform(t, "![a diagram](./flow.svg)\n")
	if len(got.Placeholders) != 1 || got.Placeholders[0].Kind != "image" {
		t.Fatalf("placeholders = %+v", got.Placeholders)
	}
	if got.Placeholders[0].Source != "./flow.svg" {
		t.Errorf("Source = %q", got.Placeholders[0].Source)
	}
}

func TestTransformLeavesRemoteImages(t *testing.T) {
	got := transform(t, "![a](https://example.com/x.png)\n")
	if len(got.Placeholders) != 0 {
		t.Errorf("remote image should not become a placeholder")
	}
}

func TestTransformHoistsNestedTables(t *testing.T) {
	src := "1. first\n2. second:\n\n   | A | B |\n   |---|---|\n   | 1 | 2 |\n\n   tail of item two\n3. third\n"
	got := transform(t, src)
	if got.Hoisted != 1 {
		t.Errorf("Hoisted = %d, want 1", got.Hoisted)
	}
	for _, line := range strings.Split(got.Markdown, "\n") {
		if strings.HasPrefix(line, "   |") || strings.HasPrefix(line, "\t|") {
			t.Errorf("table still indented: %q", line)
		}
	}
	if !strings.Contains(got.Markdown, "3. third") {
		t.Errorf("numbering lost: %q", got.Markdown)
	}
}

func TestTransformUnwrapsLocalLinks(t *testing.T) {
	got := transform(t, "see [adjacent-notes.md](./adjacent-notes.md) for detail\n")
	if !strings.Contains(got.Markdown, "see adjacent-notes.md for detail") {
		t.Errorf("got %q", got.Markdown)
	}
	if len(got.Unwrapped) != 1 || got.Unwrapped[0] != "./adjacent-notes.md" {
		t.Errorf("Unwrapped = %v", got.Unwrapped)
	}
}

func TestTransformKeepsAbsoluteAndFragmentLinks(t *testing.T) {
	got := transform(t, "[a](https://example.com) and [b](#anchor)\n")
	if !strings.Contains(got.Markdown, "[a](https://example.com)") {
		t.Errorf("absolute link changed: %q", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "[b](#anchor)") {
		t.Errorf("fragment link changed: %q", got.Markdown)
	}
	if len(got.Unwrapped) != 0 {
		t.Errorf("Unwrapped = %v, want none", got.Unwrapped)
	}
}
