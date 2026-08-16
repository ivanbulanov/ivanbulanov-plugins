package mdpublish

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagramSlug(t *testing.T) {
	tests := map[string]string{
		"6. Flows":                        "6-flows",
		"Change the address and validate": "change-the-address-and-validate",
		"":                                "diagram",
	}
	for in, want := range tests {
		if got := DiagramSlug(0, in); !strings.HasPrefix(got, want) {
			t.Errorf("DiagramSlug(%q) = %q, want prefix %q", in, got, want)
		}
	}
}

func TestMermaidPathReportsInstallCommand(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	// The integration run points MMDC at a pinned binary; without this the
	// override would satisfy the lookup and the test would never see an error.
	t.Setenv("MMDC", "")
	_, err := MermaidPath()
	if err == nil {
		t.Fatal("want an error when mmdc is absent")
	}
	if !strings.Contains(err.Error(), "@mermaid-js/mermaid-cli") {
		t.Errorf("error must name the install command, got: %v", err)
	}
}

func TestMermaidPathHonoursOverride(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	pinned := filepath.Join(t.TempDir(), "mmdc")
	if err := os.WriteFile(pinned, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MMDC", pinned)

	got, err := MermaidPath()
	if err != nil {
		t.Fatalf("MermaidPath: %v", err)
	}
	if got != pinned {
		t.Errorf("MermaidPath() = %q, want %q", got, pinned)
	}

	t.Setenv("MMDC", filepath.Join(t.TempDir(), "absent"))
	if _, err := MermaidPath(); err == nil {
		t.Error("want an error when MMDC points at a missing file")
	}
}

// svgWidth is the one place a real mmdc quirk bit: the root carries
// width="100%" and the pixel size lives in the style's max-width.
func TestSVGWidthReadsMermaidOutput(t *testing.T) {
	tests := map[string]struct {
		svg        string
		want       int
		wantHeight int
	}{
		"mmdc shape": {
			`<svg id="my-svg" width="100%" xmlns="http://www.w3.org/2000/svg" ` +
				`class="flowchart" style="max-width: 109.797px; background-color: white;" ` +
				`viewBox="0 0 109.797 174">`,
			110,
			174,
		},
		"viewbox only": {
			`<svg width="100%" viewBox="0 0 640.5 200">`,
			641,
			200,
		},
		"negative origin viewbox": {
			`<svg width="100%" viewBox="-2 -2 729 400">`,
			729,
			400,
		},
		"plain pixel width": {
			`<svg width="480" height="200">`,
			480,
			200,
		},
		"no width at all": {
			`<svg xmlns="http://www.w3.org/2000/svg">`,
			0,
			0,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := svgWidth(tc.svg); got != tc.want {
				t.Errorf("svgWidth() = %d, want %d", got, tc.want)
			}
			if got := svgHeight(tc.svg); got != tc.wantHeight {
				t.Errorf("svgHeight() = %d, want %d", got, tc.wantHeight)
			}
		})
	}
}

func TestRenderMermaidIsContentAddressed(t *testing.T) {
	if _, err := MermaidPath(); err != nil {
		t.Skip("mmdc not installed")
	}
	dir := t.TempDir()
	src := "sequenceDiagram\n    A->>B: hello\n"

	first, width, height, err := RenderMermaid(context.Background(), src, dir, "flow")
	if err != nil {
		t.Fatalf("RenderMermaid: %v", err)
	}
	if width <= 0 {
		t.Errorf("width = %d, want the SVG's intrinsic width", width)
	}
	if height <= 0 {
		t.Errorf("height = %d, want the SVG's intrinsic height", height)
	}
	if !strings.HasSuffix(first, ".svg") {
		t.Errorf("path = %q", first)
	}

	data, err := os.ReadFile(first)
	if err != nil {
		t.Fatalf("read rendered svg: %v", err)
	}
	if strings.Contains(string(data), `width="100%"`) {
		t.Error("the SVG written to disk must carry an explicit pixel width, not width=\"100%\"")
	}

	second, _, _, err := RenderMermaid(context.Background(), src, dir, "flow")
	if err != nil {
		t.Fatalf("second RenderMermaid: %v", err)
	}
	if first != second {
		t.Errorf("same source produced two names: %q and %q", first, second)
	}

	changed, _, _, err := RenderMermaid(context.Background(), src+"\n    B->>A: bye\n", dir, "flow")
	if err != nil {
		t.Fatalf("changed RenderMermaid: %v", err)
	}
	if changed == first {
		t.Error("changed source must produce a different filename")
	}

	entries, _ := os.ReadDir(dir)
	svgs := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".svg" {
			svgs++
		}
	}
	if svgs != 2 {
		t.Errorf("got %d svg files, want 2", svgs)
	}
}

