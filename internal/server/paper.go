package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type paperVersionsResponse struct {
	Versions []string `json:"versions"`
}

type paperBuildsResponse struct {
	Builds []struct {
		Build     int `json:"build"`
		Downloads struct {
			Application struct {
				Name string `json:"name"`
			} `json:"application"`
		} `json:"downloads"`
	} `json:"builds"`
}

// PaperDownloadURL resolves the download URL for a Paper server JAR.
func PaperDownloadURL(ctx context.Context, version string) (string, error) {
	if version == "latest" {
		var err error
		version, err = paperLatestVersion(ctx)
		if err != nil {
			return "", err
		}
	}

	url := fmt.Sprintf("https://api.papermc.io/v2/projects/paper/versions/%s/builds", version)
	body, err := httpGet(ctx, url)
	if err != nil {
		return "", fmt.Errorf("fetching Paper builds: %w", err)
	}

	var builds paperBuildsResponse
	if err := json.Unmarshal(body, &builds); err != nil {
		return "", fmt.Errorf("parsing Paper builds: %w", err)
	}

	if len(builds.Builds) == 0 {
		return "", fmt.Errorf("no builds found for Paper %s", version)
	}

	latest := builds.Builds[len(builds.Builds)-1]
	filename := latest.Downloads.Application.Name
	if filename == "" {
		return "", fmt.Errorf("no download found for Paper %s build %d", version, latest.Build)
	}

	return fmt.Sprintf("https://api.papermc.io/v2/projects/paper/versions/%s/builds/%d/downloads/%s",
		version, latest.Build, filename), nil
}

func paperLatestVersion(ctx context.Context) (string, error) {
	body, err := httpGet(ctx, "https://api.papermc.io/v2/projects/paper")
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

	return resp.Versions[len(resp.Versions)-1], nil
}

func httpGet(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
