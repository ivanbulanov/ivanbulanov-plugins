package urlparse

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	issueKeyRe      = regexp.MustCompile(`^[A-Z][A-Z0-9]+-\d+$`)
	jiraBrowseRe    = regexp.MustCompile(`/browse/([A-Z][A-Z0-9]+-\d+)`)
	confluencePageRe = regexp.MustCompile(`/wiki/spaces/[^/]+/pages/(\d+)`)
	numericRe       = regexp.MustCompile(`^\d+$`)
)

type JiraRef struct {
	IssueKey string
	Site     string
}

type ConfluenceRef struct {
	PageID string
	Site   string
	Space  string
}

func ParseJiraRef(input string) (JiraRef, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return JiraRef{}, false
	}

	// Direct issue key
	if issueKeyRe.MatchString(input) {
		return JiraRef{IssueKey: input}, true
	}

	// URL
	u, err := url.Parse(input)
	if err != nil || u.Host == "" {
		return JiraRef{}, false
	}

	matches := jiraBrowseRe.FindStringSubmatch(u.Path)
	if len(matches) < 2 {
		return JiraRef{}, false
	}

	return JiraRef{
		IssueKey: matches[1],
		Site:     u.Host,
	}, true
}

func ParseConfluenceRef(input string) (ConfluenceRef, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return ConfluenceRef{}, false
	}

	// Numeric page ID
	if numericRe.MatchString(input) {
		return ConfluenceRef{PageID: input}, true
	}

	// URL
	u, err := url.Parse(input)
	if err != nil || u.Host == "" {
		return ConfluenceRef{}, false
	}

	matches := confluencePageRe.FindStringSubmatch(u.Path)
	if len(matches) < 2 {
		return ConfluenceRef{}, false
	}

	// Extract space key from path: /wiki/spaces/{SPACE}/pages/...
	parts := strings.Split(u.Path, "/")
	var space string
	for i, p := range parts {
		if p == "spaces" && i+1 < len(parts) {
			space = parts[i+1]
			break
		}
	}

	return ConfluenceRef{
		PageID: matches[1],
		Site:   u.Host,
		Space:  space,
	}, true
}
