package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/spf13/cobra"

	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/auth"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/confluence"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/mdpublish"
	"github.com/ivanbulanov/ivanbulanov-plugins/plugins/atlassian-cloud/cli/internal/urlparse"
)

// ExitCodeRefused is returned when the run stopped before writing anything.
const ExitCodeRefused = 3

// ExitCodeUnverified is returned when the page was published but a check failed.
const ExitCodeUnverified = 4

var confluencePublishCmd = &cobra.Command{
	Use:   "publish <file.md>",
	Short: "Publish a Markdown document to a Confluence page",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfluencePublish,
}

var (
	publishPage      string
	publishTitle     string
	publishAssetsDir string
	publishLinkRefs  string
	publishDryRun    bool
	publishForce     bool
	publishNoTOC     bool
)

func init() {
	confluenceCmd.AddCommand(confluencePublishCmd)
	confluencePublishCmd.Flags().StringVar(&publishPage, "page", "", "Target page id or URL")
	confluencePublishCmd.Flags().StringVar(&publishTitle, "title", "", "Set the page title")
	confluencePublishCmd.Flags().StringVar(&publishAssetsDir, "assets-dir", "", "Where rendered diagrams go")
	confluencePublishCmd.Flags().StringVar(&publishLinkRefs, "link-refs", "all", "Link cross-references: all or none")
	confluencePublishCmd.Flags().BoolVar(&publishDryRun, "dry-run", false, "Generate and check, publish nothing")
	confluencePublishCmd.Flags().BoolVar(&publishForce, "force", false, "Publish over a page this tool did not last write")
	confluencePublishCmd.Flags().BoolVar(&publishNoTOC, "no-toc", false, "Do not insert a table of contents")
}

