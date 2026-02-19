package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/download"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var jiraAttachmentCmd = &cobra.Command{
	Use:   "attachment",
	Short: "Attachment operations",
}

var jiraAttachmentDownloadCmd = &cobra.Command{
	Use:   "download <issue-key-or-url> [filename]",
	Short: "Download attachments from a Jira issue",
	Long:  "Download a single attachment by filename, or all attachments with --all. Files are saved to a temp directory by default; use --output-dir to specify a target directory.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runJiraAttachmentDownload,
}

var (
	jiraDownloadAll       bool
	jiraDownloadOutputDir string
)

func init() {
	jiraCmd.AddCommand(jiraAttachmentCmd)
	jiraAttachmentCmd.AddCommand(jiraAttachmentDownloadCmd)

	jiraAttachmentDownloadCmd.Flags().BoolVar(&jiraDownloadAll, "all", false, "Download all attachments")
	jiraAttachmentDownloadCmd.Flags().StringVar(&jiraDownloadOutputDir, "output-dir", "", "Directory to save files (default: OS temp dir)")
}

func runJiraAttachmentDownload(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseJiraRef(args[0])
	if !ok {
		return fmt.Errorf("invalid issue reference: %s", args[0])
	}

	var filename string
	if len(args) > 1 {
		filename = args[1]
	}

	if filename == "" && !jiraDownloadAll {
		return fmt.Errorf("specify a filename or use --all to download all attachments")
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	// Fetch issue with attachment field
	fields := []string{"attachment"}
	issue, response, err := clients.Jira.Issue.Get(context.Background(), ref.IssueKey, fields, nil)
	if err != nil {
		if response != nil && response.Code == 404 {
			return fmt.Errorf("issue %s not found", ref.IssueKey)
		}
		code := 0
		if response != nil {
			code = response.Code
		}
		return auth.WrapAPIError(code, fmt.Errorf("cannot get issue: %w", err))
	}
	_ = issue // metadata used via raw JSON

	attachments := ExtractAttachments(response.Bytes.Bytes())
	if len(attachments) == 0 {
		return fmt.Errorf("no attachments found on %s", ref.IssueKey)
	}

	// Filter attachments
	type dlTarget struct {
		filename string
		url      string
	}
	var targets []dlTarget

	if jiraDownloadAll {
		for _, a := range attachments {
			targets = append(targets, dlTarget{filename: a.Filename, url: a.Content})
		}
	} else {
		found := false
		for _, a := range attachments {
			if a.Filename == filename {
				targets = append(targets, dlTarget{filename: a.Filename, url: a.Content})
				found = true
				break
			}
		}
		if !found {
			var names []string
			for _, a := range attachments {
				names = append(names, a.Filename)
			}
			return fmt.Errorf("attachment %q not found on %s; available: %v", filename, ref.IssueKey, names)
		}
	}

	// Determine output directory
	outputDir := jiraDownloadOutputDir
	if outputDir == "" {
		outputDir, err = os.MkdirTemp("", "jira-attachments-*")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
	} else {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
	}

	// Download each attachment
	ctx := context.Background()
	for _, t := range targets {
		destPath := filepath.Join(outputDir, t.filename)
		if err := download.ToFile(ctx, clients.HTTPClient, t.url, destPath); err != nil {
			if authErr, ok := err.(*download.AuthError); ok {
				return &auth.AuthRequiredError{Message: authErr.Message}
			}
			return fmt.Errorf("download %s: %w", t.filename, err)
		}
		fmt.Println(destPath)
	}

	return nil
}
