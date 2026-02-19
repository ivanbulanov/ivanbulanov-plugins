package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication for Atlassian Cloud",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate via OAuth2 browser flow",
	Long:  "Opens a browser window for Atlassian OAuth2 authorization. Requires ATLASSIAN_CLIENT_ID and ATLASSIAN_CLIENT_SECRET environment variables.",
	RunE:  runAuthLogin,
}

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Authenticate with an API token",
	Long:  "Configure authentication using an Atlassian API token (email + token pair). Use --site to specify the Atlassian site domain (e.g. mycompany.atlassian.net).",
	RunE:  runAuthToken,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE:  runAuthStatus,
}

var (
	tokenEmail string
	tokenToken string
	tokenSite  string
)

func init() {
	authTokenCmd.Flags().StringVar(&tokenEmail, "email", "", "Atlassian account email")
	authTokenCmd.Flags().StringVar(&tokenToken, "token", "", "Atlassian API token (or set ATLASSIAN_API_TOKEN)")
	authTokenCmd.Flags().StringVar(&tokenSite, "site", "", "Atlassian site (e.g. mycompany.atlassian.net)")
	_ = authTokenCmd.MarkFlagRequired("email")
	_ = authTokenCmd.MarkFlagRequired("site")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authTokenCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(_ *cobra.Command, _ []string) error {
	oauthCfg := auth.LoadOAuthConfigFromEnv()
	if oauthCfg == nil {
		return fmt.Errorf("ATLASSIAN_CLIENT_ID and ATLASSIAN_CLIENT_SECRET must be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := auth.RunOAuthFlow(ctx, oauthCfg)
	if err != nil {
		return fmt.Errorf("OAuth2 flow failed: %w", err)
	}

	if len(result.Resources) == 0 {
		return fmt.Errorf("no accessible Atlassian sites found for this account")
	}

	cfg, err := config.LoadAuthConfig()
	if err != nil {
		return fmt.Errorf("cannot load auth config: %w", err)
	}

	for _, resource := range result.Resources {
		cfg.Sites[resource.Name] = config.SiteAuth{
			Method:       config.AuthMethodOAuth2,
			AccessToken:  result.Token.AccessToken,
			RefreshToken: result.Token.RefreshToken,
			TokenExpiry:  time.Now().Add(time.Duration(result.Token.ExpiresIn) * time.Second).Format(time.RFC3339),
			CloudID:      resource.ID,
			Scopes:       oauthCfg.Scopes,
		}
		fmt.Printf("Authenticated: %s (%s)\n", resource.Name, resource.URL)
	}

	if cfg.DefaultSite == "" {
		cfg.DefaultSite = result.Resources[0].Name
		fmt.Printf("Default site: %s\n", cfg.DefaultSite)
	}

	if err := config.SaveAuthConfig(cfg); err != nil {
		return fmt.Errorf("cannot save auth config: %w", err)
	}

	fmt.Println("Authentication saved.")
	return nil
}

func runAuthToken(_ *cobra.Command, _ []string) error {
	token := tokenToken
	if token == "" {
		token = os.Getenv("ATLASSIAN_API_TOKEN")
	}
	if token == "" {
		return fmt.Errorf("API token required: use --token flag or ATLASSIAN_API_TOKEN env var")
	}

	cfg, err := config.LoadAuthConfig()
	if err != nil {
		return fmt.Errorf("cannot load auth config: %w", err)
	}

	cfg.Sites[tokenSite] = config.SiteAuth{
		Method:   config.AuthMethodToken,
		Email:    tokenEmail,
		APIToken: token,
	}

	if cfg.DefaultSite == "" {
		cfg.DefaultSite = tokenSite
	}

	if err := config.SaveAuthConfig(cfg); err != nil {
		return fmt.Errorf("cannot save auth config: %w", err)
	}

	fmt.Printf("Authenticated: %s (API token)\n", tokenSite)
	fmt.Println("Authentication saved.")
	return nil
}

func runAuthStatus(_ *cobra.Command, _ []string) error {
	cfg, err := config.LoadAuthConfig()
	if err != nil {
		return fmt.Errorf("cannot load auth config: %w", err)
	}

	if len(cfg.Sites) == 0 {
		fmt.Println("Not authenticated. Run: atlassian-cloud auth login")
		return nil
	}

	fmt.Printf("Default site: %s\n\n", cfg.DefaultSite)
	for name, site := range cfg.Sites {
		status := "valid"
		if site.Method == config.AuthMethodOAuth2 {
			if expiry, parseErr := site.ExpiryTime(); parseErr == nil && time.Now().After(expiry) {
				status = "expired (will refresh on next use)"
			}
		}
		fmt.Printf("  %s\n", name)
		fmt.Printf("    Method:  %s\n", site.Method)
		fmt.Printf("    Status:  %s\n", status)
		if site.CloudID != "" {
			fmt.Printf("    CloudID: %s\n", site.CloudID)
		}
		if site.Email != "" {
			fmt.Printf("    Email:   %s\n", site.Email)
		}
		fmt.Println()
	}

	return nil
}