func runConfluencePublish(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sourcePath := args[0]

	src, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	cfg, _, err := mdpublish.Scan(src)
	if err != nil {
		return err
	}

	target := publishPage
	if target == "" {
		target = cfg.PageURL
	}
	if target == "" {
		return refused(fmt.Errorf("no target page: pass --page or add a confluence-page comment to the document"))
	}

	ref, ok := urlparse.ParseConfluenceRef(target)
	if !ok {
		return refused(fmt.Errorf("invalid page reference: %s", target))
	}

	clients, err := auth.NewClients(resolveSite(ref.Site))
	if err != nil {
		return err
	}
	pageID, err := strconv.Atoi(ref.PageID)
	if err != nil {
		return refused(fmt.Errorf("invalid page id: %s", ref.PageID))
	}

	page, response, err := clients.ConfluenceV2.Page.Get(ctx, pageID, "atlas_doc_format", false, 0)
	if err != nil {
		code := 0
		if response != nil {
			code = response.Code
		}
		return auth.WrapAPIError(code, fmt.Errorf("cannot read target page: %w", err))
	}
	// Body and Version are pointers on PageScheme and are nil when Confluence
	// returns a page without them; dereferencing blind would panic.
	if page.Body == nil || page.Body.AtlasDocFormat == nil || page.Version == nil {
		return refused(fmt.Errorf("target page %s returned no body or version", ref.PageID))
	}

	assets := publishAssetsDir
	if assets == "" {
		assets = dirOf(sourcePath)
	}

	// A page URL needs the space key; one built from the numeric space id
	// does not resolve.
	spaceKey := ref.Space
	if spaceKey == "" {
		spaceKey, err = confluence.SpaceKey(ctx, clients.HTTPClient, clients.ConfluenceRESTBase, page.SpaceID)
		if err != nil {
			return err
		}
	}

	opts := mdpublish.Options{
		SourcePath: sourcePath,
		PageURL:    pageURLOf(clients.ConfluenceRESTBase, spaceKey, page),
		PageID:     ref.PageID,
		Title:      publishTitle,
		AssetsDir:  assets,
		LinkRefs:   publishLinkRefs != "none",
		DryRun:     publishDryRun,
		Force:      publishForce,
		NoTOC:      publishNoTOC,
	}

	conv := confluence.NewConverter(clients.HTTPClient, clients.ConfluenceRESTBase)
	result, newADF, err := mdpublish.Prepare(ctx, conv, opts)
	if err != nil {
		return refused(err)
	}
	storage := result.Storage

	current := mdpublish.PageState{
		ID:             ref.PageID,
		Title:          page.Title,
		Version:        page.Version.Number,
		VersionMessage: page.Version.Message,
		ADF:            []byte(page.Body.AtlasDocFormat.Value),
	}

	skip, err := mdpublish.ShouldSkip(current, newADF)
	if err != nil {
		return err
	}
	// --force means publish anyway: it is the escape hatch when the page and
	// the document agree but the page still needs rewriting.
	if skip && !publishForce {
		fmt.Println("no change; the page already says this")
		return nil
	}

	if publishDryRun {
		// Into the assets directory, not next to the source: --assets-dir
		// exists so a run can keep every artefact out of a repository the
		// document merely happens to live in.
		out := filepath.Join(assets, filepath.Base(sourcePath)+".storage.xml")
		if err := os.WriteFile(out, []byte(storage), 0o600); err != nil {
			return err
		}
		fmt.Printf("dry run: %d references linked, %d tables hoisted, %d links unwrapped\n",
			result.Linked, result.Hoisted, len(result.Unwrapped))
		fmt.Printf("storage written to %s; nothing was published\n", out)
		return nil
	}

	if !mdpublish.IsOurVersion(current.VersionMessage) && !publishForce {
		return refused(fmt.Errorf(
			"page version %d was not written by this tool (message: %q).\n"+
				"Publishing would overwrite edits made in Confluence. Re-run with --force to proceed",
			current.Version, current.VersionMessage))
	}

	existing, err := confluence.AttachmentTitles(ctx, clients.HTTPClient, clients.ConfluenceRESTBase, ref.PageID)
	if err != nil {
		return err
	}
	uploaded := 0
	for _, path := range result.Assets {
		name := path[strings.LastIndexByte(path, '/')+1:]
		if slices.Contains(existing, name) {
			// The filename carries a hash of the content, so a match means
			// this exact diagram is already attached.
			continue
		}
		if _, err := confluence.UploadAttachment(ctx, clients.HTTPClient,
			clients.ConfluenceRESTBase, ref.PageID, path); err != nil {
			return err
		}
		uploaded++
	}

	title := current.Title
	if publishTitle != "" {
		title = publishTitle
	}

	updated, response, err := clients.ConfluenceV2.Page.Update(ctx, pageID, &models.PageUpdatePayloadScheme{
		ID:     ref.PageID,
		Status: "current",
		Title:  title,
		Body: &models.PageBodyRepresentationScheme{
			Representation: "storage",
			Value:          storage,
		},
		Version: &models.PageUpdatePayloadVersionScheme{
			Number:  current.Version + 1,
			Message: mdpublish.VersionMarker(mdpublish.SourceHash(src)),
		},
	})
	if err != nil {
		code := 0
		if response != nil {
			code = response.Code
		}
		if code == 409 {
			return refused(fmt.Errorf(
				"the page changed while this run was preparing; nothing was written. Re-run to pick up the new version"))
		}
		if code == 403 {
			return refused(fmt.Errorf(
				"not permitted to write this page. If authenticated with OAuth, the token needs Confluence write scope; re-run: atlassian-cloud auth login"))
		}
		return auth.WrapAPIError(code, fmt.Errorf("cannot publish: %w", err))
	}

	fmt.Printf("published version %d: %d references linked, %d tables hoisted, %d attachments uploaded\n",
		updated.Version.Number, result.Linked, result.Hoisted, uploaded)

	published, _, err := clients.ConfluenceV2.Page.Get(ctx, pageID, "atlas_doc_format", false, 0)
	if err != nil {
		return fmt.Errorf("published, but could not re-read the page to verify: %w", err)
	}
	if published.Body == nil || published.Body.AtlasDocFormat == nil {
		return fmt.Errorf("published, but the page returned no body to verify")
	}
	landed := []byte(published.Body.AtlasDocFormat.Value)

	links, broken, err := mdpublish.CheckPublished(landed, ref.PageID)
	if err != nil {
		return err
	}
	landedText, err := mdpublish.Text(landed)
	if err != nil {
		return err
	}
	sentText, err := mdpublish.Text(newADF)
	if err != nil {
		return err
	}

	fmt.Printf("verified: %d same-page links, %d broken\n", links, len(broken))
	if len(broken) > 0 || landedText != sentText {
		fmt.Fprintf(os.Stderr, "VERIFICATION FAILED. Restore version %d from page history if needed.\n", current.Version)
		for _, b := range broken {
			fmt.Fprintf(os.Stderr, "  unresolved fragment: %s\n", b)
		}
		if landedText != sentText {
			fmt.Fprintln(os.Stderr, "  the published text differs from what was sent")
		}
		os.Exit(ExitCodeUnverified)
	}

	return nil
}

// refusedError marks an error that stopped the run before anything was written.
type refusedError struct{ err error }

func (e *refusedError) Error() string { return e.err.Error() }

func refused(err error) error { return &refusedError{err: err} }

func dirOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return "."
}

func pageURLOf(baseURL, spaceKey string, page *models.PageScheme) string {
	return fmt.Sprintf("%s/spaces/%s/pages/%s", strings.TrimSuffix(baseURL, "/"), spaceKey, page.ID)
}
