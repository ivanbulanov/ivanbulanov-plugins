package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// SpaceKey resolves a numeric space id to the key that appears in page URLs.
// A URL built from the numeric id does not resolve.
func SpaceKey(ctx context.Context, httpClient *http.Client, baseURL, spaceID string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v2/spaces/%s", strings.TrimSuffix(baseURL, "/"), spaceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create space request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get space: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read space response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("get space %s failed: HTTP %d", spaceID, resp.StatusCode)
	}

	var out struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("decode space response: %w", err)
	}
	if out.Key == "" {
		return "", fmt.Errorf("space %s has no key", spaceID)
	}
	return out.Key, nil
}
