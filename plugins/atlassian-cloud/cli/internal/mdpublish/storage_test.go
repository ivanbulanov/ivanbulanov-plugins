package mdpublish

import (
	"strings"
	"testing"
)

func TestSpliceTOC(t *testing.T) {
	got := SpliceTOC("<p>@@TOC@@</p><h2>One</h2>")
	if strings.Contains(got, "@@TOC@@") {
		t.Errorf("marker survived: %q", got)
	}
	if !strings.Contains(got, `<ac:structured-macro ac:name="toc"`) {
		t.Errorf("toc macro missing: %q", got)
	}
	if !strings.HasSuffix(got, "<h2>One</h2>") {
		t.Errorf("macro placed wrongly: %q", got)
	}
}

func TestSpliceImages(t *testing.T) {
	got, err := SpliceImages("<p>a</p><p>@@ph0@@</p><p>b</p>",
		map[string]string{"@@ph0@@": "flow-abc.svg"},
		map[string]int{"@@ph0@@": 1188},
		map[string]int{"@@ph0@@": 490})
	if err != nil {
		t.Fatalf("SpliceImages: %v", err)
	}
	if !strings.Contains(got, `<ri:attachment ri:filename="flow-abc.svg" />`) {
		t.Errorf("attachment reference missing: %q", got)
	}
	if !strings.Contains(got, `ac:width="992"`) {
		t.Errorf("want width capped at 992: %q", got)
	}
	// The real diagrams render at 1104px and 1282px, so both clamp. "wide" is
	// the layout the prototype verified renders correctly at that width.
	if !strings.Contains(got, `ac:layout="wide"`) {
		t.Errorf("want the wide layout: %q", got)
	}
	// ac:original-width/height carry the unclamped intrinsic size, which is
	// what Confluence actually reads to size the placeholder box; ac:width
	// alone (or ac:height) is not enough, per the live conversion API.
	if !strings.Contains(got, `ac:original-width="1188"`) {
		t.Errorf("want the unclamped intrinsic width: %q", got)
	}
	if !strings.Contains(got, `ac:original-height="490"`) {
		t.Errorf("want the intrinsic height: %q", got)
	}
	if strings.Contains(got, `ac:height=`) {
		t.Errorf("ac:height is silently dropped by Confluence's converter and must not be emitted: %q", got)
	}
}

func TestSpliceImagesDoesNotUpscale(t *testing.T) {
	got, err := SpliceImages("<p>@@ph0@@</p>",
		map[string]string{"@@ph0@@": "small.svg"},
		map[string]int{"@@ph0@@": 400},
		map[string]int{"@@ph0@@": 150})
	if err != nil {
		t.Fatalf("SpliceImages: %v", err)
	}
	if !strings.Contains(got, `ac:width="400"`) {
		t.Errorf("a narrow diagram must not be blown up: %q", got)
	}
	if !strings.Contains(got, `ac:original-width="400"`) || !strings.Contains(got, `ac:original-height="150"`) {
		t.Errorf("want the intrinsic size recorded even when it is not clamped: %q", got)
	}
}

func TestSpliceImagesOmitsOriginalSizeWhenHeightUnknown(t *testing.T) {
	// A regular Markdown image has a filename but no measured width or
	// height; a wrong intrinsic size is worse than none, so neither
	// ac:original-width nor ac:original-height should appear.
	got, err := SpliceImages("<p>@@ph0@@</p>",
		map[string]string{"@@ph0@@": "photo.png"},
		map[string]int{},
		map[string]int{})
	if err != nil {
		t.Fatalf("SpliceImages: %v", err)
	}
	if strings.Contains(got, `ac:original-width=`) || strings.Contains(got, `ac:original-height=`) {
		t.Errorf("must not emit a bogus intrinsic size: %q", got)
	}
}

