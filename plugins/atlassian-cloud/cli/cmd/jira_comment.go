package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/output"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var jiraCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Comment operations",
}

var jiraCommentListCmd = &cobra.Command{
	Use:   "list <issue-key-or-url>",
	Short: "List comments on an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraCommentList,
}

var jiraCommentAddCmd = &cobra.Command{
	Use:   "add <issue-key-or-url>",
	Short: "Add a comment to an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runJiraCommentAdd,
}

var jiraCommentUpdateCmd = &cobra.Command{
	Use:   "update <issue-key-or-url> [comment-id]",
	Short: "Update a comment on an issue",
	Long: `Update an existing comment. Provide the comment ID as a second argument,
or use a Jira URL with focusedCommentId query parameter.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runJiraCommentUpdate,
}

var (
	commentBody  string
	commentStdin bool
)

func init() {
	jiraCmd.AddCommand(jiraCommentCmd)
	jiraCommentCmd.AddCommand(jiraCommentListCmd)
	jiraCommentCmd.AddCommand(jiraCommentAddCmd)
	jiraCommentCmd.AddCommand(jiraCommentUpdateCmd)

	jiraCommentAddCmd.Flags().StringVar(&commentBody, "body", "", "Comment text")
	jiraCommentAddCmd.Flags().BoolVar(&commentStdin, "stdin", false, "Read comment body from stdin")

	jiraCommentUpdateCmd.Flags().StringVar(&commentBody, "body", "", "Comment text")
	jiraCommentUpdateCmd.Flags().BoolVar(&commentStdin, "stdin", false, "Read comment body from stdin")
}

func runJiraCommentList(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	comments, response, err := clients.Jira.Issue.Comment.Gets(context.Background(), ref.IssueKey, "", nil, 0, 50)
	if err != nil {
		code := 0
		if response != nil {
			code = response.Code
		}
		return auth.WrapAPIError(code, fmt.Errorf("cannot list comments: %w", err))
	}

	fmt.Print(output.FormatComments(comments.Comments))
	return nil
}

func runJiraCommentAdd(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	body := commentBody
	if commentStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		body = string(data)
	}

	if body == "" {
		return fmt.Errorf("comment body required (use --body or --stdin)")
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	payload := &models.CommentPayloadSchemeV2{Body: body}
	comment, response, err := clients.Jira.Issue.Comment.Add(context.Background(), ref.IssueKey, payload, nil)
	if err != nil {
		code := 0
		if response != nil {
			code = response.Code
		}
		return auth.WrapAPIError(code, fmt.Errorf("cannot add comment: %w", err))
	}

	fmt.Printf("Comment added (ID: %s)\n", comment.ID)
	return nil
}

func runJiraCommentUpdate(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	commentID := ref.CommentID
	if len(args) > 1 {
		commentID = args[1]
	}
	if commentID == "" {
		return fmt.Errorf("comment ID required (provide as second argument or use a URL with focusedCommentId)")
	}

	body := commentBody
	if commentStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		body = string(data)
	}

	if body == "" {
		return fmt.Errorf("comment body required (use --body or --stdin)")
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	payload := &models.CommentPayloadSchemeV2{Body: body}
	comment, response, err := clients.Jira.Issue.Comment.Update(context.Background(), ref.IssueKey, commentID, payload, nil)
	if err != nil {
		code := 0
		if response != nil {
			code = response.Code
		}
		return auth.WrapAPIError(code, fmt.Errorf("cannot update comment: %w", err))
	}

	fmt.Printf("Comment updated (ID: %s)\n", comment.ID)
	return nil
}
