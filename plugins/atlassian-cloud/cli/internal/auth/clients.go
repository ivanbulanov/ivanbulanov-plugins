package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	confluence "github.com/ctreminiom/go-atlassian/v2/confluence"
	confluencev2 "github.com/ctreminiom/go-atlassian/v2/confluence/v2"
	jira "github.com/ctreminiom/go-atlassian/v2/jira/v2"
	"github.com/ctreminiom/go-atlassian/v2/service/common"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/config"
)

const (
	// ExitCodeAuthRequired is returned when no valid authentication is available.
	ExitCodeAuthRequired = 2

	tokenExpiryBuffer = 5 * time.Minute
)

// AuthRequiredError indicates that authentication is missing or expired.
type AuthRequiredError struct {
	Message string
}

func (e *AuthRequiredError) Error() string { return e.Message }

// WrapAPIError checks an HTTP status code from an API response and returns
// a typed error for 401 (authentication expired). For all other codes it
// returns the original error unchanged. Pass 0 when the response is nil.
func WrapAPIError(httpCode int, err error) error {
	if err == nil {
		return nil
	}
	if httpCode == 401 {
		return &AuthRequiredError{Message: "authentication failed; run: atlassian-cloud auth login"}
	}
	return err
}

// Clients bundles authenticated Jira and Confluence API clients for a single site.
type Clients struct {
	Jira              *jira.Client
	ConfluenceV1      *confluence.Client
	ConfluenceV2      *confluencev2.Client
	JiraBaseURL       string
	ConfluenceBaseURL string
	// ConfluenceRESTBase is the prefix for hand-built Confluence requests and
	// for page URLs. Confluence sits under the /wiki context path on both the
	// site URL and the OAuth gateway; go-atlassian appends it internally, so
	// anything bypassing go-atlassian has to append it here. Leaving it off
	// returns 404 on every endpoint.
	ConfluenceRESTBase string
	HTTPClient         *http.Client
}

// NewClients loads the auth config, resolves the given site (or the default),
// and returns fully authenticated API clients ready for use.
func NewClients(site string) (*Clients, error) {
	cfg, err := config.LoadAuthConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot load auth config: %w", err)
	}

	if site == "" {
		site = cfg.DefaultSite
	}
	if site == "" {
		return nil, &AuthRequiredError{Message: "not authenticated; run: atlassian-cloud auth login"}
	}

	siteAuth, ok := cfg.Sites[site]
	if !ok {
		return nil, &AuthRequiredError{Message: fmt.Sprintf("no credentials for site %q; run: atlassian-cloud auth login", site)}
	}

	switch siteAuth.Method {
	case config.AuthMethodOAuth2:
		return newOAuth2Clients(cfg, site, &siteAuth)
	case config.AuthMethodToken:
		return newTokenClients(site, &siteAuth)
	default:
		return nil, fmt.Errorf("unknown auth method %q for site %q", siteAuth.Method, site)
	}
}

func newOAuth2Clients(cfg *config.AuthConfig, site string, siteAuth *config.SiteAuth) (*Clients, error) {
	accessToken, err := refreshTokenIfNeeded(cfg, site, siteAuth)
	if err != nil {
		return nil, err
	}

	cloudID := siteAuth.CloudID
	jiraURL := fmt.Sprintf("https://api.atlassian.com/ex/jira/%s", cloudID)
	confluenceURL := fmt.Sprintf("https://api.atlassian.com/ex/confluence/%s", cloudID)

	httpClient := &http.Client{
		Transport: &bearerTransport{host: "api.atlassian.com", token: accessToken},
	}

	return buildClients(jiraURL, confluenceURL, func(a common.Authentication) {
		a.SetBearerToken(accessToken)
	}, httpClient)
}

func newTokenClients(site string, siteAuth *config.SiteAuth) (*Clients, error) {
	siteURL := fmt.Sprintf("https://%s", site)

	httpClient := &http.Client{
		Transport: &basicAuthTransport{host: site, email: siteAuth.Email, token: siteAuth.APIToken},
	}

	return buildClients(siteURL, siteURL, func(a common.Authentication) {
		a.SetBasicAuth(siteAuth.Email, siteAuth.APIToken)
	}, httpClient)
}

func buildClients(jiraURL, confluenceURL string, configureAuth func(common.Authentication), httpClient *http.Client) (*Clients, error) {
	jiraClient, err := jira.New(http.DefaultClient, jiraURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Jira client: %w", err)
	}
	configureAuth(jiraClient.Auth)

	confV1, err := confluence.New(http.DefaultClient, confluenceURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v1 client: %w", err)
	}
	configureAuth(confV1.Auth)

	confV2, err := confluencev2.New(http.DefaultClient, confluenceURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v2 client: %w", err)
	}
	configureAuth(confV2.Auth)

	return &Clients{
		Jira:               jiraClient,
		ConfluenceV1:       confV1,
		ConfluenceV2:       confV2,
		JiraBaseURL:        jiraURL,
		ConfluenceBaseURL:  confluenceURL,
		ConfluenceRESTBase: strings.TrimSuffix(confluenceURL, "/") + "/wiki",
		HTTPClient:         httpClient,
	}, nil
}

type bearerTransport struct {
	host  string
	token string
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	// Only attach credentials for the API host; attachment downloads redirect
	// to a media CDN on a different host that must not receive our token.
	if req.URL.Host == t.host {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return http.DefaultTransport.RoundTrip(req)
}

type basicAuthTransport struct {
	host  string
	email string
	token string
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	// Only attach credentials for the API host; attachment downloads redirect
	// to a media CDN on a different host that must not receive our credentials.
	if req.URL.Host == t.host {
		req.SetBasicAuth(t.email, t.token)
	}
	return http.DefaultTransport.RoundTrip(req)
}

func refreshTokenIfNeeded(cfg *config.AuthConfig, site string, siteAuth *config.SiteAuth) (string, error) {
	if siteAuth.TokenExpiry == "" {
		return siteAuth.AccessToken, nil
	}

	expiry, err := siteAuth.ExpiryTime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cannot parse token expiry %q, attempting refresh\n", siteAuth.TokenExpiry)
	} else if !time.Now().Add(tokenExpiryBuffer).After(expiry) {
		return siteAuth.AccessToken, nil
	}

	oauthCfg := LoadOAuthConfigFromEnv()
	if oauthCfg == nil {
		return "", fmt.Errorf("ATLASSIAN_CLIENT_ID and ATLASSIAN_CLIENT_SECRET required for token refresh")
	}

	token, err := RefreshAccessToken(context.Background(), oauthCfg, siteAuth.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("cannot refresh token: %w", err)
	}

	siteAuth.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		siteAuth.RefreshToken = token.RefreshToken
	}
	siteAuth.TokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Format(time.RFC3339)
	cfg.Sites[site] = *siteAuth
	if saveErr := config.SaveAuthConfig(cfg); saveErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not persist refreshed token: %v\n", saveErr)
	}

	return token.AccessToken, nil
}
