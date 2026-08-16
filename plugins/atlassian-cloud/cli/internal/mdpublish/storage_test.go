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
		map[string]int{"@@ph0@@": 1188})
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
}

func TestSpliceImagesDoesNotUpscale(t *testing.T) {
	got, err := SpliceImages("<p>@@ph0@@</p>",
		map[string]string{"@@ph0@@": "small.svg"},
		map[string]int{"@@ph0@@": 400})
	if err != nil {
		t.Fatalf("SpliceImages: %v", err)
	}
	if !strings.Contains(got, `ac:width="400"`) {
		t.Errorf("a narrow diagram must not be blown up: %q", got)
	}
}

func TestSpliceImagesReportsMissingPlaceholder(t *testing.T) {
	_, err := SpliceImages("<p>nothing here</p>",
		map[string]string{"@@ph0@@": "x.svg"}, map[string]int{"@@ph0@@": 100})
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
