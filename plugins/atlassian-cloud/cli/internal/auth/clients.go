package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

// Clients bundles authenticated Jira and Confluence API clients for a single site.
type Clients struct {
	Jira         *jira.Client
	ConfluenceV1 *confluence.Client
	ConfluenceV2 *confluencev2.Client
	SiteURL      string
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
		fmt.Fprintln(os.Stderr, "Not authenticated. Run: atlassian-cloud auth login")
		os.Exit(ExitCodeAuthRequired)
	}

	siteAuth, ok := cfg.Sites[site]
	if !ok {
		fmt.Fprintf(os.Stderr, "No credentials for site %q. Run: atlassian-cloud auth login\n", site)
		os.Exit(ExitCodeAuthRequired)
	}

	switch siteAuth.Method {
	case "oauth2":
		return newOAuth2Clients(cfg, site, &siteAuth)
	case "token":
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

	return buildClients(jiraURL, confluenceURL, func(a common.Authentication) {
		a.SetBearerToken(accessToken)
	})
}

func newTokenClients(site string, siteAuth *config.SiteAuth) (*Clients, error) {
	siteURL := fmt.Sprintf("https://%s", site)
	return buildClients(siteURL, siteURL, func(a common.Authentication) {
		a.SetBasicAuth(siteAuth.Email, siteAuth.APIToken)
	})
}

func buildClients(jiraURL, confluenceURL string, configureAuth func(common.Authentication)) (*Clients, error) {
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
		Jira:         jiraClient,
		ConfluenceV1: confV1,
		ConfluenceV2: confV2,
		SiteURL:      jiraURL,
	}, nil
}

func refreshTokenIfNeeded(cfg *config.AuthConfig, site string, siteAuth *config.SiteAuth) (string, error) {
	if siteAuth.TokenExpiry == "" {
		return siteAuth.AccessToken, nil
	}

	expiry, err := time.Parse(time.RFC3339, siteAuth.TokenExpiry)
	if err != nil || !time.Now().Add(tokenExpiryBuffer).After(expiry) {
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
