package plugins

import (
	"context"
	"fmt"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/verify"
)

// githubAPIBase is the GitHub REST API root. Overridden in tests.
var githubAPIBase = "https://api.github.com"

type githubRelease struct {
	Assets []struct {
		Name               string `json:"name"`
		Size               int64  `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// githubSource resolves the first JAR asset of a repository's latest release.
// Releases commonly ship checksums, signatures, and source archives alongside
// the plugin, so the asset list is filtered by extension rather than taking
// whatever happens to be listed first.
//
// GitHub publishes no digest for release assets, so the asset size is the only
// integrity metadata available — the same weak check the Containerfile makes.
func githubSource(ctx context.Context, owner, repo string) (source, error) {
	var release githubRelease
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPIBase, owner, repo)
	if err := getJSON(ctx, url, &release); err != nil {
		return source{}, fmt.Errorf("fetching GitHub release for %s/%s: %w", owner, repo, err)
	}

	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".jar") && asset.BrowserDownloadURL != "" {
			return source{
				url:      asset.BrowserDownloadURL,
				expected: verify.Expected{Size: asset.Size},
			}, nil
		}
	}

	return source{}, fmt.Errorf("no JAR asset found for %s/%s latest release", owner, repo)
}
