package mdpublish

import (
	"bytes"
	"fmt"
	"math"
	"strings"
)

// MaxImageWidth is the rendered width of a full-width diagram, matching what
// the existing published page uses.
const MaxImageWidth = 992

const tocMacro = `<ac:structured-macro ac:name="toc" ac:schema-version="1">` +
	`<ac:parameter ac:name="style">none</ac:parameter></ac:structured-macro>`

// SpliceTOC replaces the marker paragraph with the table-of-contents macro.
// Confluence escapes raw ac: tags in Markdown, so the macro can only be put
// in after conversion.
func SpliceTOC(storage string) string {
	return strings.ReplaceAll(storage, "<p>"+TOCMarker+"</p>", tocMacro)
}

// SpliceImages replaces each placeholder paragraph with an image referencing
// an attachment by filename. ac:width is the diagram's own width, capped, so
// a narrow diagram is not blown up to page width.
//
// ac:original-width and ac:original-height carry the unclamped intrinsic
// size. Measured against the live conversion API: ac:height is silently
// dropped by the converter no matter what else is present, while
// ac:original-width plus ac:original-height together do reach the ADF media
// node's width/height attrs. Without them, one diagram on the live page had
// no recorded size at all, another was recorded as 1x1, and Confluence
// padded its wrapper to 794px around what actually rendered as a 441px-tall
// image. Both are omitted together when either dimension is unknown, since a
// wrong intrinsic size is worse than none.
func SpliceImages(storage string, filenames map[string]string, widths, heights map[string]int) (string, error) {
	for key, filename := range filenames {
		marker := "<p>" + key + "</p>"
		if !strings.Contains(storage, marker) {
			return "", fmt.Errorf("placeholder %s is not in the converted storage", key)
		}
		intrinsicWidth := widths[key]
		width := intrinsicWidth
		if width == 0 || width > MaxImageWidth {
			width = MaxImageWidth
		}
		originalAttrs := ""
		if intrinsicHeight := heights[key]; intrinsicWidth > 0 && intrinsicHeight > 0 {
			originalAttrs = fmt.Sprintf(` ac:original-width="%d" ac:original-height="%d"`,
				intrinsicWidth, intrinsicHeight)
		}
		// layout="wide" is what the prototype verified on the live page: at 992px
		// a "center" layout overflows the default content column.
		image := fmt.Sprintf(`<ac:image ac:align="center" ac:layout="wide" ac:custom-width="true" `+
			`ac:width="%d"%s ac:alt="%s"><ri:attachment ri:filename="%s" /></ac:image>`,
			width, originalAttrs, filename, filename)
		storage = strings.ReplaceAll(storage, marker, image)
	}
	return storage, nil
}

// Regions of storage whose bytes are content rather than markup: an escaped
// tag is meant literally there, and a newline is significant. Both of the
// rewrites below step over them untouched.
var protectedRegions = [][2]string{
	{"<code>", "</code>"},
	{"<pre>", "</pre>"},
	{"<ac:plain-text-body>", "</ac:plain-text-body>"},
}

// protectedSpan returns the length of the protected region beginning at
// storage[i:], or 0 when none begins there. An unclosed region runs to the end
// rather than being treated as ordinary markup, so malformed input loses no
// content.
func protectedSpan(storage string, i int) int {
	for _, r := range protectedRegions {
		if !strings.HasPrefix(storage[i:], r[0]) {
			continue
		}
		if end := strings.Index(storage[i:], r[1]); end >= 0 {
			return end + len(r[1])
		}
		return len(storage) - i
	}
	return 0
}

// RestoreHardBreaks turns an escaped <br/> back into a real one, which
// Confluence converts to an ADF hardBreak.
func RestoreHardBreaks(storage string) string {
	var b strings.Builder
	b.Grow(len(storage))

	for i := 0; i < len(storage); {
		if n := protectedSpan(storage, i); n > 0 {
			b.WriteString(storage[i : i+n])
			i += n
			continue
		}

		switch {
		case strings.HasPrefix(storage[i:], "&lt;br/&gt;"):
			b.WriteString("<br/>")
			i += len("&lt;br/&gt;")
		case strings.HasPrefix(storage[i:], "&lt;br&gt;"):
			b.WriteString("<br/>")
			i += len("&lt;br&gt;")
		default:
			b.WriteByte(storage[i])
			i++
		}
	}

	return b.String()
}

