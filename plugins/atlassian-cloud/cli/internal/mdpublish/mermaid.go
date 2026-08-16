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

// svgFormatVersion must be bumped whenever RenderMermaid's post-processing of
// the rendered SVG changes (for example setSVGIntrinsicSize). It is folded
// into the cache filename below so that a change here invalidates both the
// on-disk cache and, via the filename becoming new, the page's existing
// attachment: without it, unchanged Mermaid source would keep resolving to
// the same filename and the old, broken SVG would never be re-uploaded.
const svgFormatVersion = "2"

var (
	slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)
	// mmdc emits width="100%" and carries the real size in the style's
	// max-width, so that is read first; viewBox is the fallback.
	svgMaxWidthRe = regexp.MustCompile(`max-width:\s*([0-9.]+)px`)
	svgViewBoxRe  = regexp.MustCompile(`viewBox="\s*[-0-9.]+\s+[-0-9.]+\s+([0-9.]+)\s+[0-9.]+\s*"`)
	svgWidthRe    = regexp.MustCompile(`<svg[^>]*\bwidth="([0-9.]+)(?:px)?"`)
	// mmdc's SVG carries no max-height style equivalent, so height is read
	// from the viewBox's fourth number, falling back to a plain attribute.
	svgViewBoxHeightRe = regexp.MustCompile(`viewBox="\s*[-0-9.]+\s+[-0-9.]+\s+[0-9.]+\s+([0-9.]+)\s*"`)
	svgHeightRe        = regexp.MustCompile(`<svg[^>]*\bheight="([0-9.]+)(?:px)?"`)
	// Strip any existing width/height on the root <svg> tag before
	// setSVGIntrinsicSize writes fresh pixel values, regardless of whether
	// the old value was numeric or a percentage.
	svgRootWidthAttrRe  = regexp.MustCompile(`\s+width="[^"]*"`)
	svgRootHeightAttrRe = regexp.MustCompile(`\s+height="[^"]*"`)
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

// diagramFilename returns the content-addressed name RenderMermaid writes and
// looks for on disk and among a page's attachments. svgFormatVersion is
// folded into the hash so that changing how the SVG is post-processed (not
// just changing the Mermaid source) yields a new name, forcing a fresh
// render and a fresh upload instead of reusing a stale cached file or an
// already-attached image with the old, broken markup.
func diagramFilename(source, slug string) string {
	sum := sha256.Sum256([]byte(source + "\x00" + svgFormatVersion))
	return fmt.Sprintf("%s-%s.svg", slug, hex.EncodeToString(sum[:])[:12])
}

// RenderMermaid renders one diagram to outDir and returns its path and
// intrinsic width and height. The filename carries a hash of the source and
// svgFormatVersion, so an unchanged diagram rendered by an unchanged
// post-processing step keeps its name and is neither re-rendered nor
// re-uploaded.
func RenderMermaid(ctx context.Context, source, outDir, slug string) (path string, width, height int, err error) {
	name := diagramFilename(source, slug)
	outPath := filepath.Join(outDir, name)

	if data, err := os.ReadFile(outPath); err == nil {
		svg := string(data)
		return outPath, svgWidth(svg), svgHeight(svg), nil
	}

	mmdc, err := MermaidPath()
	if err != nil {
		return "", 0, 0, err
	}

	inPath := filepath.Join(outDir, name+".mmd")
	if err := os.WriteFile(inPath, []byte(source), 0o600); err != nil {
		return "", 0, 0, fmt.Errorf("write diagram source: %w", err)
	}
	defer os.Remove(inPath)

	cfgPath := filepath.Join(outDir, ".mermaid-config.json")
	if err := os.WriteFile(cfgPath, []byte(mermaidConfig), 0o600); err != nil {
		return "", 0, 0, fmt.Errorf("write mermaid config: %w", err)
	}
	defer os.Remove(cfgPath)

	cmd := exec.CommandContext(ctx, mmdc, "-i", inPath, "-o", outPath, "-c", cfgPath, "-b", "transparent")
	cmd.Env = os.Environ()
	// Reuse a system Chrome so a clean machine does not download one.
	if chrome := systemChrome(); chrome != "" {
		cmd.Env = append(cmd.Env, "PUPPETEER_EXECUTABLE_PATH="+chrome)
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		return "", 0, 0, fmt.Errorf("mmdc failed: %w\n%s", err, out)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("read rendered diagram: %w", err)
	}

	processed := setSVGIntrinsicSize(string(data))
	if err := os.WriteFile(outPath, []byte(processed), 0o600); err != nil {
		return "", 0, 0, fmt.Errorf("write processed diagram: %w", err)
	}

	return outPath, svgWidth(processed), svgHeight(processed), nil
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

// svgHeight reads the diagram's intrinsic height in pixels. Unlike width,
// mmdc's root carries no style-based max-height, so the viewBox's fourth
// number is read first, falling back to a plain height attribute.
func svgHeight(svg string) int {
	for _, re := range []*regexp.Regexp{svgViewBoxHeightRe, svgHeightRe} {
		m := re.FindStringSubmatch(svg)
		if m == nil {
			continue
		}
		height, err := strconv.ParseFloat(m[1], 64)
		if err != nil || height <= 0 {
			continue
		}
		return int(height + 0.5)
	}
	return 0
}

// setSVGIntrinsicSize rewrites the SVG root's width and height to explicit
// pixel integers so Confluence can size its placeholder without downloading
// and decoding the attachment first. mmdc's root has width="100%" and no
// height at all; on the live page that left one diagram with no recorded
// size and another recorded as 1x1, so Confluence reserved boxes with no
// relation to the rendered image. Only the first <svg ...> tag is touched;
// style and viewBox are left exactly as mmdc wrote them.
func setSVGIntrinsicSize(svg string) string {
	width := svgWidth(svg)
	height := svgHeight(svg)
	if width <= 0 || height <= 0 {
		return svg
	}

	start := strings.Index(svg, "<svg")
	if start < 0 {
		return svg
	}
	relEnd := strings.IndexByte(svg[start:], '>')
	if relEnd < 0 {
		return svg
	}
	end := start + relEnd

	tag := svg[start:end]
	tag = svgRootWidthAttrRe.ReplaceAllString(tag, "")
	tag = svgRootHeightAttrRe.ReplaceAllString(tag, "")
	tag += fmt.Sprintf(` width="%d" height="%d"`, width, height)

	return svg[:start] + tag + svg[end:]
}
