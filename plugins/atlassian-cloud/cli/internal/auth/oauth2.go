package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/oauth2"
	"github.com/ctreminiom/go-atlassian/v2/service/common"
)

// CallbackPort is the local port used for the OAuth2 callback server.
const CallbackPort = 19872

// OAuthResult holds the outcome of a successful OAuth2 flow.
type OAuthResult struct {
	Token     *common.OAuth2Token
	Resources []*common.AccessibleResource
}

// LoadOAuthConfigFromEnv reads ATLASSIAN_CLIENT_ID and ATLASSIAN_CLIENT_SECRET
// from environment variables and returns an OAuthConfig suitable for the
// authorization code flow. Returns nil if either variable is missing.
func LoadOAuthConfigFromEnv() *common.OAuth2Config {
	clientID := os.Getenv("ATLASSIAN_CLIENT_ID")
	clientSecret := os.Getenv("ATLASSIAN_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil
	}
	return &common.OAuth2Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  fmt.Sprintf("http://localhost:%d/callback", CallbackPort),
		Scopes: []string{
			"read:jira-work", "write:jira-work", "read:jira-user",
			"read:confluence-content.all", "read:confluence-space.summary",
			// Publishing a page needs write:confluence-content, and uploading
			// its diagrams needs write:confluence-file. Anyone who authorised
			// before these were added holds a token without them and has to
			// run `auth login` again; the API answers 403 until they do.
			"write:confluence-content", "write:confluence-file",
			"offline_access",
		},
	}
}

// RunOAuthFlow starts a local HTTP server, opens the browser for Atlassian
// authorization, waits for the callback, exchanges the code for tokens, and
// retrieves accessible resources.
func RunOAuthFlow(ctx context.Context, cfg *common.OAuth2Config) (*OAuthResult, error) {
	svc, err := oauth2.NewOAuth2Service(http.DefaultClient, cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create OAuth2 service: %w", err)
	}

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("cannot generate OAuth2 state: %w", err)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errCh <- fmt.Errorf("OAuth error: state mismatch (possible CSRF)")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			fmt.Fprintf(w, "<html><body><h2>Authorization failed</h2><p>%s</p></body></html>", html.EscapeString(errMsg))
			errCh <- fmt.Errorf("OAuth error: %s", errMsg)
			return
		}
		fmt.Fprint(w, "<html><body><h2>Authorization successful!</h2><p>You can close this window.</p></body></html>")
		codeCh <- code
	})

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", CallbackPort))
	if err != nil {
		return nil, fmt.Errorf("cannot start callback server on port %d: %w", CallbackPort, err)
	}

	server := &http.Server{Handler: mux}
	go func() {
		if serveErr := server.Serve(listener); serveErr != http.ErrServerClosed {
			errCh <- serveErr
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	authURL, err := svc.GetAuthorizationURL(cfg.Scopes, state)
	if err != nil {
		return nil, fmt.Errorf("cannot build authorization URL: %w", err)
	}

	fmt.Println("Opening browser for authorization...")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", authURL.String())
	openBrowser(authURL.String())

	select {
	case code := <-codeCh:
		return exchangeAndFetch(ctx, svc, code)
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// RefreshAccessToken uses an existing refresh token to obtain a new access token.
func RefreshAccessToken(ctx context.Context, cfg *common.OAuth2Config, refreshToken string) (*common.OAuth2Token, error) {
	svc, err := oauth2.NewOAuth2Service(http.DefaultClient, cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create OAuth2 service: %w", err)
	}
	token, err := svc.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	return token, nil
}

func exchangeAndFetch(ctx context.Context, svc *oauth2.Service, code string) (*OAuthResult, error) {
	token, err := svc.ExchangeAuthorizationCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	resources, err := svc.GetAccessibleResources(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("cannot get accessible resources: %w", err)
	}

	return &OAuthResult{
		Token:     token,
		Resources: resources,
	}, nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
