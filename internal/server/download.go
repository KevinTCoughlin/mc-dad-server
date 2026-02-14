package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// Download fetches the server JAR for the given type and version.
func Download(ctx context.Context, serverType, version, destDir string, runner platform.CommandRunner, output *ui.UI) error {
	jarPath := filepath.Join(destDir, "server.jar")

	// Backup existing JAR
	if _, err := os.Stat(jarPath); err == nil {
		output.Warn("server.jar already exists. Backing up to server.jar.bak")
		data, err := os.ReadFile(jarPath)
		if err == nil {
			os.WriteFile(jarPath+".bak", data, 0o644)
		}
	}

	switch serverType {
	case "paper":
		output.Info("Fetching Paper MC server...")
		url, err := PaperDownloadURL(ctx, version)
		if err != nil {
			return err
		}
		output.Info("Downloading from: %s", url)
		if err := downloadFile(ctx, url, jarPath); err != nil {
			return err
		}
		output.Success("Server JAR downloaded")

	case "vanilla":
		output.Info("Fetching Vanilla MC server...")
		url, err := VanillaDownloadURL(ctx, version)
		if err != nil {
			return err
		}
		output.Info("Downloading from: %s", url)
		if err := downloadFile(ctx, url, jarPath); err != nil {
			return err
		}
		output.Success("Server JAR downloaded")

	case "fabric":
		output.Info("Fetching Fabric MC server...")
		if err := FabricDownload(ctx, version, destDir, runner, output); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown server type: %s", serverType)
	}

	return nil
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return fmt.Errorf("writing %s: %w", dest, err)
	}

	return nil
}