func TestSpliceImagesReportsMissingPlaceholder(t *testing.T) {
	_, err := SpliceImages("<p>nothing here</p>",
		map[string]string{"@@ph0@@": "x.svg"}, map[string]int{"@@ph0@@": 100}, map[string]int{"@@ph0@@": 50})
	if err == nil {
		t.Fatal("want an error when a placeholder is not present in the storage")
	}
}

func TestRestoreHardBreaks(t *testing.T) {
	got := RestoreHardBreaks("<p>one&lt;br/&gt;two</p>")
	if got != "<p>one<br/>two</p>" {
		t.Errorf("got %q", got)
	}
}

func TestRestoreHardBreaksLeavesCodeAlone(t *testing.T) {
	in := "<p>a <code>&lt;br/&gt;</code> b</p><ac:plain-text-body><![CDATA[x &lt;br/&gt; y]]></ac:plain-text-body>"
	if got := RestoreHardBreaks(in); got != in {
		t.Errorf("code content changed:\n got %q\nwant %q", got, in)
	}
}

func TestStyleCodeBlocksAddsAllThreeParams(t *testing.T) {
	// Long enough that the block genuinely needs to break out of the column.
	in := codeMacro("SELECT " + strings.Repeat("a", 200))

	got := StyleCodeBlocks(in)

	for _, want := range []string{
		`<ac:parameter ac:name="wrap">true</ac:parameter>`,
		`<ac:parameter ac:name="breakoutMode">wide</ac:parameter>`,
		`<ac:parameter ac:name="breakoutWidth">1011</ac:parameter>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in: %s", want, got)
		}
	}
	if !strings.Contains(got, "SELECT ") {
		t.Errorf("body lost: %s", got)
	}
}

func TestStyleCodeBlocksHandlesMacroWithoutLanguage(t *testing.T) {
	in := `<ac:structured-macro ac:name="code" ac:schema-version="1">` +
		`<ac:plain-text-body><![CDATA[bare fence ` + strings.Repeat("w", 200) +
		`]]></ac:plain-text-body></ac:structured-macro>`

	got := StyleCodeBlocks(in)

	for _, want := range []string{`ac:name="wrap"`, `ac:name="breakoutMode"`, `ac:name="breakoutWidth"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in: %s", want, got)
		}
	}
}

func TestStyleCodeBlocksIsIdempotent(t *testing.T) {
	// Both paths have to be stable: a narrow block gains only wrap, a wide one
	// gains all three, and neither may accumulate duplicates on a second pass.
	cases := map[string]struct {
		body            string
		wantBreakoutSet bool
	}{
		"narrow block": {"x", false},
		"wide block":   {strings.Repeat("x", 200), true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			once := StyleCodeBlocks(codeMacro(tc.body))
			twice := StyleCodeBlocks(once)

			if once != twice {
				t.Errorf("second application changed the output:\n got  %s\nwant  %s", twice, once)
			}
			if n := strings.Count(twice, `ac:name="wrap"`); n != 1 {
				t.Errorf("wrap appears %d times, want 1: %s", n, twice)
			}
			for _, name := range []string{"breakoutMode", "breakoutWidth"} {
				want := 0
				if tc.wantBreakoutSet {
					want = 1
				}
				if n := strings.Count(twice, `ac:name="`+name+`"`); n != want {
					t.Errorf("%s appears %d times, want %d: %s", name, n, want, twice)
				}
			}
		})
	}
}

func TestStyleCodeBlocksKeepsExistingBreakoutMode(t *testing.T) {
	// Wide enough to reach the breakout branch, so an existing choice is
	// genuinely at risk of being overwritten.
	in := `<ac:structured-macro ac:name="code" ac:schema-version="1">` +
		`<ac:parameter ac:name="breakoutMode">full-width</ac:parameter>` +
		`<ac:plain-text-body><![CDATA[` + strings.Repeat("x", 200) +
		`]]></ac:plain-text-body></ac:structured-macro>`

	got := StyleCodeBlocks(in)

	if !strings.Contains(got, `<ac:parameter ac:name="breakoutMode">full-width</ac:parameter>`) {
		t.Errorf("an existing breakout mode must be left alone: %s", got)
	}
	if strings.Contains(got, `<ac:parameter ac:name="breakoutMode">wide</ac:parameter>`) {
		t.Errorf("a second breakout mode was added: %s", got)
	}
}

