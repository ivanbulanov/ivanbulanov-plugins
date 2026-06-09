package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// mustHost extracts the host:port portion of a URL for use as a transport's
// scoped host.
func mustHost(t *testing.T, rawURL string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url %q: %v", rawURL, err)
	}
	return u.Host
}

// TestBasicAuthTransport_addsAuthForMatchingHost verifies that credentials are
// attached when the request targets the transport's configured host.
func TestBasicAuthTransport_addsAuthForMatchingHost(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	client := &http.Client{Transport: &basicAuthTransport{
		host:  mustHost(t, srv.URL),
		email: "user@example.com",
		token: "secret-token",
	}}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	resp.Body.Close()

	if gotAuth == "" {
		t.Fatal("expected Authorization header for matching host, got none")
	}
	user, pass, ok := parseBasicAuth(gotAuth)
	if !ok || user != "user@example.com" || pass != "secret-token" {
		t.Errorf("unexpected basic auth: user=%q pass=%q ok=%v", user, pass, ok)
	}
}

// TestBasicAuthTransport_skipsForeignHost verifies that credentials are NOT
// attached when the request targets a host other than the configured one. This
// is what prevents leaking Atlassian credentials to the media CDN that
// attachment downloads redirect to.
func TestBasicAuthTransport_skipsForeignHost(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	client := &http.Client{Transport: &basicAuthTransport{
		host:  "some-other-host.example.com",
		email: "user@example.com",
		token: "secret-token",
	}}

	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	resp.Body.Close()

	if gotAuth != "" {
		t.Errorf("Authorization should not be sent to a foreign host, got %q", gotAuth)
	}
}

// TestBasicAuthTransport_dropsAuthOnCrossHostRedirect models the real download
// flow: the tenant returns a 302 to a different media host, and the credentials
// must NOT follow across that redirect.
func TestBasicAuthTransport_dropsAuthOnCrossHostRedirect(t *testing.T) {
	var mediaCalled bool
	var mediaAuth string
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaCalled = true
		mediaAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("FILEBYTES"))
	}))
	defer media.Close()

	var apiAuth string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiAuth = r.Header.Get("Authorization")
		http.Redirect(w, r, media.URL, http.StatusFound)
	}))
	defer api.Close()

	client := &http.Client{Transport: &basicAuthTransport{
		host:  mustHost(t, api.URL),
		email: "user@example.com",
		token: "secret-token",
	}}

	resp, err := client.Get(api.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if apiAuth == "" {
		t.Error("expected Authorization on the tenant (api) host request")
	}
	if !mediaCalled {
		t.Fatal("media host was never reached")
	}
	if mediaAuth != "" {
		t.Errorf("Authorization leaked to media host: %q", mediaAuth)
	}
	if string(body) != "FILEBYTES" {
		t.Errorf("unexpected body %q", body)
	}
}

// TestBearerTransport_dropsAuthOnCrossHostRedirect is the OAuth equivalent of
// the basic-auth redirect test.
func TestBearerTransport_dropsAuthOnCrossHostRedirect(t *testing.T) {
	var mediaAuth string
	var mediaCalled bool
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mediaCalled = true
		mediaAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("FILEBYTES"))
	}))
	defer media.Close()

	var apiAuth string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiAuth = r.Header.Get("Authorization")
		http.Redirect(w, r, media.URL, http.StatusFound)
	}))
	defer api.Close()

	client := &http.Client{Transport: &bearerTransport{
		host:  mustHost(t, api.URL),
		token: "access-token-123",
	}}

	resp, err := client.Get(api.URL)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	resp.Body.Close()

	if apiAuth != "Bearer access-token-123" {
		t.Errorf("expected bearer token on api host, got %q", apiAuth)
	}
	if !mediaCalled {
		t.Fatal("media host was never reached")
	}
	if mediaAuth != "" {
		t.Errorf("bearer token leaked to media host: %q", mediaAuth)
	}
}

// parseBasicAuth decodes a "Basic <base64>" Authorization header value.
func parseBasicAuth(header string) (user, pass string, ok bool) {
	const prefix = "Basic "
	if len(header) < len(prefix) || header[:len(prefix)] != prefix {
		return "", "", false
	}
	r := &http.Request{Header: http.Header{"Authorization": {header}}}
	return r.BasicAuth()
}
