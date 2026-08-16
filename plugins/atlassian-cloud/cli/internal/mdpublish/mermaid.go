package mdpublish

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// mermaidConfig pins the diagram margins. The spec measured two pixels of
// slack in mmdc's own output, so margins are set here rather than trimmed
// from the SVG afterwards, which would need text metrics and a browser.
const mermaidConfig = `{"theme":"default","sequence":{"diagramMarginX":8,"diagramMarginY":8},` +
	`"flowchart":{"diagramPadding":8}}`

var (
	slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)
	// mmdc emits width="100%" and carries the real size in the style's
	// max-width, so that is read first; viewBox is the fallback.
	svgMaxWidthRe = regexp.MustCompile(`max-width:\s*([0-9.]+)px`)
	svgViewBoxRe  = regexp.MustCompile(`viewBox="\s*[-0-9.]+\s+[-0-9.]+\s+([0-9.]+)\s+[0-9.]+\s*"`)
	svgWidthRe    = regexp.MustCompile(`<svg[^>]*\bwidth="([0-9.]+)(?:px)?"`)
)

// MermaidPath locates mmdc. MMDC overrides the lookup so a pinned binary can be
// used without putting it on PATH. It is never installed automatically: the
// caller is told what to run and the publish stops.
func MermaidPath() (string, error) {
	if pinned := os.Getenv("MMDC"); pinned != "" {
		if _, err := os.Stat(pinned); err != nil {
			return "", fmt.Errorf("MMDC is set to %q, which is not usable: %w", pinned, err)
		}
		return pinned, nil
	}
	path, err := exec.LookPath("mmdc")
	if err != nil {
		return "", fmt.Errorf("mmdc not found on PATH; the document has Mermaid diagrams. " +
			"Install it with: npm install -g @mermaid-js/mermaid-cli")
	}
	return path, nil
}

// DiagramSlug names a diagram after the section it appears in, so a human can
// match a file to a place in the document.
func DiagramSlug(index int, heading string) string {
	slug := slugUnsafe.ReplaceAllString(strings.ToLower(heading), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return fmt.Sprintf("diagram-%d", index)
	}
	return slug
}

// RenderMermaid renders one diagram to outDir and returns its path and
// intrinsic width. The filename carries a hash of the source, so an unchanged
// diagram keeps its name and is neither re-rendered nor re-uploaded.
func RenderMermaid(ctx context.Context, source, outDir, slug string) (string, int, error) {
	sum := sha256.Sum256([]byte(source))
	name := fmt.Sprintf("%s-%s.svg", slug, hex.EncodeToString(sum[:])[:12])
	outPath := filepath.Join(outDir, name)

	if data, err := os.ReadFile(outPath); err == nil {
		return outPath, svgWidth(string(data)), nil
	}

	mmdc, err := MermaidPath()
	if err != nil {
		return "", 0, err
	}

	inPath := filepath.Join(outDir, name+".mmd")
	if err := os.WriteFile(inPath, []byte(source), 0o600); err != nil {
		return "", 0, fmt.Errorf("write diagram source: %w", err)
	}
	defer os.Remove(inPath)

	cfgPath := filepath.Join(outDir, ".mermaid-config.json")
	if err := os.WriteFile(cfgPath, []byte(mermaidConfig), 0o600); err != nil {
		return "", 0, fmt.Errorf("write mermaid config: %w", err)
	}
	defer os.Remove(cfgPath)

	cmd := exec.CommandContext(ctx, mmdc, "-i", inPath, "-o", outPath, "-c", cfgPath, "-b", "transparent")
	cmd.Env = os.Environ()
	// Reuse a system Chrome so a clean machine does not download one.
	if chrome := systemChrome(); chrome != "" {
		cmd.Env = append(cmd.Env, "PUPPETEER_EXECUTABLE_PATH="+chrome)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return "", 0, fmt.Errorf("mmdc failed: %w\n%s", err, out)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		return "", 0, fmt.Errorf("read rendered diagram: %w", err)
	}

	return outPath, svgWidth(string(data)), nil
}

func systemChrome() string {
	for _, candidate := range []string{"google-chrome-stable", "google-chrome", "chromium", "chromium-browser"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}

// svgWidth reads the diagram's intrinsic width in pixels. mmdc writes
// width="100%" on the root and puts the real number in style="max-width:Npx",
// so a naive width attribute read returns nothing; viewBox is the last resort.
func svgWidth(svg string) int {
	for _, re := range []*regexp.Regexp{svgMaxWidthRe, svgViewBoxRe, svgWidthRe} {
		m := re.FindStringSubmatch(svg)
		if m == nil {
			continue
		}
		width, err := strconv.ParseFloat(m[1], 64)
		if err != nil || width <= 0 {
			continue
		}
		return int(width + 0.5)
	}
	return 0
}
