package filterdat

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func LoadSource(ctx context.Context, source string) ([]byte, error) {
	if isHTTPURL(source) {
		return loadURL(ctx, source)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", source, err)
	}
	return data, nil
}

func isHTTPURL(source string) bool {
	u, err := url.Parse(source)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

func loadURL(ctx context.Context, source string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %q: %w", source, err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %q: %w", source, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download %q: unexpected status %s", source, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response from %q: %w", source, err)
	}
	return data, nil
}
