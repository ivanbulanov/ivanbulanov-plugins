package auth

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ctreminiom/go-atlassian/v2/service/common"
)

// Confluence sits under the /wiki context path. go-atlassian appends it
// internally, so code that builds requests by hand must append it too —
// omitting it returns 404 on every endpoint, for both auth methods. Tests that
// inject an httptest server URL cannot catch that, so it is asserted here.
func TestConfluenceRESTBaseIncludesWikiContextPath(t *testing.T) {
	tests := map[string]struct {
		confluenceURL string
		want          string
	}{
		"site url used by the api-token method": {
			"https://example.atlassian.net",
			"https://example.atlassian.net/wiki",
		},
		"gateway url used by the oauth method": {
			"https://api.atlassian.com/ex/confluence/1111-2222",
			"https://api.atlassian.com/ex/confluence/1111-2222/wiki",
		},
		"trailing slash is not doubled": {
			"https://example.atlassian.net/",
			"https://example.atlassian.net/wiki",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			clients, err := buildClients(tc.confluenceURL, tc.confluenceURL,
				func(common.Authentication) {}, http.DefaultClient)
			if err != nil {
				t.Fatalf("buildClients: %v", err)
			}
			if clients.ConfluenceRESTBase != tc.want {
				t.Errorf("ConfluenceRESTBase = %q, want %q", clients.ConfluenceRESTBase, tc.want)
			}
			if strings.Contains(clients.ConfluenceRESTBase, "//wiki") {
				t.Errorf("doubled separator: %q", clients.ConfluenceRESTBase)
			}
		})
	}
}
