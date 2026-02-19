package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// siteName is the global --site flag used by all commands to specify the Atlassian site.
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
		os.Exit(1)
	}
}
