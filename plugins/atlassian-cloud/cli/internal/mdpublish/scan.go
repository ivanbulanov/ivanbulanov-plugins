package mdpublish

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Pattern maps a cross-reference token to the heading it names.
type Pattern struct {
	// Token matches one reference and must have exactly one capture group.
	Token *regexp.Regexp
	// HeadingPrefix is the prefix a heading's text must start with, where
	// {1} stands for the token's capture group.
	HeadingPrefix string
}

// Config is what the document says about how to publish it.
type Config struct {
	PageURL      string
	HasTOCMarker bool
	Patterns     []Pattern
}

// Problem is a construct Confluence cannot represent, with the line to fix.
type Problem struct {
	Line   int
	Kind   string
	Detail string
}

var (
	pageCommentRe = regexp.MustCompile(`(?m)^<!--\s*confluence-page:\s*(\S+)\s*-->\s*$`)
	tocCommentRe  = regexp.MustCompile(`(?m)^<!--\s*confluence-toc\s*-->\s*$`)
	footnoteRe    = regexp.MustCompile(`\[\^[^\]]+\]`)
	definitionRe  = regexp.MustCompile(`(?m)^:[ \t]+\S`)
)

// DefaultPatterns are the cross-reference patterns of the first consumer:
// §N names the heading beginning "N." and D1-D9 name the heading beginning
// with that identifier.
func DefaultPatterns() []Pattern {
	return []Pattern{
		{Token: regexp.MustCompile(`§(\d+)`), HeadingPrefix: "{1}."},
		{Token: regexp.MustCompile(`\bD([1-9])\b`), HeadingPrefix: "D{1}"},
	}
}

// Parser returns the goldmark parser this package uses everywhere, so that
// scanning, transforming and rewriting all see the same AST.
func Parser() goldmark.Markdown {
	// Footnote is enabled so a footnote reference parses into its own AST
	// node (extast.FootnoteLink) instead of being silently absorbed as a
	// generic link-reference-definition target, which would hide it from
	// Scan's rejection check.
	return goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote))
}

// Scan reads the document's configuration comments and reports every
// construct that Confluence would render as literal text or as uneditable
// legacy content. It reads the AST, never the lines: <br/> inside a Mermaid
// fence is diagram syntax, not HTML.
func Scan(src []byte) (Config, []Problem, error) {
	cfg := Config{Patterns: DefaultPatterns()}
	if m := pageCommentRe.FindSubmatch(src); m != nil {
		cfg.PageURL = string(m[1])
	}
	cfg.HasTOCMarker = tocCommentRe.Match(src)

	reader := text.NewReader(src)
	doc := Parser().Parser().Parse(reader)

	var problems []Problem
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.HTMLBlock:
			// A configuration comment is stripped later and is not a problem.
			raw := nodeSource(node, src)
			if !strings.HasPrefix(strings.TrimSpace(raw), "<!--") {
				problems = append(problems, Problem{
					Line:   lineOf(node, src),
					Kind:   "raw-html",
					Detail: firstLine(raw),
				})
			}
		case *ast.RawHTML:
			problems = append(problems, Problem{
				Line:   lineOf(node, src),
				Kind:   "raw-html",
				Detail: firstLine(nodeSource(node, src)),
			})
		case *ast.Blockquote:
			if _, nested := node.Parent().(*ast.Blockquote); nested {
				problems = append(problems, Problem{
					Line:   lineOf(node, src),
					Kind:   "nested-blockquote",
					Detail: "ADF cannot nest a blockquote inside a blockquote",
				})
			}
		case *extast.FootnoteLink:
			// With the Footnote extension enabled, a reference such as
			// [^1] that has a matching [^1]: definition parses into its
			// own node rather than leaving "[^1]" as literal text.
			problems = append(problems, Problem{
				Line:   lineOf(node, src),
				Kind:   "footnote",
				Detail: "footnote reference",
			})
		case *ast.Text:
			// An orphan reference such as [^1] with no matching
			// definition is not claimed by the footnote parser and
			// remains literal text, so it is still caught here.
			raw := string(node.Segment.Value(src))
			if footnoteRe.MatchString(raw) {
				problems = append(problems, Problem{
					Line:   lineOf(node, src),
					Kind:   "footnote",
					Detail: footnoteRe.FindString(raw),
				})
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return cfg, nil, fmt.Errorf("walk document: %w", err)
	}

	// A definition list is not a goldmark node; detect it on paragraph text.
	if loc := definitionRe.FindIndex(src); loc != nil {
		problems = append(problems, Problem{
			Line:   lineNumber(src, loc[0]),
			Kind:   "definition-list",
			Detail: "Confluence renders a definition list as literal text",
		})
	}

	return cfg, problems, nil
}

func nodeSource(n ast.Node, src []byte) string {
	switch node := n.(type) {
	case *ast.RawHTML:
		return string(node.Segments.Value(src))
	default:
		lines := n.Lines()
		if lines == nil || lines.Len() == 0 {
			return ""
		}
		return string(src[lines.At(0).Start:lines.At(lines.Len()-1).Stop])
	}
}

func lineOf(n ast.Node, src []byte) int {
	switch node := n.(type) {
	case *ast.RawHTML:
		if node.Segments.Len() > 0 {
			return lineNumber(src, node.Segments.At(0).Start)
		}
		return 0
	case *ast.Text:
		return lineNumber(src, node.Segment.Start)
	}

	// Calling Lines() on an inline node panics. A position-less inline
	// node (e.g. extast.FootnoteLink, which carries no source offset of
	// its own) borrows the line of its nearest block ancestor instead.
	if n.Type() != ast.TypeBlock {
		if offset, ok := nearestBlockOffset(n); ok {
			return lineNumber(src, offset)
		}
		return 0
	}

	if lines := n.Lines(); lines != nil && lines.Len() > 0 {
		return lineNumber(src, lines.At(0).Start)
	}
	// A container block (e.g. Blockquote) has no lines of its own; borrow
	// the first line of its first descendant that does.
	if offset, ok := firstLeafOffset(n.FirstChild()); ok {
		return lineNumber(src, offset)
	}
	return 0
}

// nearestBlockOffset climbs n's ancestors for the first block with a
// populated line, returning that line's start offset.
func nearestBlockOffset(n ast.Node) (int, bool) {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Type() != ast.TypeBlock {
			continue
		}
		if lines := p.Lines(); lines != nil && lines.Len() > 0 {
			return lines.At(0).Start, true
		}
	}
	return 0, false
}

// firstLeafOffset descends the first-child chain starting at n for the
// first node with a source position, returning its start offset.
func firstLeafOffset(n ast.Node) (int, bool) {
	for n != nil {
		switch node := n.(type) {
		case *ast.Text:
			return node.Segment.Start, true
		case *ast.RawHTML:
			if node.Segments.Len() > 0 {
				return node.Segments.At(0).Start, true
			}
			return 0, false
		}
		if n.Type() == ast.TypeBlock {
			if lines := n.Lines(); lines != nil && lines.Len() > 0 {
				return lines.At(0).Start, true
			}
		}
		n = n.FirstChild()
	}
	return 0, false
}

func lineNumber(src []byte, offset int) int {
	if offset > len(src) {
		offset = len(src)
	}
	return 1 + strings.Count(string(src[:offset]), "\n")
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
