package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "atlassian-cloud",
	Short: "Context-efficient Jira and Confluence Cloud CLI",
	Long:  "Access Jira issues, Confluence pages, and more with progressive disclosure and OAuth2 authentication.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
