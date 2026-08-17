package mdpublish

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const versionMarkerPrefix = "published by atlassian-cloud confluence publish; source "

// PageState is what the target page looks like before we touch it.
type PageState struct {
	ID             string
	Title          string
	Version        int
	VersionMessage string
	ADF            []byte
}

// Options are the inputs of one publish run.
type Options struct {
	SourcePath string
	PageURL    string
	PageID     string
	Title      string
	AssetsDir  string
	LinkRefs   bool
	DryRun     bool
	Force      bool
	// NoTOC turns off the table of contents. It is emitted by default, at the
	// top unless the document chose a spot with a <!-- confluence-toc -->
	// comment; HasTOCMarker selects the position, never whether there is one.
	NoTOC bool
}

// Result is what a run produced, for the run summary.
type Result struct {
	Storage   string
	Linked    int
	Hoisted   int
	Unwrapped []string
	// Assets are local file paths to upload as attachments before the write.
	Assets []string
}

// VersionMarker is the version message this tool writes, carrying a hash of
// the source so a later run can tell which revision produced the page.
func VersionMarker(sourceHash string) string {
	return versionMarkerPrefix + sourceHash
}

// IsOurVersion reports whether a page version was written by this tool. An API
// token publishes as the user, so the message is the only signal.
func IsOurVersion(message string) bool {
	return strings.HasPrefix(message, versionMarkerPrefix)
}

// ShouldSkip reports whether the page already says what we would publish.
func ShouldSkip(current PageState, newADF []byte) (bool, error) {
	if len(current.ADF) == 0 {
		return false, nil
	}
	currentText, err := Text(current.ADF)
	if err != nil {
		return false, err
	}
	newText, err := Text(newADF)
	if err != nil {
		return false, err
	}
	if currentText != newText {
		return false, nil
	}
	// Text alone cannot see a formatting change. Code-block wrapping, diagram
	// dimensions and breakout width all leave the text byte-identical, so a
	// text-only comparison reports "no change" and a rendering fix can never
	// reach the page. Compare the structure the tool controls as well.
	currentShape, err := StructuralFingerprint(current.ADF)
	if err != nil {
		return false, err
	}
	newShape, err := StructuralFingerprint(newADF)
	if err != nil {
		return false, err
	}
	return currentShape == newShape, nil
}

// fingerprintAttrs are the attributes the publish pipeline controls. This is an
// allowlist rather than a denylist on purpose: Confluence stamps generated
// identifiers into its own output (macroId, localId, media ids), and a field it
// starts generating tomorrow must not make every run look like a change.
var fingerprintAttrs = map[string]bool{
	"level": true, "language": true, "wrap": true, "mode": true,
	"width": true, "height": true, "layout": true, "widthType": true,
	"alt": true, "href": true, "extensionKey": true,
	"order": true, "panelType": true, "url": true,
}

// converterOnlyAttrs are added when storage is converted to a document but are
// absent from the same document read back off the page. Including any of them
// would make every run differ from the page it just wrote, so the no-op guard
// would never skip. "alt" carries the attachment's identity on both sides and
// is content-addressed, so a changed diagram still registers as a change.
var converterOnlyAttrs = map[string]bool{
	"__fileName": true, "__fileSize": true, "__fileMimeType": true,
}

// StructuralFingerprint reduces a document to the shape this tool is
// responsible for: node types in document order, each with its marks and the
// attributes above. It deliberately ignores text, which ShouldSkip compares
// separately, and every identifier Confluence generates for itself.
func StructuralFingerprint(adf []byte) (string, error) {
	var doc any
	if err := json.Unmarshal(adf, &doc); err != nil {
		return "", fmt.Errorf("parse document: %w", err)
	}
	var b strings.Builder
	writeShape(&b, doc)
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:]), nil
}

func writeShape(b *strings.Builder, node any) {
	switch n := node.(type) {
	case []any:
		for _, child := range n {
			writeShape(b, child)
		}
	case map[string]any:
		kind, named := n["type"].(string)
		if named {
			b.WriteString("<")
			b.WriteString(kind)
			writeShapeAttrs(b, n["attrs"])
			if marks, ok := n["marks"].([]any); ok {
				for _, m := range marks {
					mark, ok := m.(map[string]any)
					if !ok {
						continue
					}
					if markType, ok := mark["type"].(string); ok {
						b.WriteString("@")
						b.WriteString(markType)
						writeShapeAttrs(b, mark["attrs"])
					}
				}
			}
		}
		writeShape(b, n["content"])
		if named {
			b.WriteString(">")
		}
	}
}

