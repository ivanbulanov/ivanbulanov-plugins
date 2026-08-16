package mdpublish

import (
	"fmt"
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

// RestoreHardBreaks turns an escaped <br/> back into a real one, which
// Confluence converts to an ADF hardBreak. Inside <code> and inside a code
// macro's body an escaped tag is content, so those regions are skipped.
func RestoreHardBreaks(storage string) string {
	var b strings.Builder
	b.Grow(len(storage))

	protectedOpen := []string{"<code>", "<ac:plain-text-body>"}
	protectedClose := []string{"</code>", "</ac:plain-text-body>"}

	i := 0
	for i < len(storage) {
		skipped := false
		for k, open := range protectedOpen {
			if strings.HasPrefix(storage[i:], open) {
				end := strings.Index(storage[i:], protectedClose[k])
				if end < 0 {
					end = len(storage) - i
				} else {
					end += len(protectedClose[k])
				}
				b.WriteString(storage[i : i+end])
				i += end
				skipped = true
				break
			}
		}
		if skipped {
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

// codeBreakoutWidth is the pixel width Confluence's own editor writes into
// ac:breakoutWidth when a user sets a code block to the wide breakout: 1011.
// wrap is what actually stops a long line overflowing its column; breakout
// only buys the block extra room to use before wrap has to kick in.
const codeBreakoutWidth = 1011

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
	if !strings.Contains(existing, `ac:name="wrap"`) {
		params.WriteString(`<ac:parameter ac:name="wrap">true</ac:parameter>`)
	}
	if !strings.Contains(existing, `ac:name="breakoutMode"`) {
		params.WriteString(`<ac:parameter ac:name="breakoutMode">wide</ac:parameter>`)
	}
	if !strings.Contains(existing, `ac:name="breakoutWidth"`) {
		fmt.Fprintf(&params, `<ac:parameter ac:name="breakoutWidth">%d</ac:parameter>`, codeBreakoutWidth)
	}
	return params.String()
}
