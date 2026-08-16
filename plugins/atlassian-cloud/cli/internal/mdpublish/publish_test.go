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
