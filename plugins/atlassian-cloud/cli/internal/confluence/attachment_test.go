package confluence

import (
	"context"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploadAttachment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "diagram-abc123.svg")
	if err := os.WriteFile(path, []byte("<svg/>"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotToken, gotFilename, gotContent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Atlassian-Token")
		_, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			if part.FormName() == "file" {
				gotFilename = part.FileName()
				buf := make([]byte, 64)
				n, _ := part.Read(buf)
				gotContent = string(buf[:n])
			}
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"diagram-abc123.svg"}]}`))
	}))
	defer srv.Close()

	title, err := UploadAttachment(context.Background(), srv.Client(), srv.URL, "123", path)
	if err != nil {
		t.Fatalf("UploadAttachment: %v", err)
	}
	if title != "diagram-abc123.svg" {
		t.Errorf("title = %q", title)
	}
	if gotToken != "no-check" {
		t.Errorf("X-Atlassian-Token = %q, want no-check", gotToken)
	}
	if gotFilename != "diagram-abc123.svg" {
		t.Errorf("filename = %q", gotFilename)
	}
	if gotContent != "<svg/>" {
		t.Errorf("content = %q", gotContent)
	}
}

func TestAttachmentTitles(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/child/attachment") {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"a.svg"},{"title":"b.svg"}]}`))
	}))
	defer srv.Close()

	got, err := AttachmentTitles(context.Background(), srv.Client(), srv.URL, "123")
	if err != nil {
		t.Fatalf("AttachmentTitles: %v", err)
	}
	if len(got) != 2 || got[0] != "a.svg" || got[1] != "b.svg" {
		t.Errorf("titles = %v", got)
	}
}

func TestUploadAttachmentReportsHTTPError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.svg")
	if err := os.WriteFile(path, []byte("<svg/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	if _, err := UploadAttachment(context.Background(), srv.Client(), srv.URL, "123", path); err == nil {
		t.Fatal("want an error on HTTP 403")
	}
}
