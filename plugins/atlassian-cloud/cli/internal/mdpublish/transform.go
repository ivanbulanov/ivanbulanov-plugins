package mdpublish

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// TOCMarker is the paragraph the table-of-contents macro replaces once the
// document has been converted to storage format.
const TOCMarker = "@@TOC@@"

// Placeholder is a block replaced by an attachment after the page exists.
type Placeholder struct {
	Key    string
	Kind   string
	Source string
}

// Transformed is a document ready to be converted, with everything Confluence
// cannot take expressed as a placeholder instead.
type Transformed struct {
	Markdown     string
	Title        string
	Placeholders []Placeholder
	Hoisted      int
	Unwrapped    []string
}

var (
	htmlCommentRe = regexp.MustCompile(`(?s)<!--.*?-->`)
	indentedTable = regexp.MustCompile(`(?m)^[ \t]+\|`)
	schemeRe      = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*:`)
)

// Transform rewrites the source into Markdown that Confluence's converter can
// take, and returns what was set aside.
func Transform(src []byte, cfg Config) (Transformed, error) {
	out := Transformed{}
	var edits []Edit

	doc := Parser().Parser().Parse(text.NewReader(src))

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			if node.Level == 1 && out.Title == "" {
				lines := node.Lines()
				if lines.Len() > 0 {
					titleSeg := lines.At(0)
					out.Title = string(titleSeg.Value(src))
					start, end := blockRange(node, src)
					edits = append(edits, Edit{Start: start, End: end, Replacement: ""})
				}
			}
		case *ast.FencedCodeBlock:
			if string(node.Language(src)) != "mermaid" {
				return ast.WalkContinue, nil
			}
			var body strings.Builder
			for i := 0; i < node.Lines().Len(); i++ {
				lineSeg := node.Lines().At(i)
				body.Write(lineSeg.Value(src))
			}
			key := fmt.Sprintf("@@ph%d@@", len(out.Placeholders))
			out.Placeholders = append(out.Placeholders, Placeholder{
				Key: key, Kind: "mermaid", Source: body.String(),
			})
			start, end := fenceRange(node, src)
			edits = append(edits, Edit{Start: start, End: end, Replacement: key})
		case *ast.Image:
			dest := string(node.Destination)
			if schemeRe.MatchString(dest) {
				return ast.WalkContinue, nil
			}
			key := fmt.Sprintf("@@ph%d@@", len(out.Placeholders))
			out.Placeholders = append(out.Placeholders, Placeholder{
				Key: key, Kind: "image", Source: dest,
			})
			start, end := inlineRange(node, src)
			edits = append(edits, Edit{Start: start, End: end, Replacement: key})
		case *ast.Link:
			dest := string(node.Destination)
			if schemeRe.MatchString(dest) || strings.HasPrefix(dest, "#") {
				return ast.WalkContinue, nil
			}
			out.Unwrapped = append(out.Unwrapped, dest)
			start, end := inlineRange(node, src)
			edits = append(edits, Edit{Start: start, End: end, Replacement: linkText(node, src)})
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return out, fmt.Errorf("walk document: %w", err)
	}

	// HTML comments are stripped wholesale: Confluence renders them as
	// visible text, and the configuration comments have already been read.
	// The comment that marked the TOC position is replaced with the TOC
	// marker in place, rather than dropped along with the rest.
	for _, loc := range htmlCommentRe.FindAllIndex(src, -1) {
		replacement := ""
		if tocCommentRe.Match(src[loc[0]:loc[1]]) && !cfg.SuppressTOC {
			replacement = TOCMarker
		}
		edits = append(edits, Edit{Start: loc[0], End: loc[1], Replacement: replacement})
	}

	merged, err := ApplyEdits(src, dropContained(edits))
	if err != nil {
		return out, err
	}

	merged, out.Hoisted = hoistIndentedTables(merged)
	merged = placeTOC(merged, !cfg.SuppressTOC)
	out.Markdown = merged

	return out, nil
}

// hoistIndentedTables de-indents any table indented under a list item. ADF
// cannot nest a table in a list item; de-indenting splits the list, and an
// explicitly numbered list resumes its numbering afterwards.
func hoistIndentedTables(md string) (string, int) {
	lines := strings.Split(md, "\n")
	var out []string
	hoisted := 0

	for i := 0; i < len(lines); i++ {
		if !indentedTable.MatchString(lines[i]) {
			out = append(out, lines[i])
			continue
		}
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
			out = append(out, strings.TrimSpace(lines[i]))
			i++
		}
		if i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			out = append(out, "")
		}
		i--
		hoisted++
	}

	return strings.Join(out, "\n"), hoisted
}

// placeTOC ensures the document carries the table-of-contents marker when one
// is wanted: in the spot the document chose, otherwise at the top.
func placeTOC(md string, want bool) string {
	// The document may already carry the marker, substituted in place where a
	// confluence-toc comment stood. Checking for it keeps this idempotent, so
	// an author-chosen position is never duplicated by a second one at the top.
	if !want || strings.Contains(md, TOCMarker) {
		return md
	}
	return TOCMarker + "\n\n" + strings.TrimLeft(md, "\n")
}

func blockRange(n ast.Node, src []byte) (int, int) {
	lines := n.Lines()
	start := lines.At(0).Start
	end := lines.At(lines.Len() - 1).Stop
	// Take the newline that ends the block, so no blank line is left behind.
	for end < len(src) && src[end] == '\n' {
		end++
	}
	// Take the line's leading marker, e.g. "# ".
	for start > 0 && src[start-1] != '\n' {
		start--
	}
	return start, end
}

func fenceRange(n *ast.FencedCodeBlock, src []byte) (int, int) {
	// goldmark's Lines() span only the fence body, so the opening and closing
	// fence lines have to be taken explicitly. Leaving the opening line behind
	// is not a harmless cosmetic bug: "```mermaid" is not a valid closing
	// fence, so the leftover opener starts a code block that swallows every
	// heading until the next bare fence.
	start := 0
	switch {
	case n.Info != nil:
		start = n.Info.Segment.Start
	case n.Lines().Len() > 0:
		start = n.Lines().At(0).Start
	}
	for start > 0 && src[start-1] != '\n' {
		start--
	}

	// Walk whole lines from the opening fence until the closing fence line.
	// Counting lines rather than trusting the body's end offset keeps this
	// correct whether or not a line segment includes its newline, and handles
	// a fence with no body at all. The range stops just before the closing
	// fence's newline: that newline is half of the blank line separating the
	// block from what follows, and eating it would glue the replacement
	// placeholder onto the next paragraph instead of leaving it standing
	// alone, which in turn stops it converting to its own <p>.
	end := start
	opened := false
	for end < len(src) {
		lineStart := end
		lineEnd := lineStart
		for lineEnd < len(src) && src[lineEnd] != '\n' {
			lineEnd++
		}
		line := strings.TrimSpace(string(src[lineStart:lineEnd]))
		isFence := strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~")
		if opened && isFence {
			return start, lineEnd
		}
		opened = opened || isFence
		end = lineEnd
		if end < len(src) {
			end++
		}
	}
	return start, end
}

