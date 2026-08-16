package mdpublish

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestPublishScratchPage runs the real command against a scratch page. It writes
// to a live wiki, so it is skipped unless both variables below are set. The
// document and the target page are never named in this repository:
//
//	CONFLUENCE_TEST_SOURCE=/path/to/doc.md \
//	CONFLUENCE_TEST_PAGE=<page id or url> \
//	go test ./internal/mdpublish/ -run TestPublishScratchPage -v
//
// Diagrams and the generated storage go to a temporary directory, so a run
// leaves nothing beside a source document the tool does not own.
func TestPublishScratchPage(t *testing.T) {
	source := os.Getenv("CONFLUENCE_TEST_SOURCE")
	page := os.Getenv("CONFLUENCE_TEST_PAGE")
	if source == "" || page == "" {
		t.Skip("set CONFLUENCE_TEST_SOURCE and CONFLUENCE_TEST_PAGE to run")
	}
	if _, err := os.Stat(source); err != nil {
		t.Skipf("source not readable: %v", err)
	}

	cmd := exec.Command("../../bin/atlassian-cloud", "confluence", "publish", source,
		"--page", page, "--force", "--assets-dir", t.TempDir())
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	text := string(out)
	if !strings.Contains(text, "published version") {
		t.Errorf("no publish reported")
	}
	if !strings.Contains(text, ", 0 broken") {
		t.Errorf("verification did not report zero broken links")
	}
}
