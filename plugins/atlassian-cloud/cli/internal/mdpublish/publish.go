package mdpublish

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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
	return currentText == newText, nil
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
	return conv.StorageToADF(ctx, storage, "")
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
	for i, ph := range transformed.Placeholders {
		switch ph.Kind {
		case "mermaid":
			path, width, err := RenderMermaid(ctx, ph.Source, opts.AssetsDir, DiagramSlug(i, ""))
			if err != nil {
				return result, nil, err
			}
			filenames[ph.Key] = filepath.Base(path)
			widths[ph.Key] = width
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
	storage = SpliceTOC(storage)
	storage = RestoreHardBreaks(storage)
	storage, err = SpliceImages(storage, filenames, widths)
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
