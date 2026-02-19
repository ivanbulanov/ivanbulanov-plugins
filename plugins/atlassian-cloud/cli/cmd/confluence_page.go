package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/output"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

var confluenceCmd = &cobra.Command{
	Use:   "confluence",
	Short: "Confluence operations",
}

var confluencePageCmd = &cobra.Command{
	Use:   "page",
	Short: "Page operations",
}

var confluencePageGetCmd = &cobra.Command{
	Use:   "get <page-id-or-url>",
	Short: "Get page details",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfluencePageGet,
}

var confluenceSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search Confluence pages using CQL",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfluenceSearch,
}

var (
	pageBody        bool
	pageAttachments bool
	searchSpace     string
	searchMaxConf   int
)

func init() {
	rootCmd.AddCommand(confluenceCmd)
	confluenceCmd.AddCommand(confluencePageCmd)
	confluenceCmd.AddCommand(confluenceSearchCmd)
	confluencePageCmd.AddCommand(confluencePageGetCmd)

	confluencePageGetCmd.Flags().BoolVar(&pageBody, "body", false, "Include page body content")
	confluencePageGetCmd.Flags().BoolVar(&pageAttachments, "attachments", false, "Include attachments list")

	confluenceSearchCmd.Flags().StringVar(&searchSpace, "space", "", "Limit search to space key")
	confluenceSearchCmd.Flags().IntVar(&searchMaxConf, "max", 10, "Maximum results")
}

func runConfluencePageGet(_ *cobra.Command, args []string) error {
	ref, ok := urlparse.ParseConfluenceRef(args[0])
	if !ok {
		return fmt.Errorf("invalid page reference: %s", args[0])
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}

	pageID, err := strconv.Atoi(ref.PageID)
	if err != nil {
		return fmt.Errorf("invalid page ID: %s", ref.PageID)
	}

	bodyFormat := ""
	if pageBody {
		bodyFormat = "atlas_doc_format"
	}

	page, _, err := clients.ConfluenceV2.Page.Get(context.Background(), pageID, bodyFormat, false, 0)
	if err != nil {
		return fmt.Errorf("cannot get page: %w", err)
	}

	fmt.Print(output.FormatPageSummary(page))

	if pageBody {
		fmt.Print(output.FormatPageBody(page))
	}

	if pageAttachments {
		attachments, _, err := clients.ConfluenceV2.Attachment.Gets(context.Background(), pageID, "pages", nil, "", 50)
		if err != nil {
			fmt.Printf("\n*Cannot load attachments: %v*\n", err)
		} else {
			fmt.Print(output.FormatConfluenceAttachments(attachments.Results))
		}
	}

	return nil
}

func runConfluenceSearch(_ *cobra.Command, args []string) error {
	clients, err := auth.NewClients(resolveSite(""))
	if err != nil {
		return err
	}

	cql := `type = page`
	if searchSpace != "" {
		cql += fmt.Sprintf(` AND space = "%s"`, escapeCQL(searchSpace))
	}
	cql += fmt.Sprintf(` AND text ~ "%s"`, escapeCQL(args[0]))

	results, _, err := clients.ConfluenceV1.Search.Content(context.Background(), cql, &models.SearchContentOptions{
		Limit: searchMaxConf,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	fmt.Print(output.FormatSearchResultsConfluence(results.Results, results.TotalSize))
	return nil
}

func escapeCQL(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
