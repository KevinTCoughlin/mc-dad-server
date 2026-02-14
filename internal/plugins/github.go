package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type githubRelease struct {
	Assets []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// githubLatestAssetURL returns the download URL for the first asset of the latest release.
func githubLatestAssetURL(ctx context.Context, owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching GitHub release for %s/%s: %w", owner, repo, err)
	}
	defer resp.Body.Close()

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

	if len(release.Assets) == 0 {
		return "", fmt.Errorf("no assets found for %s/%s latest release", owner, repo)
	}

	return release.Assets[0].BrowserDownloadURL, nil
}
