// Package mdpublish converts a Markdown design document into a Confluence
// Cloud page body, resolving cross-references to real heading anchors.
package mdpublish

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Heading is one heading of a Confluence page, with the anchor id its
// renderer will generate for it.
type Heading struct {
	Level int
	Text  string
	ID    string
}

// adfNode is the subset of an ADF node this package reads.
type adfNode struct {
	Type    string    `json:"type"`
	Text    string    `json:"text"`
	Attrs   adfAttrs  `json:"attrs"`
	Content []adfNode `json:"content"`
}

type adfAttrs struct {
	Level     int    `json:"level"`
	Text      string `json:"text"`
	ShortName string `json:"shortName"`
}

// whitespace matches what JavaScript's /\s/ matches, which is what
// Confluence's renderer replaces with hyphens.
var whitespace = regexp.MustCompile("[\\t\\n\\v\\f\\r \\x{00A0}\\x{1680}\\x{2000}-\\x{200A}\\x{2028}\\x{2029}\\x{202F}\\x{205F}\\x{3000}\\x{FEFF}]")

func parseADF(adf []byte) (adfNode, error) {
	var doc adfNode
	if err := json.Unmarshal(adf, &doc); err != nil {
		return adfNode{}, fmt.Errorf("parse ADF: %w", err)
	}
	return doc, nil
}

// nodeText is the port of getText from @atlaskit/renderer. It reads one node
// only; the caller joins a heading's direct children.
func nodeText(n adfNode) string {
	if n.Text != "" {
		return n.Text
	}
	if n.Attrs.Text != "" {
		return n.Attrs.Text
	}
	if n.Attrs.ShortName != "" {
		return n.Attrs.ShortName
	}
	return "[" + n.Type + "]"
}

// Headings returns every heading of an ADF document with the anchor id
// Confluence's renderer generates for it. This is a port of
// ReactSerializer.getHeadingId from @atlaskit/renderer; see the spec for why
// the ids reported by the REST API cannot be used instead.
func Headings(adf []byte) ([]Heading, error) {
	doc, err := parseADF(adf)
	if err != nil {
		return nil, err
	}

	var out []Heading
	var seen []string

	var unique func(base string) string
	unique = func(base string) string {
		if !contains(seen, base) {
			seen = append(seen, base)
			return base
		}
		for n := 1; ; n++ {
			candidate := fmt.Sprintf("%s.%d", base, n)
			if !contains(seen, candidate) {
				seen = append(seen, candidate)
				return candidate
			}
		}
	}

	var walk func(n adfNode)
	walk = func(n adfNode) {
		if n.Type == "heading" {
			var raw strings.Builder
			for _, child := range n.Content {
				raw.WriteString(nodeText(child))
			}
			base := whitespace.ReplaceAllString(strings.TrimSpace(raw.String()), "-")
			if base != "" {
				out = append(out, Heading{
					Level: n.Attrs.Level,
					Text:  strings.TrimSpace(raw.String()),
					ID:    unique(base),
				})
			}
		}
		for _, child := range n.Content {
			walk(child)
		}
	}
	walk(doc)

	return out, nil
}

// HeadingInsideExpand reports whether any heading sits inside an expand.
// Confluence numbers those in a separate namespace (its own source calls this
// a known defect, ED-9668), so their anchors are not derivable and a document
// containing one cannot be published.
func HeadingInsideExpand(adf []byte) (bool, error) {
	doc, err := parseADF(adf)
	if err != nil {
		return false, err
	}

	var walk func(n adfNode, inExpand bool) bool
	walk = func(n adfNode, inExpand bool) bool {
		if n.Type == "heading" && inExpand {
			return true
		}
		if n.Type == "expand" || n.Type == "nestedExpand" {
			inExpand = true
		}
		for _, child := range n.Content {
			if walk(child, inExpand) {
				return true
			}
		}
		return false
	}

	return walk(doc, false), nil
}

// Text returns every text node of an ADF document concatenated in document
// order. Two documents with equal Text carry the same words in the same
// places, which is what the integrity checks compare.
func Text(adf []byte) (string, error) {
	doc, err := parseADF(adf)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	var walk func(n adfNode)
	walk = func(n adfNode) {
		b.WriteString(n.Text)
		for _, child := range n.Content {
			walk(child)
		}
	}
	walk(doc)

	return b.String(), nil
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
