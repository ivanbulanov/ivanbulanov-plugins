// Package confluence holds the Confluence Cloud REST calls that go-atlassian
// does not provide: body conversion and attachment upload.
package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Converter calls Confluence's own body conversion endpoint. Using Atlassian's
// converter rather than a local Markdown-to-storage library is what makes the
// derived anchors match what the renderer produces.
type Converter struct {
	HTTP    *http.Client
	BaseURL string
}

// NewConverter returns a Converter for one authenticated site. baseURL is the
// Confluence base, for example https://example.atlassian.net/wiki.
func NewConverter(httpClient *http.Client, baseURL string) *Converter {
	return &Converter{HTTP: httpClient, BaseURL: strings.TrimSuffix(baseURL, "/")}
}

type convertRequest struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

type convertResponse struct {
	Value string `json:"value"`
}

// MarkdownToStorage converts Markdown to Confluence storage format.
func (c *Converter) MarkdownToStorage(ctx context.Context, markdown string) (string, error) {
	out, err := c.convert(ctx, "storage", convertRequest{Value: markdown, Representation: "markdown"}, "")
	if err != nil {
		return "", err
	}
	return out, nil
}

// StorageToADF converts storage format to ADF. pageIDContext may be empty; set
// it to the target page id so that ri:attachment references resolve.
func (c *Converter) StorageToADF(ctx context.Context, storage, pageIDContext string) ([]byte, error) {
	out, err := c.convert(ctx, "atlas_doc_format", convertRequest{Value: storage, Representation: "storage"}, pageIDContext)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

func (c *Converter) convert(ctx context.Context, to string, payload convertRequest, pageIDContext string) (string, error) {
	endpoint := fmt.Sprintf("%s/rest/api/contentbody/convert/%s", c.BaseURL, to)
	if pageIDContext != "" {
		endpoint += "?" + url.Values{"contentIdContext": {pageIDContext}}.Encode()
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode convert request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create convert request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("convert to %s: %w", to, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read convert response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("convert to %s failed: HTTP %d: %s", to, resp.StatusCode, truncate(string(raw), 300))
	}

	var out convertResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode convert response: %w", err)
	}
	return out.Value, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
