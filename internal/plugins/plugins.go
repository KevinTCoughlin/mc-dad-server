package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// InstallAll downloads all default plugins for a Paper server.
func InstallAll(ctx context.Context, serverDir string, output *ui.UI) error {
	pluginsDir := filepath.Join(serverDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fmt.Errorf("creating plugins dir: %w", err)
	}

	// Geyser
	downloadPlugin(ctx, "Geyser", pluginsDir,
		"https://download.geysermc.org/v2/projects/geyser/versions/latest/builds/latest/downloads/spigot",
		"Geyser-Spigot.jar", output)

	// Floodgate
	downloadPlugin(ctx, "Floodgate", pluginsDir,
		"https://download.geysermc.org/v2/projects/floodgate/versions/latest/builds/latest/downloads/spigot",
		"Floodgate-Spigot.jar", output)

	// Parkour from GitHub releases
	parkourURL, err := githubLatestAssetURL(ctx, "A5H73Y", "Parkour")
	if err == nil && parkourURL != "" {
		downloadPlugin(ctx, "Parkour", pluginsDir, parkourURL, "Parkour.jar", output)
	} else {
		output.Warn("Could not find Parkour download URL — install manually from https://github.com/A5H73Y/Parkour/releases")
	}

	// Multiverse-Core from Hangar
	mvVersion, err := hangarLatestVersion(ctx, "Multiverse-Core")
	if err == nil && mvVersion != "" {
		mvURL := fmt.Sprintf("https://hangar.papermc.io/api/v1/projects/Multiverse-Core/versions/%s/PAPER/download", mvVersion)
		downloadPlugin(ctx, "Multiverse-Core", pluginsDir, mvURL, "Multiverse-Core.jar", output)
	} else {
		output.Warn("Could not resolve Multiverse-Core version — install manually")
	}

	// WorldEdit from Hangar
	weVersion, err := hangarLatestVersion(ctx, "WorldEdit")
	if err == nil && weVersion != "" {
		weURL := fmt.Sprintf("https://hangar.papermc.io/api/v1/projects/WorldEdit/versions/%s/PAPER/download", weVersion)
		downloadPlugin(ctx, "WorldEdit", pluginsDir, weURL, "WorldEdit.jar", output)
	} else {
		output.Warn("Could not resolve WorldEdit version — install manually")
	}

	output.Success("Plugin installation complete")
	return nil
}

func downloadPlugin(ctx context.Context, name, pluginsDir, url, filename string, output *ui.UI) {
	dest := filepath.Join(pluginsDir, filename)
	if _, err := os.Stat(dest); err == nil {
		output.Success("%s already downloaded", name)
		return
	}

	output.Info("Downloading %s...", name)
	if err := downloadFile(ctx, url, dest); err != nil {
		output.Warn("Failed to download %s: %v — you can install it manually", name, err)
		return
	}
	output.Success("%s downloaded", name)
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = io.Copy(f, resp.Body)
	return err
}
