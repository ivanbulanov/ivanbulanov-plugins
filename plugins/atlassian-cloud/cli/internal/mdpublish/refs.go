package mdpublish

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// RefResult is the outcome of rewriting cross-references into links.
type RefResult struct {
	Markdown   string
	Linked     int
	Unresolved []string
	Ambiguous  []string
}

// Fragment percent-encodes an anchor id for use in a URL. This is
// encodeURIComponent plus parentheses: the link is written in Markdown, where
// an unencoded ')' ends the destination.
func Fragment(anchorID string) string {
	const unreserved = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.~"
	var b strings.Builder
	for _, octet := range []byte(anchorID) {
		if strings.IndexByte(unreserved, octet) >= 0 {
			b.WriteByte(octet)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", octet)
	}
	return b.String()
}

// LinkReferences rewrites every cross-reference token into an inline link.
// It walks the AST and edits only text nodes that are not inside a heading,
// code span, code block or existing link, so a token in code is never touched.
func LinkReferences(md string, headings []Heading, patterns []Pattern, pageURL string) (RefResult, error) {
	src := []byte(md)
	result := RefResult{}

	byPrefix := map[string][]Heading{}
	for _, h := range headings {
		byPrefix[h.Text] = append(byPrefix[h.Text], h)
	}

	unresolved := map[string]bool{}
	ambiguous := map[string]bool{}
	var edits []Edit

	doc := Parser().Parser().Parse(text.NewReader(src))
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		case *ast.Heading, *ast.CodeSpan, *ast.FencedCodeBlock, *ast.CodeBlock, *ast.Link, *ast.AutoLink, *ast.Image:
			// Everything inside these is off limits.
			return ast.WalkSkipChildren, nil
		}

		node, ok := n.(*ast.Text)
		if !ok {
			return ast.WalkContinue, nil
		}

		segment := node.Segment
		raw := string(segment.Value(src))
		for _, p := range patterns {
			for _, loc := range p.Token.FindAllStringSubmatchIndex(raw, -1) {
				token := raw[loc[0]:loc[1]]
				capture := raw[loc[2]:loc[3]]
				prefix := strings.ReplaceAll(p.HeadingPrefix, "{1}", capture)

				matches := matchingHeadings(headings, prefix)
				if len(matches) == 0 {
					unresolved[token] = true
					continue
				}
				if len(byPrefix[matches[0].Text]) > 1 {
					ambiguous[token] = true
					continue
				}

				edits = append(edits, Edit{
					Start:       segment.Start + loc[0],
					End:         segment.Start + loc[1],
					Replacement: fmt.Sprintf("[%s](%s#%s)", token, pageURL, Fragment(matches[0].ID)),
				})
				result.Linked++
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return result, fmt.Errorf("walk document: %w", err)
	}

	out, err := ApplyEdits(src, edits)
	if err != nil {
		return result, err
	}
	result.Markdown = out
	result.Unresolved = sortedKeys(unresolved)
	result.Ambiguous = sortedKeys(ambiguous)

	return result, nil
}

// matchingHeadings returns headings whose text begins with prefix, preferring
// the shallowest so that "5." names the section rather than a subsection.
func matchingHeadings(headings []Heading, prefix string) []Heading {
	var out []Heading
	for _, h := range headings {
		if strings.HasPrefix(h.Text, prefix) {
			out = append(out, h)
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
