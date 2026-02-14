package server

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

type fabricInstaller struct {
	URL string `json:"url"`
}

// FabricDownload downloads and runs the Fabric installer.
func FabricDownload(ctx context.Context, version, destDir string, runner platform.CommandRunner, output *ui.UI) error {
	body, err := httpGet(ctx, "https://meta.fabricmc.net/v2/versions/installer")
	if err != nil {
		return fmt.Errorf("fetching Fabric installer versions: %w", err)
	}

	var installers []fabricInstaller
	if err := json.Unmarshal(body, &installers); err != nil {
		return fmt.Errorf("parsing Fabric installer versions: %w", err)
	}

	if len(installers) == 0 {
		return fmt.Errorf("no Fabric installers found")
	}

	installerURL := installers[0].URL
	installerPath := filepath.Join(destDir, "fabric-installer.jar")

	if err := downloadFile(ctx, installerURL, installerPath); err != nil {
		return fmt.Errorf("downloading Fabric installer: %w", err)
	}

	mcVersion := version
	if mcVersion == "latest" {
		mcVersion = "" // Fabric installer uses latest by default
	}

	args := []string{"-jar", installerPath, "server", "-downloadMinecraft"}
	if mcVersion != "" {
		args = append(args, "-mcversion", mcVersion)
	}

	if err := runner.Run(ctx, "java", args...); err != nil {
		return fmt.Errorf("running Fabric installer: %w", err)
	}

	// Rename the output
	src := filepath.Join(destDir, "fabric-server-launch.jar")
	dst := filepath.Join(destDir, "server.jar")
	if err := runner.Run(ctx, "mv", src, dst); err != nil {
		output.Warn("Could not rename fabric-server-launch.jar to server.jar")
	}

	// Cleanup installer
	_ = runner.Run(ctx, "rm", "-f", installerPath)

	output.Success("Fabric server downloaded")
	return nil
}
