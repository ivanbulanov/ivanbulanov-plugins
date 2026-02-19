package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

// AuthError indicates an authentication failure during download.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// ToFile downloads the content at url using httpClient and writes it to destPath.
// The parent directory of destPath must exist.
// Returns an *AuthError on 401 responses, or a generic error on other non-2xx status codes.
func ToFile(ctx context.Context, httpClient *http.Client, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return &AuthError{Message: "authentication failed; run: atlassian-cloud auth login"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
