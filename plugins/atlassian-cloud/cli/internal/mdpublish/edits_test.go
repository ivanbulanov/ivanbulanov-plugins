package mdpublish

import "testing"

func TestApplyEdits(t *testing.T) {
	src := []byte("alpha beta gamma")
	got, err := ApplyEdits(src, []Edit{
		{Start: 11, End: 16, Replacement: "GAMMA"},
		{Start: 0, End: 5, Replacement: "ALPHA"},
	})
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if got != "ALPHA beta GAMMA" {
		t.Errorf("got %q", got)
	}
}

func TestApplyEditsDeletion(t *testing.T) {
	got, err := ApplyEdits([]byte("keep drop keep"), []Edit{{Start: 5, End: 10, Replacement: ""}})
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if got != "keep keep" {
		t.Errorf("got %q", got)
	}
}

func TestApplyEditsRejectsOverlap(t *testing.T) {
	_, err := ApplyEdits([]byte("abcdef"), []Edit{
		{Start: 0, End: 3, Replacement: "X"},
		{Start: 2, End: 5, Replacement: "Y"},
	})
	if err == nil {
		t.Fatal("want an error on overlapping edits")
	}
}

func TestApplyEditsRejectsOutOfRange(t *testing.T) {
	if _, err := ApplyEdits([]byte("abc"), []Edit{{Start: 1, End: 99, Replacement: "X"}}); err == nil {
		t.Fatal("want an error when an edit runs past the end")
	}
}

func TestApplyEditsPreservesMultibyte(t *testing.T) {
	// Offsets are byte offsets; an edit must not split a rune.
	src := []byte("D1 — Cache")
	got, err := ApplyEdits(src, []Edit{{Start: 0, End: 2, Replacement: "D9"}})
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if got != "D9 — Cache" {
		t.Errorf("got %q", got)
	}
}
