package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
			if writeErr := os.WriteFile(jarPath+".bak", data, 0o644); writeErr != nil {
				output.Warn("Failed to write backup: %v", writeErr)
			}
		}
	}

	switch serverType {
	case "paper":
		output.Info("Fetching Paper MC server...")
		art, err := PaperArtifact(ctx, version)
		if err != nil {
			return err
		}
		if err := fetchAndVerify(ctx, art, jarPath, output); err != nil {
			return err
		}

	case "vanilla":
		output.Info("Fetching Vanilla MC server...")
		art, err := VanillaArtifact(ctx, version)
		if err != nil {
			return err
		}
		if err := fetchAndVerify(ctx, art, jarPath, output); err != nil {
			return err
		}

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

// httpClient is used for JAR downloads. http.DefaultClient has no timeout, so
// an unresponsive mirror would hang the installer indefinitely.
var httpClient = &http.Client{Timeout: 10 * time.Minute}

// fetchAndVerify downloads the artifact to dest and checks it against the
// checksum published by the upstream API, removing the file if it fails.
func fetchAndVerify(ctx context.Context, art Artifact, dest string, output *ui.UI) error {
	output.Info("Downloading from: %s", art.URL)
	if err := downloadFile(ctx, art.URL, dest); err != nil {
		return err
	}

	if err := art.Verify(dest); err != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("verifying server JAR: %w", err)
	}

	if art.SHA256 == "" && art.SHA1 == "" {
		output.Warn("Server JAR downloaded (no upstream checksum published to verify against)")
		return nil
	}
	output.Success("Server JAR downloaded and checksum verified")
	return nil
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, url)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("writing %s: %w", dest, err)
	}

	return nil
}
