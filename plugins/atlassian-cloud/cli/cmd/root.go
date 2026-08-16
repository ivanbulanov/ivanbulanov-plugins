package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
)

var siteName string

var rootCmd = &cobra.Command{
	Use:   "atlassian-cloud",
	Short: "Context-efficient Jira and Confluence Cloud CLI",
	Long:  "Access Jira issues, Confluence pages, and more with progressive disclosure and OAuth2 authentication.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&siteName, "site", "", "Atlassian site (overrides default)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var authErr *auth.AuthRequiredError
		if errors.As(err, &authErr) {
			os.Exit(auth.ExitCodeAuthRequired)
		}
		var refusedErr *refusedError
		if errors.As(err, &refusedErr) {
			os.Exit(ExitCodeRefused)
		}
		os.Exit(1)
	}
}

func resolveSite(refSite string) string {
	if siteName != "" {
		return siteName
	}
	return refSite
}
