package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type githubRelease struct {
	Assets []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// githubLatestAssetURL returns the download URL for the first JAR asset of the
// latest release. Releases commonly ship checksums, signatures, and source
// archives alongside the plugin, so the asset list is filtered by extension
// rather than taking whatever happens to be listed first.
func githubLatestAssetURL(ctx context.Context, owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching GitHub release for %s/%s: %w", owner, repo, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d from GitHub for %s/%s", resp.StatusCode, owner, repo)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("parsing GitHub release: %w", err)
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".jar") && asset.BrowserDownloadURL != "" {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no JAR asset found for %s/%s latest release", owner, repo)
}