func inlineRange(n ast.Node, src []byte) (int, int) {
	start, end := -1, -1
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := c.(*ast.Text); ok {
			if start < 0 || t.Segment.Start < start {
				start = t.Segment.Start
			}
			if t.Segment.Stop > end {
				end = t.Segment.Stop
			}
		}
		return ast.WalkContinue, nil
	})
	// Widen to the surrounding link or image syntax.
	for start > 0 && src[start-1] != '[' {
		start--
	}
	start--
	if start > 0 && src[start-1] == '!' {
		start--
	}
	depth := 0
	for end < len(src) {
		if src[end] == '(' {
			depth++
		}
		if src[end] == ')' {
			depth--
			if depth == 0 {
				end++
				break
			}
		}
		end++
	}
	return start, end
}

func linkText(n ast.Node, src []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if t, ok := c.(*ast.Text); ok {
				b.Write(t.Segment.Value(src))
			}
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// dropContained removes edits fully inside another edit, which happens when a
// comment sits inside a block that is being removed anyway.
func dropContained(edits []Edit) []Edit {
	var out []Edit
	for i, e := range edits {
		contained := false
		for j, other := range edits {
			if i != j && other.Start <= e.Start && other.End >= e.End && !(other.Start == e.Start && other.End == e.End) {
				contained = true
				break
			}
		}
		if !contained {
			out = append(out, e)
		}
	}
	return out
}