// CollapseSoftBreaks turns the newlines Confluence leaves inside text into
// single spaces.
//
// Markdown says a lone newline inside a paragraph is a space, but Confluence
// only half agrees: its markdown-to-storage step keeps the source's line
// breaks as literal newlines inside <p>, and its storage-to-ADF step copies
// them into the ADF text nodes, where the renderer draws every one as a line
// break. A document wrapped at 100 columns therefore publishes with a ragged
// break every 100 characters. This restores the paragraph.
//
// A deliberate hard break is unaffected: the converter has already turned it
// into a <br /> tag, and collapsing the newline that trails the tag is what
// stops one requested break from rendering as two.
//
// Whitespace between two block-level tags is kept exactly as it was: it is
// insignificant there, and preserving it stops the storage that --dry-run
// writes out for a human to read from collapsing onto a single line. The test
// is deliberately one-sided. A tag this does not recognise counts as inline,
// so the whitespace collapses — harmless between blocks, and necessary between
// inline tags, where a newline is text and would break the line after all.
func CollapseSoftBreaks(storage string) string {
	out := make([]byte, 0, len(storage))

	for i := 0; i < len(storage); {
		if n := protectedSpan(storage, i); n > 0 {
			out = append(out, storage[i:i+n]...)
			i += n
			continue
		}

		if c := storage[i]; c != '\n' && c != '\r' {
			out = append(out, c)
			i++
			continue
		}

		// Take the whole run, the next line's indentation included, so a
		// rewrapped line does not rejoin with a row of spaces.
		start := i
		for i < len(storage) && isASCIISpace(storage[i]) {
			i++
		}
		switch {
		case i >= len(storage):
			// Trailing whitespace: nothing follows for it to separate.
		case isBlockTag(precedingTag(out)) && isBlockTag(followingTag(storage[i:])):
			out = append(out, storage[start:i]...)
		case bytes.HasSuffix(out, []byte("<br/>")) || bytes.HasSuffix(out, []byte("<br />")):
			// A space here would indent the line the author asked to break.
		case len(out) > 0 && out[len(out)-1] != ' ':
			out = append(out, ' ')
		}
	}

	return string(out)
}

// Block-level tags in the storage Confluence generates. Whitespace directly
// around one of these is layout, never content. The list errs towards being
// short: omitting a tag only costs a newline in the generated file.
var blockTags = map[string]bool{
	"p": true, "div": true, "hr": true, "blockquote": true, "pre": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"ul": true, "ol": true, "li": true,
	"table": true, "thead": true, "tbody": true, "tfoot": true,
	"tr": true, "td": true, "th": true, "colgroup": true, "col": true,
	"ac:structured-macro": true, "ac:parameter": true,
	"ac:rich-text-body": true, "ac:plain-text-body": true,
	"ac:layout": true, "ac:layout-section": true, "ac:layout-cell": true,
	"ac:task-list": true, "ac:task": true,
}

func isBlockTag(name string) bool { return blockTags[name] }

// precedingTag names the tag that written output ends with, or "" when it does
// not end with one. The backward scan is bounded: a tag longer than this is
// not one of the names above.
func precedingTag(out []byte) string {
	if len(out) == 0 || out[len(out)-1] != '>' {
		return ""
	}
	start := len(out) - 1
	for limit := 0; start > 0 && limit < maxTagLen; limit++ {
		start--
		if out[start] == '<' {
			return tagName(string(out[start:]))
		}
	}
	return ""
}

// followingTag names the tag that rest begins with, or "" when it does not
// begin with one.
func followingTag(rest string) string {
	if !strings.HasPrefix(rest, "<") {
		return ""
	}
	return tagName(rest)
}

const maxTagLen = 64

// tagName reads the element name out of a tag, ignoring whether it opens or
// closes and ignoring any attributes.
func tagName(tag string) string {
	name := strings.TrimPrefix(tag[1:], "/")
	end := strings.IndexAny(name, " \t\r\n/>")
	if end < 0 {
		return ""
	}
	return name[:end]
}

func isASCIISpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Code-block width constants, measured in the rendered page rather than
// guessed: Confluence sets code in 14px Atlassian Mono, where every character
// advances exactly 8.4px (canvas metrics and a DOM Range agree), and the
// line-number gutter plus left padding occupies 41px. codeBlockChrome adds a
// right-hand allowance on top of that gutter.
//
// These are estimates from one browser at one zoom level, which is precisely
// why wrap is written unconditionally: an underestimate only makes a block
// wrap a little sooner, and an overestimate only makes it slightly wide.
// Neither can bring back the horizontal overflow that wrap exists to prevent.
const (
	codeCharWidth    = 8.4
	codeBlockChrome  = 60
	codeDefaultWidth = 760
	codeMaxBreakout  = 1011
	codeTabColumns   = 4
)

// codeBlockWidth returns the breakout width a block needs, or 0 when it fits
// the default content column and should not break out at all. Widening every
// block to the same size leaves a four-line snippet stretched as wide as a
// sixty-line schema, which is the disproportion this avoids.
func codeBlockWidth(body string) int {
	longest := 0
	for _, line := range strings.Split(body, "\n") {
		if n := displayColumns(line); n > longest {
			longest = n
		}
	}
	if longest == 0 {
		return 0
	}
	needed := codeBlockChrome + int(math.Ceil(float64(longest)*codeCharWidth))
	if needed <= codeDefaultWidth {
		return 0
	}
	if needed > codeMaxBreakout {
		return codeMaxBreakout
	}
	return needed
}

