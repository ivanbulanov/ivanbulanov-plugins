package mdpublish

import (
	"strings"
	"testing"
)

func TestVersionMarkerRoundTrip(t *testing.T) {
	msg := VersionMarker("abc123")
	if !IsOurVersion(msg) {
		t.Errorf("our own marker not recognised: %q", msg)
	}
	if !strings.Contains(msg, "abc123") {
		t.Errorf("marker must carry the source hash: %q", msg)
	}
}

func TestIsOurVersionRejectsHumanEdits(t *testing.T) {
	for _, msg := range []string{"", "typo fix", "Updated by Ivan", "atlassian-cloud"} {
		if IsOurVersion(msg) {
			t.Errorf("%q must not count as a tool version", msg)
		}
	}
}

func TestShouldSkipWhenNothingChanges(t *testing.T) {
	current := PageState{ADF: adfParagraphs("same words")}
	skip, err := ShouldSkip(current, adfParagraphs("same words"))
	if err != nil {
		t.Fatalf("ShouldSkip: %v", err)
	}
	if !skip {
		t.Error("want skip — re-publishing identical content must add no version")
	}
}

func TestShouldNotSkipWhenTextChanges(t *testing.T) {
	current := PageState{ADF: adfParagraphs("old words")}
	skip, err := ShouldSkip(current, adfParagraphs("new words"))
	if err != nil {
		t.Fatalf("ShouldSkip: %v", err)
	}
	if skip {
		t.Error("want publish")
	}
}

// A formatting-only change leaves the text byte-identical. If the no-op guard
// looks at text alone it reports "no change" and the fix never reaches the
// page — which is exactly what happened with code wrapping and diagram sizing.
func TestShouldSkipSeesFormattingOnlyChanges(t *testing.T) {
	plain := []byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"codeBlock","attrs":{"language":"sql"},` +
		`"content":[{"type":"text","text":"SELECT 1"}]}]}`)
	styled := []byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"codeBlock","attrs":{"language":"sql","wrap":true},` +
		`"marks":[{"type":"breakout","attrs":{"mode":"wide","width":1011}}],` +
		`"content":[{"type":"text","text":"SELECT 1"}]}]}`)

	plainText, err := Text(plain)
	if err != nil {
		t.Fatal(err)
	}
	styledText, err := Text(styled)
	if err != nil {
		t.Fatal(err)
	}
	if plainText != styledText {
		t.Fatalf("fixture is wrong: the texts must be identical, got %q and %q", plainText, styledText)
	}

	skip, err := ShouldSkip(PageState{ADF: plain}, styled)
	if err != nil {
		t.Fatalf("ShouldSkip: %v", err)
	}
	if skip {
		t.Error("want a republish: the code block gained wrap and breakout")
	}

	skip, err = ShouldSkip(PageState{ADF: styled}, styled)
	if err != nil {
		t.Fatalf("ShouldSkip: %v", err)
	}
	if !skip {
		t.Error("want a skip: the page already says exactly this")
	}
}

// Confluence stamps its own identifiers into the document it returns. Those
// must not read as a change, or every run would publish a new version.
func TestStructuralFingerprintIgnoresGeneratedIdentifiers(t *testing.T) {
	withIDs := []byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"extension","attrs":{"extensionKey":"toc","localId":"abc",` +
		`"parameters":{"macroMetadata":{"macroId":{"value":"11111111"}}}}},` +
		`{"type":"mediaSingle","attrs":{"layout":"wide","width":992},"content":[` +
		`{"type":"media","attrs":{"__fileName":"d.svg","width":1104,"height":491,` +
		`"id":"uuid-one","collection":"contentId-1","__fileSize":10}}]}]}`)
	sameShapeOtherIDs := []byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"extension","attrs":{"extensionKey":"toc","localId":"zzz",` +
		`"parameters":{"macroMetadata":{"macroId":{"value":"99999999"}}}}},` +
		`{"type":"mediaSingle","attrs":{"layout":"wide","width":992},"content":[` +
		`{"type":"media","attrs":{"__fileName":"d.svg","width":1104,"height":491,` +
		`"id":"uuid-two","collection":"contentId-2","__fileSize":20}}]}]}`)

	a, err := StructuralFingerprint(withIDs)
	if err != nil {
		t.Fatalf("StructuralFingerprint: %v", err)
	}
	b, err := StructuralFingerprint(sameShapeOtherIDs)
	if err != nil {
		t.Fatalf("StructuralFingerprint: %v", err)
	}
	if a != b {
		t.Error("generated identifiers must not count as a change")
	}

	resized := []byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"extension","attrs":{"extensionKey":"toc"}},` +
		`{"type":"mediaSingle","attrs":{"layout":"wide","width":992},"content":[` +
		`{"type":"media","attrs":{"__fileName":"d.svg","width":1,"height":1}}]}]}`)
	c, err := StructuralFingerprint(resized)
	if err != nil {
		t.Fatalf("StructuralFingerprint: %v", err)
	}
	if a == c {
		t.Error("a diagram recorded as 1x1 must differ from one with real dimensions")
	}
}

// Converting storage to a document adds __fileName, __fileSize and
// __fileMimeType to every media node; reading the same document back off the
// page does not. Counting them made an unchanged republish look like a change,
// so the guard published a new version on every run.
func TestStructuralFingerprintIgnoresConverterOnlyAttributes(t *testing.T) {
	asStored := []byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"mediaSingle","attrs":{"layout":"wide","width":992},"content":[` +
		`{"type":"media","attrs":{"alt":"d.svg","width":1104,"height":491}}]}]}`)
	asConverted := []byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"mediaSingle","attrs":{"layout":"wide","width":992},"content":[` +
		`{"type":"media","attrs":{"alt":"d.svg","width":1104,"height":491,` +
		`"__fileName":"d.svg","__fileSize":26490,"__fileMimeType":"image/svg+xml"}}]}]}`)

	stored, err := StructuralFingerprint(asStored)
	if err != nil {
		t.Fatalf("StructuralFingerprint: %v", err)
	}
	converted, err := StructuralFingerprint(asConverted)
	if err != nil {
		t.Fatalf("StructuralFingerprint: %v", err)
	}
	if stored != converted {
		t.Error("converter-only attributes must not read as a change")
	}

	// A different diagram must still register: the filename is content-addressed.
	changed, err := StructuralFingerprint([]byte(`{"type":"doc","version":1,"content":[` +
		`{"type":"mediaSingle","attrs":{"layout":"wide","width":992},"content":[` +
		`{"type":"media","attrs":{"alt":"other.svg","width":1104,"height":491}}]}]}`))
	if err != nil {
		t.Fatalf("StructuralFingerprint: %v", err)
	}
	if changed == stored {
		t.Error("a different attachment must register as a change")
	}
}
