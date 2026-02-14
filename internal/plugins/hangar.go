package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// hangarLatestVersion fetches the latest release version string from Hangar.
func hangarLatestVersion(ctx context.Context, project string) (string, error) {
	url := fmt.Sprintf("https://hangar.papermc.io/api/v1/projects/%s/latestrelease", project)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching Hangar version for %s: %w", project, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from Hangar for %s", resp.StatusCode, project)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return "", fmt.Errorf("empty version for %s", project)
	}

	return version, nil
}
