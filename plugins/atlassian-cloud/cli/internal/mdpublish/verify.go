package mdpublish

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const unsupportedKey = "__confluenceADFMigrationUnsupportedContentInternalExtension__"

// Preflight is what the checks found before anything was written.
type Preflight struct {
	Unsupported  []string
	HeadingDrift []string
	TextDiff     string
}

// OK reports whether the document is safe to publish.
func (p Preflight) OK() bool {
	return len(p.Unsupported) == 0 && len(p.HeadingDrift) == 0 && p.TextDiff == ""
}

// CheckPreflight compares the document as converted from Markdown against the
// document as it will be stored. Linking cannot change text — a link is a mark
// and the link text is the token itself — so once placeholders are removed the
// two must be byte-identical. Any difference at all is corruption.
func CheckPreflight(before, after []byte, placeholders []Placeholder) (Preflight, error) {
	var p Preflight

	if strings.Contains(string(after), unsupportedKey) {
		p.Unsupported = append(p.Unsupported, unsupportedKey)
	}

	beforeHeadings, err := Headings(before)
	if err != nil {
		return p, err
	}
	afterHeadings, err := Headings(after)
	if err != nil {
		return p, err
	}
	beforeIDs := map[string]bool{}
	for _, h := range beforeHeadings {
		beforeIDs[h.ID] = true
	}
	for _, h := range afterHeadings {
		if !beforeIDs[h.ID] {
			p.HeadingDrift = append(p.HeadingDrift, h.ID)
		}
	}

	beforeText, err := Text(before)
	if err != nil {
		return p, err
	}
	afterText, err := Text(after)
	if err != nil {
		return p, err
	}
	// Splicing swaps each marker for a node that carries no text: the images
	// for <ac:image>, the table-of-contents marker for the toc macro. Both
	// have to come out of the comparison, or the check reports a difference
	// the splice was supposed to make.
	beforeText = strings.ReplaceAll(beforeText, TOCMarker, "")
	for _, ph := range placeholders {
		beforeText = strings.ReplaceAll(beforeText, ph.Key, "")
	}
	if beforeText != afterText {
		p.TextDiff = firstDifference(beforeText, afterText)
	}

	return p, nil
}

// CheckPublished walks the page that actually landed and reports how many
// same-page links it carries and which of their fragments match no heading.
func CheckPublished(adf []byte, pageID string) (int, []string, error) {
	headings, err := Headings(adf)
	if err != nil {
		return 0, nil, err
	}
	ids := map[string]bool{}
	for _, h := range headings {
		ids[h.ID] = true
	}

	links := 0
	var broken []string
	for _, href := range hrefs(adf) {
		if !strings.Contains(href, pageID) {
			continue
		}
		hash := strings.Index(href, "#")
		if hash < 0 {
			continue
		}
		links++
		fragment, err := url.QueryUnescape(href[hash+1:])
		if err != nil {
			fragment = href[hash+1:]
		}
		if !ids[fragment] {
			broken = append(broken, fragment)
		}
	}

	return links, broken, nil
}

// hrefs collects every link mark's href in document order.
func hrefs(adf []byte) []string {
	var doc map[string]any
	if err := json.Unmarshal(adf, &doc); err != nil {
		return nil
	}

	var out []string
	var walk func(v any)
	walk = func(v any) {
		switch node := v.(type) {
		case map[string]any:
			if marks, ok := node["marks"].([]any); ok {
				for _, m := range marks {
					mark, ok := m.(map[string]any)
					if !ok || mark["type"] != "link" {
						continue
					}
					attrs, ok := mark["attrs"].(map[string]any)
					if !ok {
						continue
					}
					if href, ok := attrs["href"].(string); ok {
						out = append(out, href)
					}
				}
			}
			for _, child := range node {
				walk(child)
			}
		case []any:
			for _, child := range node {
				walk(child)
			}
		}
	}
	walk(doc)

	return out
}

func firstDifference(a, b string) string {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			return fmt.Sprintf("at byte %d: expected %q, got %q",
				i, window(a, i), window(b, i))
		}
	}
	return fmt.Sprintf("length differs: %d expected, %d produced", len(a), len(b))
}

func window(s string, i int) string {
	start := i - 30
	if start < 0 {
		start = 0
	}
	end := i + 30
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}