// displayColumns counts a line in rendered columns rather than runes, so a
// tab-indented block is not measured as narrower than it draws.
func displayColumns(line string) int {
	cols := 0
	for _, r := range line {
		if r == '\t' {
			cols += codeTabColumns - cols%codeTabColumns
			continue
		}
		cols++
	}
	return cols
}

// codeBlockBody returns the literal source inside the next plain-text body.
func codeBlockBody(after string) string {
	start := strings.Index(after, "<ac:plain-text-body>")
	if start < 0 {
		return ""
	}
	body := after[start+len("<ac:plain-text-body>"):]
	if end := strings.Index(body, "</ac:plain-text-body>"); end >= 0 {
		body = body[:end]
	}
	if open := strings.Index(body, "<![CDATA["); open >= 0 {
		body = body[open+len("<![CDATA["):]
		if close := strings.LastIndex(body, "]]>"); close >= 0 {
			body = body[:close]
		}
	}
	return body
}

// StyleCodeBlocks turns on word-wrap and the wide breakout for every code
// macro Confluence's Markdown converter emits, matching the parameters
// Confluence's own editor writes when a user manually widens a code block.
// Confluence silently drops any parameter name other than "wrap" for the
// wrap behaviour itself (wrapText, nowrap, collapse, linenumbers were all
// probed and dropped), and "wrap" alone does not widen the column, so both
// wrap and the breakout pair are needed together.
//
// Insertion is anchored on each macro's opening tag rather than found by
// scanning the body, so a <ac:plain-text-body> CDATA payload that happens to
// contain text resembling a macro tag is copied through untouched instead of
// being mistaken for a real one.
func StyleCodeBlocks(storage string) string {
	var b strings.Builder
	b.Grow(len(storage))

	i := 0
	for i < len(storage) {
		if strings.HasPrefix(storage[i:], "<ac:plain-text-body>") {
			end := strings.Index(storage[i:], "</ac:plain-text-body>")
			if end < 0 {
				end = len(storage) - i
			} else {
				end += len("</ac:plain-text-body>")
			}
			b.WriteString(storage[i : i+end])
			i += end
			continue
		}

		if strings.HasPrefix(storage[i:], "<ac:structured-macro") {
			if tagEnd := strings.IndexByte(storage[i:], '>'); tagEnd >= 0 {
				tag := storage[i : i+tagEnd+1]
				selfClosed := strings.HasSuffix(strings.TrimRight(tag[:len(tag)-1], " "), "/")
				if !selfClosed && strings.Contains(tag, `ac:name="code"`) {
					b.WriteString(tag)
					i += tagEnd + 1
					b.WriteString(codeBlockParams(storage[i:]))
					continue
				}
			}
		}

		b.WriteByte(storage[i])
		i++
	}

	return b.String()
}

// codeBlockParams returns the <ac:parameter> elements to insert right after a
// code macro's opening tag: whichever of wrap/breakoutMode/breakoutWidth are
// not already set somewhere ahead of this macro's own body or close tag.
// after is everything in the document following the opening tag, so the
// search window is bounded to this one macro rather than running to the end
// of the document.
func codeBlockParams(after string) string {
	boundary := len(after)
	if idx := strings.Index(after, "<ac:plain-text-body>"); idx >= 0 && idx < boundary {
		boundary = idx
	}
	if idx := strings.Index(after, "</ac:structured-macro>"); idx >= 0 && idx < boundary {
		boundary = idx
	}
	existing := after[:boundary]

	var params strings.Builder
	// wrap is unconditional. It is the guarantee that no line overflows,
	// whatever width the block settles at; the width below is only an
	// optimisation on top of it, and is derived from an estimate.
	if !strings.Contains(existing, `ac:name="wrap"`) {
		params.WriteString(`<ac:parameter ac:name="wrap">true</ac:parameter>`)
	}

	// A block whose longest line already fits the text column keeps the text
	// column's width, so short snippets sit at the same measure as the prose
	// around them instead of being stretched to match the widest block.
	if codeBlockWidth(codeBlockBody(after)) == 0 {
		return params.String()
	}

	if !strings.Contains(existing, `ac:name="breakoutMode"`) {
		params.WriteString(`<ac:parameter ac:name="breakoutMode">wide</ac:parameter>`)
	}
	if !strings.Contains(existing, `ac:name="breakoutWidth"`) {
		fmt.Fprintf(&params, `<ac:parameter ac:name="breakoutWidth">%d</ac:parameter>`,
			codeBlockWidth(codeBlockBody(after)))
	}
	return params.String()
}
