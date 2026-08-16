package mdpublish

import (
	"context"
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
		svg  string
		want int
	}{
		"mmdc shape": {
			`<svg id="my-svg" width="100%" xmlns="http://www.w3.org/2000/svg" ` +
				`class="flowchart" style="max-width: 109.797px; background-color: white;" ` +
				`viewBox="0 0 109.797 174">`,
			110,
		},
		"viewbox only": {
			`<svg width="100%" viewBox="0 0 640.5 200">`,
			641,
		},
		"negative origin viewbox": {
			`<svg width="100%" viewBox="-2 -2 729 400">`,
			729,
		},
		"plain pixel width": {
			`<svg width="480" height="200">`,
			480,
		},
		"no width at all": {
			`<svg xmlns="http://www.w3.org/2000/svg">`,
			0,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := svgWidth(tc.svg); got != tc.want {
				t.Errorf("svgWidth() = %d, want %d", got, tc.want)
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

	first, width, err := RenderMermaid(context.Background(), src, dir, "flow")
	if err != nil {
		t.Fatalf("RenderMermaid: %v", err)
	}
	if width <= 0 {
		t.Errorf("width = %d, want the SVG's intrinsic width", width)
	}
	if !strings.HasSuffix(first, ".svg") {
		t.Errorf("path = %q", first)
	}

	second, _, err := RenderMermaid(context.Background(), src, dir, "flow")
	if err != nil {
		t.Fatalf("second RenderMermaid: %v", err)
	}
	if first != second {
		t.Errorf("same source produced two names: %q and %q", first, second)
	}

	changed, _, err := RenderMermaid(context.Background(), src+"\n    B->>A: bye\n", dir, "flow")
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
