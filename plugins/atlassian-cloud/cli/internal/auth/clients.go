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

	// tokenExpiryBuffer is how far before expiry we proactively refresh.
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
	accessToken := siteAuth.AccessToken

	// Refresh if the token is expired or about to expire.
	if siteAuth.TokenExpiry != "" {
		expiry, err := time.Parse(time.RFC3339, siteAuth.TokenExpiry)
		if err == nil && time.Now().Add(tokenExpiryBuffer).After(expiry) {
			oauthCfg := LoadOAuthConfigFromEnv()
			if oauthCfg == nil {
				return nil, fmt.Errorf("ATLASSIAN_CLIENT_ID and ATLASSIAN_CLIENT_SECRET required for token refresh")
			}
			token, err := RefreshAccessToken(context.Background(), oauthCfg, siteAuth.RefreshToken)
			if err != nil {
				return nil, fmt.Errorf("cannot refresh token: %w", err)
			}
			accessToken = token.AccessToken

			// Persist the refreshed token.
			siteAuth.AccessToken = token.AccessToken
			if token.RefreshToken != "" {
				siteAuth.RefreshToken = token.RefreshToken
			}
			siteAuth.TokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second).Format(time.RFC3339)
			cfg.Sites[site] = *siteAuth
			if saveErr := config.SaveAuthConfig(cfg); saveErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not persist refreshed token: %v\n", saveErr)
			}
		}
	}

	cloudID := siteAuth.CloudID
	jiraSiteURL := fmt.Sprintf("https://api.atlassian.com/ex/jira/%s", cloudID)
	confluenceSiteURL := fmt.Sprintf("https://api.atlassian.com/ex/confluence/%s", cloudID)

	setBearerToken := func(auth common.Authentication) {
		auth.SetBearerToken(accessToken)
	}

	jiraClient, err := jira.New(http.DefaultClient, jiraSiteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Jira client: %w", err)
	}
	setBearerToken(jiraClient.Auth)

	confV1, err := confluence.New(http.DefaultClient, confluenceSiteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v1 client: %w", err)
	}
	setBearerToken(confV1.Auth)

	confV2, err := confluencev2.New(http.DefaultClient, confluenceSiteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v2 client: %w", err)
	}
	setBearerToken(confV2.Auth)

	return &Clients{
		Jira:         jiraClient,
		ConfluenceV1: confV1,
		ConfluenceV2: confV2,
		SiteURL:      jiraSiteURL,
	}, nil
}

func newTokenClients(site string, siteAuth *config.SiteAuth) (*Clients, error) {
	siteURL := fmt.Sprintf("https://%s", site)

	setBasicAuth := func(auth common.Authentication) {
		auth.SetBasicAuth(siteAuth.Email, siteAuth.APIToken)
	}

	jiraClient, err := jira.New(http.DefaultClient, siteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Jira client: %w", err)
	}
	setBasicAuth(jiraClient.Auth)

	confV1, err := confluence.New(http.DefaultClient, siteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v1 client: %w", err)
	}
	setBasicAuth(confV1.Auth)

	confV2, err := confluencev2.New(http.DefaultClient, siteURL)
	if err != nil {
		return nil, fmt.Errorf("cannot create Confluence v2 client: %w", err)
	}
	setBasicAuth(confV2.Auth)

	return &Clients{
		Jira:         jiraClient,
		ConfluenceV1: confV1,
		ConfluenceV2: confV2,
		SiteURL:      siteURL,
	}, nil
}

