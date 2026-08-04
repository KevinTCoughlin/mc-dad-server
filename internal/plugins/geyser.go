package plugins

import (
	"context"
	"fmt"

	"github.com/KevinTCoughlin/mc-dad-server/internal/verify"
)

// geyserAPIBase is the GeyserMC downloads API. Overridden in tests.
var geyserAPIBase = "https://download.geysermc.org/v2"

// geyserBuild is the subset of the GeyserMC builds API response we need. The
// same shape serves both the geyser and floodgate projects.
type geyserBuild struct {
	Build     int `json:"build"`
	Downloads struct {
		Spigot struct {
			Name   string `json:"name"`
			SHA256 string `json:"sha256"`
		} `json:"spigot"`
	} `json:"downloads"`
}

// geyserSource resolves the latest Spigot build for a GeyserMC project
// ("geyser" or "floodgate") along with its published SHA-256.
func geyserSource(ctx context.Context, project string) (source, error) {
	base := fmt.Sprintf("%s/projects/%s/versions/latest/builds/latest", geyserAPIBase, project)

	var build geyserBuild
	if err := getJSON(ctx, base, &build); err != nil {
		return source{}, fmt.Errorf("fetching %s build metadata: %w", project, err)
	}

	return source{
		url:      base + "/downloads/spigot",
		expected: verify.Expected{SHA256: build.Downloads.Spigot.SHA256},
	}, nil
}
