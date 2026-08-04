package plugins

import (
	"context"
	"fmt"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/verify"
)

// hangarAPIBase is the Hangar API root. Overridden in tests.
var hangarAPIBase = "https://hangar.papermc.io/api/v1"

// hangarVersion is the subset of the Hangar version response we need.
type hangarVersion struct {
	Downloads struct {
		Paper struct {
			FileInfo struct {
				Name       string `json:"name"`
				SizeBytes  int64  `json:"sizeBytes"`
				SHA256Hash string `json:"sha256Hash"`
			} `json:"fileInfo"`
		} `json:"PAPER"`
	} `json:"downloads"`
}

// hangarSource resolves the latest release of a Hangar project along with its
// published SHA-256.
func hangarSource(ctx context.Context, project string) (source, error) {
	version, err := hangarLatestVersion(ctx, project)
	if err != nil {
		return source{}, err
	}

	var meta hangarVersion
	versionURL := fmt.Sprintf("%s/projects/%s/versions/%s", hangarAPIBase, project, version)
	if err := getJSON(ctx, versionURL, &meta); err != nil {
		return source{}, fmt.Errorf("fetching Hangar metadata for %s %s: %w", project, version, err)
	}

	return source{
		url:      fmt.Sprintf("%s/projects/%s/versions/%s/PAPER/download", hangarAPIBase, project, version),
		expected: verify.Expected{SHA256: meta.Downloads.Paper.FileInfo.SHA256Hash},
	}, nil
}

// hangarLatestVersion fetches the latest release version string from Hangar.
// The endpoint returns a bare JSON string, so the surrounding quotes have to
// be stripped before the value can be used in a URL path.
func hangarLatestVersion(ctx context.Context, project string) (string, error) {
	body, err := get(ctx, fmt.Sprintf("%s/projects/%s/latestrelease", hangarAPIBase, project))
	if err != nil {
		return "", fmt.Errorf("fetching Hangar version for %s: %w", project, err)
	}

	version := strings.Trim(strings.TrimSpace(string(body)), `"`)
	if version == "" {
		return "", fmt.Errorf("empty version for %s", project)
	}

	return version, nil
}
