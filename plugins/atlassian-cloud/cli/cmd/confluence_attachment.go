package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/download"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var confluenceAttachmentCmd = &cobra.Command{
	Use:   "attachment",
	Short: "Attachment operations",
}

var confluenceAttachmentDownloadCmd = &cobra.Command{
	Use:   "download <page-id-or-url> [filename]",
	Short: "Download attachments from a Confluence page",
	Long:  "Download a single attachment by filename, or all attachments with --all. Files are saved to a temp directory by default; use --output-dir to specify a target directory.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runConfluenceAttachmentDownload,
}

var (
	confDownloadAll       bool
	confDownloadOutputDir string
)

func init() {
	confluenceCmd.AddCommand(confluenceAttachmentCmd)
	confluenceAttachmentCmd.AddCommand(confluenceAttachmentDownloadCmd)

	confluenceAttachmentDownloadCmd.Flags().BoolVar(&confDownloadAll, "all", false, "Download all attachments")
	confluenceAttachmentDownloadCmd.Flags().StringVar(&confDownloadOutputDir, "output-dir", "", "Directory to save files (default: OS temp dir)")
}

func runConfluenceAttachmentDownload(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseConfluenceRef(args[0])
	if !ok {
		return fmt.Errorf("invalid page reference: %s", args[0])
	}

	var filename string
	if len(args) > 1 {
		filename = args[1]
	}

	if filename == "" && !confDownloadAll {
		return fmt.Errorf("specify a filename or use --all to download all attachments")
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	pageID, err := strconv.Atoi(ref.PageID)
	if err != nil {
		return fmt.Errorf("invalid page ID: %s", ref.PageID)
	}

	attachments, _, err := clients.ConfluenceV2.Attachment.Gets(context.Background(), pageID, "pages", nil, "", 50)
	if err != nil {
		return fmt.Errorf("cannot get attachments: %w", err)
	}

	if attachments == nil || len(attachments.Results) == 0 {
		return fmt.Errorf("no attachments found on page %d", pageID)
	}

	// Filter attachments
	type dlTarget struct {
		filename string
		url      string
	}
	var targets []dlTarget

	if confDownloadAll {
		for _, a := range attachments.Results {
			dlURL := clients.ConfluenceBaseURL + a.DownloadLink
			targets = append(targets, dlTarget{filename: a.Title, url: dlURL})
		}
	} else {
		found := false
		for _, a := range attachments.Results {
			if a.Title == filename {
				dlURL := clients.ConfluenceBaseURL + a.DownloadLink
				targets = append(targets, dlTarget{filename: a.Title, url: dlURL})
				found = true
				break
			}
		}
		if !found {
			var names []string
			for _, a := range attachments.Results {
				names = append(names, a.Title)
			}
			return fmt.Errorf("attachment %q not found on page %d; available: %v", filename, pageID, names)
		}
	}

	// Determine output directory
	outputDir := confDownloadOutputDir
	if outputDir == "" {
		outputDir, err = os.MkdirTemp("", "confluence-attachments-*")
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