func TestStyleCodeBlocksLeavesCDATALookalikeAlone(t *testing.T) {
	in := `<ac:structured-macro ac:name="code" ac:schema-version="1">` +
		`<ac:plain-text-body><![CDATA[see <ac:structured-macro ac:name="code"> in the docs]]></ac:plain-text-body>` +
		`</ac:structured-macro>`

	got := StyleCodeBlocks(in)

	if !strings.Contains(got, `<![CDATA[see <ac:structured-macro ac:name="code"> in the docs]]>`) {
		t.Errorf("CDATA body was corrupted: %s", got)
	}
	if n := strings.Count(got, `ac:name="wrap"`); n != 1 {
		t.Errorf("wrap appears %d times, want exactly 1 (only on the real macro): %s", n, got)
	}
}

func codeMacro(body string) string {
	return `<ac:structured-macro ac:name="code" ac:schema-version="1">` +
		`<ac:parameter ac:name="language">sql</ac:parameter>` +
		`<ac:plain-text-body><![CDATA[` + body + `]]></ac:plain-text-body>` +
		`</ac:structured-macro>`
}

// Every block used to be widened to the same 1011px, so a four-line snippet
// was stretched as wide as a sixty-line schema. Width now follows content.
func TestCodeBlockWidthFollowsContent(t *testing.T) {
	// 760px column minus 60px of gutter and padding leaves 700px of text room,
	// which is 83 characters at 8.4px each.
	tests := map[string]struct {
		chars int
		want  int
	}{
		"tiny snippet":      {16, 0},
		"short request":     {41, 0},
		"just inside":       {83, 0},
		"just outside":      {84, 766},
		"typical long line": {98, 884},
		"enormous":          {432, codeMaxBreakout},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := codeBlockWidth(strings.Repeat("x", tc.chars)); got != tc.want {
				t.Errorf("codeBlockWidth(%d chars) = %d, want %d", tc.chars, got, tc.want)
			}
		})
	}
}

func TestCodeBlockWidthUsesTheLongestLine(t *testing.T) {
	// 100 columns needs 60 + 840 = 900px, comfortably inside the 1011 cap, so
	// this checks the measurement rather than the clamp.
	body := "short\n" + strings.Repeat("y", 100) + "\nshort again\n"
	if got, want := codeBlockWidth(body), 900; got != want {
		t.Errorf("codeBlockWidth = %d, want %d", got, want)
	}
	// A tab draws as several columns, so it must not be counted as one.
	plain := codeBlockWidth(strings.Repeat("z", 84))
	tabbed := codeBlockWidth("\t" + strings.Repeat("z", 84))
	if tabbed <= plain {
		t.Errorf("tab-indented line measured %d, not wider than %d", tabbed, plain)
	}
}

func TestStyleCodeBlocksLeavesSmallBlocksAtTextWidth(t *testing.T) {
	small := StyleCodeBlocks(codeMacro("SELECT 1;\nSELECT 2;"))
	if !strings.Contains(small, `ac:name="wrap"`) {
		t.Error("wrap must be written on every block, whatever its width")
	}
	if strings.Contains(small, "breakout") {
		t.Errorf("a short block must not break out: %q", small)
	}

	wide := StyleCodeBlocks(codeMacro(strings.Repeat("q", 200)))
	if !strings.Contains(wide, `ac:name="breakoutMode"`) {
		t.Errorf("a long-lined block must break out: %q", wide)
	}
	if !strings.Contains(wide, `<ac:parameter ac:name="breakoutWidth">1011</ac:parameter>`) {
		t.Errorf("want the width capped at 1011: %q", wide)
	}
}
