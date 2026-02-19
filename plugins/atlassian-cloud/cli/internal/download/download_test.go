package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestToFile_success(t *testing.T) {
	content := "hello world"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(content))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.txt")
	err := ToFile(context.Background(), srv.Client(), srv.URL, dest)
	if err != nil {
		t.Fatalf("ToFile: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestToFile_httpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.txt")
	err := ToFile(context.Background(), srv.Client(), srv.URL, dest)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestToFile_401returnsAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out.txt")
	err := ToFile(context.Background(), srv.Client(), srv.URL, dest)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	// Check that it's an AuthError
	if _, ok := err.(*AuthError); !ok {
		t.Errorf("expected *AuthError, got %T", err)
	}
}
