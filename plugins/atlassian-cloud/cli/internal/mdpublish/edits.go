package mdpublish

import (
	"fmt"
	"sort"
	"strings"
)

// Edit replaces src[Start:End] with Replacement. Offsets are byte offsets
// into the original source, so a caller can collect edits from an AST walk
// without tracking how earlier edits shifted the text.
type Edit struct {
	Start       int
	End         int
	Replacement string
}

// ApplyEdits applies edits to src, last first, so that each edit's offsets
// still refer to the original bytes. Overlapping edits are a programming
// error and are reported rather than silently resolved.
func ApplyEdits(src []byte, edits []Edit) (string, error) {
	ordered := make([]Edit, len(edits))
	copy(ordered, edits)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Start < ordered[j].Start })

	for i, e := range ordered {
		if e.Start < 0 || e.End > len(src) || e.Start > e.End {
			return "", fmt.Errorf("edit [%d:%d] is out of range for %d bytes", e.Start, e.End, len(src))
		}
		if i > 0 && e.Start < ordered[i-1].End {
			return "", fmt.Errorf("edit [%d:%d] overlaps [%d:%d]",
				e.Start, e.End, ordered[i-1].Start, ordered[i-1].End)
		}
	}

	var b strings.Builder
	prev := 0
	for _, e := range ordered {
		b.Write(src[prev:e.Start])
		b.WriteString(e.Replacement)
		prev = e.End
	}
	b.Write(src[prev:])

	return b.String(), nil
}
