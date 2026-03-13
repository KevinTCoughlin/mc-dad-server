package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// defaultPaperAPIBase is the base URL for the PaperMC Fill v3 downloads API.
const defaultPaperAPIBase = "https://fill.papermc.io/v3"

// paperUserAgent identifies this tool to the PaperMC Fill v3 API,
// which requires a descriptive User-Agent header on every request.
const paperUserAgent = "mc-dad-server (https://github.com/KevinTCoughlin/mc-dad-server)"

type paperVersionsResponse struct {
	Versions []struct {
		Version struct {
			ID string `json:"id"`
		} `json:"version"`
	} `json:"versions"`
}

type paperBuildResponse struct {
	Downloads map[string]struct {
		Name      string `json:"name"`
		URL       string `json:"url"`
		Checksums struct {
			SHA256 string `json:"sha256"`
		} `json:"checksums"`
	} `json:"downloads"`
}

// PaperDownloadURL resolves the download URL for a Paper server JAR.
func PaperDownloadURL(ctx context.Context, version string) (string, error) {
	return paperDownloadURL(ctx, version, defaultPaperAPIBase)
}

func paperDownloadURL(ctx context.Context, version, apiBase string) (string, error) {
	if version == "latest" {
		var err error
		version, err = paperLatestVersion(ctx, apiBase)
		if err != nil {
			return "", err
		}
	}

	url := fmt.Sprintf("%s/projects/paper/versions/%s/builds/latest", apiBase, version)
	body, err := httpGet(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetching Paper latest build: %w", err)
	}

	var build paperBuildResponse
	if err := json.Unmarshal(body, &build); err != nil {
		return "", fmt.Errorf("parsing Paper latest build: %w", err)
	}

	dl, ok := build.Downloads["server:default"]
	if !ok || dl.URL == "" {
		return "", fmt.Errorf("no download found for Paper %s latest build", version)
	}

	return dl.URL, nil
}

func paperLatestVersion(ctx context.Context, apiBase string) (string, error) {
	url := fmt.Sprintf("%s/projects/paper/versions", apiBase)
	body, err := httpGet(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetching Paper versions: %w", err)
	}

	var resp paperVersionsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing Paper versions: %w", err)
	}

	if len(resp.Versions) == 0 {
		return "", fmt.Errorf("no Paper versions found")
	}

	// Fill v3 returns versions newest-first.
	return resp.Versions[0].Version.ID, nil
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", paperUserAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
