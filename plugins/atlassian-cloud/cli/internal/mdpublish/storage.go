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
// an attachment by filename. Width is the diagram's own width, capped, so a
// narrow diagram is not blown up to page width.
func SpliceImages(storage string, filenames map[string]string, widths map[string]int) (string, error) {
	for key, filename := range filenames {
		marker := "<p>" + key + "</p>"
		if !strings.Contains(storage, marker) {
			return "", fmt.Errorf("placeholder %s is not in the converted storage", key)
		}
		width := widths[key]
		if width == 0 || width > MaxImageWidth {
			width = MaxImageWidth
		}
		// layout="wide" is what the prototype verified on the live page: at 992px
		// a "center" layout overflows the default content column.
		image := fmt.Sprintf(`<ac:image ac:align="center" ac:layout="wide" ac:custom-width="true" `+
			`ac:width="%d" ac:alt="%s"><ri:attachment ri:filename="%s" /></ac:image>`,
			width, filename, filename)
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
