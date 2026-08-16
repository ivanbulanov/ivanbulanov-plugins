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
	in := `<ac:structured-macro ac:name="code" ac:schema-version="1">` +
		`<ac:parameter ac:name="language">sql</ac:parameter>` +
		`<ac:plain-text-body><![CDATA[SELECT 1]]></ac:plain-text-body></ac:structured-macro>`

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
	if !strings.Contains(got, "SELECT 1") {
		t.Errorf("body lost: %s", got)
	}
}

func TestStyleCodeBlocksHandlesMacroWithoutLanguage(t *testing.T) {
	in := `<ac:structured-macro ac:name="code" ac:schema-version="1">` +
		`<ac:plain-text-body><![CDATA[bare fence]]></ac:plain-text-body></ac:structured-macro>`

	got := StyleCodeBlocks(in)

	for _, want := range []string{`ac:name="wrap"`, `ac:name="breakoutMode"`, `ac:name="breakoutWidth"`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in: %s", want, got)
		}
	}
}

func TestStyleCodeBlocksIsIdempotent(t *testing.T) {
	in := `<ac:structured-macro ac:name="code" ac:schema-version="1">` +
		`<ac:plain-text-body><![CDATA[x]]></ac:plain-text-body></ac:structured-macro>`

	once := StyleCodeBlocks(in)
	twice := StyleCodeBlocks(once)

	for _, name := range []string{"wrap", "breakoutMode", "breakoutWidth"} {
		attr := `ac:name="` + name + `"`
		if n := strings.Count(twice, attr); n != 1 {
			t.Errorf("%s appears %d times after two applications, want 1: %s", name, n, twice)
		}
	}
	if once != twice {
		t.Errorf("second application changed the output:\n got  %s\nwant  %s", twice, once)
	}
}

func TestStyleCodeBlocksKeepsExistingBreakoutMode(t *testing.T) {
	in := `<ac:structured-macro ac:name="code" ac:schema-version="1">` +
		`<ac:parameter ac:name="breakoutMode">full-width</ac:parameter>` +
		`<ac:plain-text-body><![CDATA[x]]></ac:plain-text-body></ac:structured-macro>`

	got := StyleCodeBlocks(in)

	if !strings.Contains(got, `<ac:parameter ac:name="breakoutMode">full-width</ac:parameter>`) {
		t.Errorf("existing breakoutMode value must survive: %s", got)
	}
	if n := strings.Count(got, `ac:name="breakoutMode"`); n != 1 {
		t.Errorf("breakoutMode appears %d times, want 1: %s", n, got)
	}
	if !strings.Contains(got, `ac:name="wrap"`) || !strings.Contains(got, `ac:name="breakoutWidth"`) {
		t.Errorf("wrap/breakoutWidth were absent and must still be added: %s", got)
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
