package confluence

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMarkdownToStorage(t *testing.T) {
	var gotPath, gotRepresentation, gotValue string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		_ = json.Unmarshal(body, &payload)
		gotRepresentation = payload["representation"]
		gotValue = payload["value"]
		_, _ = w.Write([]byte(`{"value":"<h2>Hello</h2>","representation":"storage"}`))
	}))
	defer srv.Close()

	c := NewConverter(srv.Client(), srv.URL)
	got, err := c.MarkdownToStorage(context.Background(), "## Hello")
	if err != nil {
		t.Fatalf("MarkdownToStorage: %v", err)
	}
	if got != "<h2>Hello</h2>" {
		t.Errorf("value = %q, want %q", got, "<h2>Hello</h2>")
	}
	if gotPath != "/rest/api/contentbody/convert/storage" {
		t.Errorf("path = %q", gotPath)
	}
	if gotRepresentation != "markdown" {
		t.Errorf("representation = %q, want markdown", gotRepresentation)
	}
	if gotValue != "## Hello" {
		t.Errorf("value sent = %q", gotValue)
	}
}

func TestStorageToADFPassesPageContext(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"value":"{\"type\":\"doc\"}","representation":"atlas_doc_format"}`))
	}))
	defer srv.Close()

	c := NewConverter(srv.Client(), srv.URL)
	got, err := c.StorageToADF(context.Background(), "<p>x</p>", "123456")
	if err != nil {
		t.Fatalf("StorageToADF: %v", err)
	}
	if string(got) != `{"type":"doc"}` {
		t.Errorf("adf = %s", got)
	}
	if !strings.Contains(gotQuery, "contentIdContext=123456") {
		t.Errorf("query = %q, want contentIdContext", gotQuery)
	}
}

func TestConvertReportsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"nope"}`))
	}))
	defer srv.Close()

	c := NewConverter(srv.Client(), srv.URL)
	if _, err := c.MarkdownToStorage(context.Background(), "x"); err == nil {
		t.Fatal("want an error on HTTP 400")
	}
}

func TestSpaceKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/spaces/987654" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"987654","key":"DOCS","name":"Documentation"}`))
	}))
	defer srv.Close()

	got, err := SpaceKey(context.Background(), srv.Client(), srv.URL, "987654")
	if err != nil {
		t.Fatalf("SpaceKey: %v", err)
	}
	if got != "DOCS" {
		t.Errorf("key = %q, want DOCS", got)
	}
}
