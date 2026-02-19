package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/output"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var jiraCmd = &cobra.Command{
	Use:   "jira",
	Short: "Jira operations",
}

var jiraIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue operations",
}

var jiraIssueGetCmd = &cobra.Command{
	Use:   "get <issue-key-or-url>",
	Short: "Get issue details",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraIssueGet,
}

var jiraSearchCmd = &cobra.Command{
	Use:   "search <jql>",
	Short: "Search issues with JQL",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraSearch,
}

var (
	issueDescription  bool
	issueComments     bool
	issueAttachments  bool
	issueAllFields    bool
	issueFields       string
	searchMax         int
	searchDescription bool
)

func init() {
	rootCmd.AddCommand(jiraCmd)
	jiraCmd.AddCommand(jiraIssueCmd)
	jiraCmd.AddCommand(jiraSearchCmd)
	jiraIssueCmd.AddCommand(jiraIssueGetCmd)

	jiraIssueGetCmd.Flags().BoolVar(&issueDescription, "description", false, "Include description")
	jiraIssueGetCmd.Flags().BoolVar(&issueComments, "comments", false, "Include comments")
	jiraIssueGetCmd.Flags().BoolVar(&issueAttachments, "attachments", false, "Include attachments")
	jiraIssueGetCmd.Flags().BoolVar(&issueAllFields, "all-fields", false, "Include all fields")
	jiraIssueGetCmd.Flags().StringVar(&issueFields, "fields", "", "Comma-separated list of fields")

	jiraSearchCmd.Flags().IntVar(&searchMax, "max", 20, "Maximum results")
	jiraSearchCmd.Flags().BoolVar(&searchDescription, "description", false, "Include descriptions in results")
}

func runJiraIssueGet(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	fields := []string{"summary", "status", "issuetype", "priority", "assignee", "reporter", "project", "labels", "created", "updated"}
	if issueDescription || issueAllFields {
		fields = append(fields, "description")
	}
	if issueComments || issueAllFields {
		fields = append(fields, "comment")
	}
	if issueAttachments || issueAllFields {
		fields = append(fields, "attachment")
	}
	if issueFields != "" {
		for _, f := range strings.Split(issueFields, ",") {
			fields = append(fields, strings.TrimSpace(f))
		}
	}

	issue, response, err := clients.Jira.Issue.Get(context.Background(), ref.IssueKey, fields, nil)
	if err != nil {
		if response != nil && response.Code == 404 {
			return fmt.Errorf("issue %s not found", ref.IssueKey)
		}
		if response != nil && response.Code == 401 {
			return &auth.AuthRequiredError{Message: "authentication failed; run: atlassian-cloud auth login"}
		}
		return fmt.Errorf("cannot get issue: %w", err)
	}

	fmt.Print(output.FormatIssueSummary(issue))

	if (issueDescription || issueAllFields) && issue.Fields.Description != "" {
		fmt.Print(output.FormatIssueDescription(issue.Fields.Description))
	}

	if (issueComments || issueAllFields) && issue.Fields.Comment != nil {
		fmt.Print(output.FormatComments(issue.Fields.Comment.Comments))
	}

	// TODO: Jira V2 model (IssueFieldsSchemeV2) lacks an Attachment field.
	// Attachments require either the V3 client or custom JSON parsing.
	// The "attachment" field is requested from the API but not captured by the struct.

	return nil
}

func runJiraSearch(_ *cobra.Command, args []string) error {
	clients, err := auth.NewClients(resolveSite(""))
	if err != nil {
		return err
	}

	fields := []string{"summary", "status", "assignee"}
	if searchDescription {
		fields = append(fields, "description")
	}

	results, _, err := clients.Jira.Issue.Search.SearchJQL(context.Background(), args[0], fields, nil, searchMax, "")
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	fmt.Print(output.FormatSearchResults(results.Issues, results.Total))

	return nil
}
