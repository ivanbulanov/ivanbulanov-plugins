package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type attachmentResults struct {
	Results []struct {
		Title string `json:"title"`
	} `json:"results"`
}

// UploadAttachment adds a file to a page and returns the stored attachment
// title, which is what an <ac:image> references. Uses the v1 endpoint because
// v2 has no attachment create.
func UploadAttachment(ctx context.Context, httpClient *http.Client, baseURL, pageID, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open attachment: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", fmt.Errorf("create multipart form: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("write attachment body: %w", err)
	}
	if err := writer.WriteField("minorEdit", "true"); err != nil {
		return "", fmt.Errorf("write minorEdit field: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart form: %w", err)
	}

	endpoint := fmt.Sprintf("%s/rest/api/content/%s/child/attachment", strings.TrimSuffix(baseURL, "/"), pageID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Confluence rejects multipart posts without this header.
	req.Header.Set("X-Atlassian-Token", "no-check")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload attachment: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read upload response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("upload %s failed: HTTP %d: %s",
			filepath.Base(filePath), resp.StatusCode, truncate(string(raw), 300))
	}

	var out attachmentResults
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	if len(out.Results) == 0 {
		return "", fmt.Errorf("upload %s returned no attachment", filepath.Base(filePath))
	}
	return out.Results[0].Title, nil
}

// AttachmentTitles lists the filenames already attached to a page, so an
// unchanged diagram is not uploaded twice.
func AttachmentTitles(ctx context.Context, httpClient *http.Client, baseURL, pageID string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/rest/api/content/%s/child/attachment?limit=200",
		strings.TrimSuffix(baseURL, "/"), pageID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create attachment list request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read attachment list: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list attachments failed: HTTP %d", resp.StatusCode)
	}

	var out attachmentResults
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode attachment list: %w", err)
	}

	titles := make([]string, 0, len(out.Results))
	for _, r := range out.Results {
		titles = append(titles, r.Title)
	}
	return titles, nil
}