// setSVGIntrinsicSize on a realistic mmdc root must produce explicit pixel
// dimensions while leaving the viewBox untouched.
func TestSetSVGIntrinsicSize(t *testing.T) {
	root := `<svg id="my-svg" width="100%" xmlns="http://www.w3.org/2000/svg" ` +
		`style="max-width: 1104px; background-color: white;" viewBox="-8 -8 1104 490.5">` +
		`<g><text>hi</text></g></svg>`

	got := setSVGIntrinsicSize(root)

	if strings.Contains(got, `width="100%"`) {
		t.Errorf("width=100%% must not survive post-processing: %s", got)
	}
	if !strings.Contains(got, `width="1104"`) {
		t.Errorf("want width=\"1104\", got: %s", got)
	}
	if !strings.Contains(got, `height="491"`) {
		t.Errorf("want height=\"491\" (490.5 rounded to nearest), got: %s", got)
	}
	if !strings.Contains(got, `viewBox="-8 -8 1104 490.5"`) {
		t.Errorf("viewBox must survive untouched, got: %s", got)
	}
	if !strings.Contains(got, `<text>hi</text>`) {
		t.Errorf("content after the root tag must be preserved, got: %s", got)
	}
}

// A root that already carries a numeric width and height must be rewritten
// consistently rather than gaining a second, duplicate attribute, and the
// result must be stable under a second application.
func TestSetSVGIntrinsicSizeDoesNotDuplicateAttributes(t *testing.T) {
	root := `<svg width="480" height="200" viewBox="0 0 480 200">` + "<g></g></svg>"

	once := setSVGIntrinsicSize(root)
	twice := setSVGIntrinsicSize(once)

	for _, got := range []string{once, twice} {
		rootTag := got[:strings.IndexByte(got, '>')]
		if n := strings.Count(rootTag, `width="`); n != 1 {
			t.Errorf("root tag has %d width= attributes, want 1: %s", n, rootTag)
		}
		if n := strings.Count(rootTag, `height="`); n != 1 {
			t.Errorf("root tag has %d height= attributes, want 1: %s", n, rootTag)
		}
		if !strings.Contains(rootTag, `width="480"`) || !strings.Contains(rootTag, `height="200"`) {
			t.Errorf("values must be preserved, got: %s", rootTag)
		}
	}
}

// The cache filename must change when svgFormatVersion changes, even though
// the Mermaid source did not: otherwise a fixed cache/attachment name would
// keep serving an SVG rendered with the old, broken post-processing.
func TestDiagramFilenameIncludesFormatVersion(t *testing.T) {
	source := "graph TD\n  A --> B\n"
	slug := "flow"

	got := diagramFilename(source, slug)

	bareSum := sha256.Sum256([]byte(source))
	bareName := fmt.Sprintf("%s-%s.svg", slug, hex.EncodeToString(bareSum[:])[:12])
	if got == bareName {
		t.Errorf("diagramFilename(%q) = %q, must differ from a hash of the bare source", source, got)
	}

	if again := diagramFilename(source, slug); again != got {
		t.Errorf("diagramFilename is not stable across calls: %q vs %q", got, again)
	}
}