func writeShapeAttrs(b *strings.Builder, attrs any) {
	m, ok := attrs.(map[string]any)
	if !ok {
		return
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if fingerprintAttrs[k] && !converterOnlyAttrs[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(b, " %s=%v", k, m[k])
	}
}

// SourceHash identifies the revision of the document being published.
func SourceHash(src []byte) string {
	sum := sha256.Sum256(src)
	return hex.EncodeToString(sum[:])[:12]
}

// Converter is the subset of the Confluence client this package needs, so
// that orchestration can be exercised without the network.
type Converter interface {
	MarkdownToStorage(ctx context.Context, markdown string) (string, error)
	StorageToADF(ctx context.Context, storage, pageIDContext string) ([]byte, error)
}

// markdownToADF converts Markdown to ADF in the two hops the API requires.
// This is where the anchor map comes from: the ids are derived from what
// Confluence itself produced, not guessed from the Markdown.
func markdownToADF(ctx context.Context, conv Converter, markdown string) ([]byte, error) {
	storage, err := conv.MarkdownToStorage(ctx, markdown)
	if err != nil {
		return nil, err
	}
	// Collapsed here as well as in Prepare: the preflight check demands the
	// two ADFs be byte-identical once placeholders are removed, so both sides
	// have to see the same whitespace.
	return conv.StorageToADF(ctx, CollapseSoftBreaks(storage), "")
}

// Prepare runs every step up to the point of writing: reject, transform, learn
// the anchors, link, render, splice, and check. It returns the storage to
// publish, in Result.Storage, and the ADF of that storage.
func Prepare(ctx context.Context, conv Converter, opts Options) (Result, []byte, error) {
	var result Result

	src, err := os.ReadFile(opts.SourcePath)
	if err != nil {
		return result, nil, fmt.Errorf("read source: %w", err)
	}

	cfg, problems, err := Scan(src)
	if err != nil {
		return result, nil, err
	}
	if len(problems) > 0 {
		var b strings.Builder
		b.WriteString("the document contains constructs Confluence cannot represent:\n")
		for _, p := range problems {
			fmt.Fprintf(&b, "  %s:%d: %s: %s\n", opts.SourcePath, p.Line, p.Kind, p.Detail)
		}
		return result, nil, fmt.Errorf("%s", b.String())
	}
	cfg.SuppressTOC = opts.NoTOC

	transformed, err := Transform(src, cfg)
	if err != nil {
		return result, nil, err
	}
	result.Hoisted = transformed.Hoisted
	result.Unwrapped = transformed.Unwrapped

	beforeADF, err := markdownToADF(ctx, conv, transformed.Markdown)
	if err != nil {
		return result, nil, err
	}

	inExpand, err := HeadingInsideExpand(beforeADF)
	if err != nil {
		return result, nil, err
	}
	if inExpand {
		return result, nil, fmt.Errorf(
			"a heading sits inside an expand; Confluence numbers those separately and its anchor cannot be derived")
	}

	headings, err := Headings(beforeADF)
	if err != nil {
		return result, nil, err
	}

	body := transformed.Markdown
	if opts.LinkRefs {
		refs, err := LinkReferences(body, headings, cfg.Patterns, opts.PageURL)
		if err != nil {
			return result, nil, err
		}
		if len(refs.Unresolved) > 0 {
			return result, nil, fmt.Errorf("cross-references resolve to no heading: %s",
				strings.Join(refs.Unresolved, ", "))
		}
		if len(refs.Ambiguous) > 0 {
			return result, nil, fmt.Errorf("cross-references name a heading whose text is duplicated: %s",
				strings.Join(refs.Ambiguous, ", "))
		}
		body = refs.Markdown
		result.Linked = refs.Linked
	}

	filenames := map[string]string{}
	widths := map[string]int{}
	heights := map[string]int{}
	for i, ph := range transformed.Placeholders {
		switch ph.Kind {
		case "mermaid":
			path, width, height, err := RenderMermaid(ctx, ph.Source, opts.AssetsDir, DiagramSlug(i, ""))
			if err != nil {
				return result, nil, err
			}
			filenames[ph.Key] = filepath.Base(path)
			widths[ph.Key] = width
			heights[ph.Key] = height
			result.Assets = append(result.Assets, path)
		case "image":
			path := ph.Source
			if !filepath.IsAbs(path) {
				path = filepath.Join(filepath.Dir(opts.SourcePath), path)
			}
			filenames[ph.Key] = filepath.Base(path)
			result.Assets = append(result.Assets, path)
		}
	}

	storage, err := conv.MarkdownToStorage(ctx, body)
	if err != nil {
		return result, nil, err
	}
	// Before any splicing, so the macros spliced in below keep the line breaks
	// that make the generated storage readable.
	storage = CollapseSoftBreaks(storage)
	storage = SpliceTOC(storage)
	storage = RestoreHardBreaks(storage)
	storage = StyleCodeBlocks(storage)
	storage, err = SpliceImages(storage, filenames, widths, heights)
	if err != nil {
		return result, nil, err
	}

	afterADF, err := conv.StorageToADF(ctx, storage, opts.PageID)
	if err != nil {
		return result, nil, err
	}

	pre, err := CheckPreflight(beforeADF, afterADF, transformed.Placeholders)
	if err != nil {
		return result, nil, err
	}
	if !pre.OK() {
		return result, nil, fmt.Errorf(
			"preflight failed; nothing was written.\n  unsupported: %v\n  heading drift: %v\n  text: %s",
			pre.Unsupported, pre.HeadingDrift, pre.TextDiff)
	}

	result.Storage = storage
	return result, afterADF, nil
}
