package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
)

var jiraFieldsCmd = &cobra.Command{
	Use:   "fields",
	Short: "Field operations",
}

var jiraFieldsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available fields",
	RunE:  runJiraFieldsList,
}

func init() {
	jiraCmd.AddCommand(jiraFieldsCmd)
	jiraFieldsCmd.AddCommand(jiraFieldsListCmd)
}

func runJiraFieldsList(_ *cobra.Command, _ []string) error {
	clients, err := auth.NewClients(siteName)
	if err != nil {
		return err
	}

	ctx := context.Background()
	fields, _, err := clients.Jira.Issue.Field.Gets(ctx)
	if err != nil {
		return fmt.Errorf("cannot list fields: %w", err)
	}

	fmt.Printf("| %-30s | %-25s | %-10s |\n", "ID", "Name", "Custom")
	fmt.Printf("|%s|%s|%s|\n", "--------------------------------", "---------------------------", "------------")

	for _, f := range fields {
		fmt.Printf("| %-30s | %-25s | %-10v |\n", f.ID, f.Name, f.Custom)
	}

	return nil
}
