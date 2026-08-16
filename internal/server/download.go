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
		restore, err := backupExistingJAR(jarPath, output)
		if err != nil {
			return err
		}
		if err := FabricDownload(ctx, version, destDir, runner, output); err != nil {
			restore()
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
	f, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".*.part")
	if err != nil {
		return fmt.Errorf("creating temporary download: %w", err)
	}
	tmpPath := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing temporary download: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if err := downloadFile(ctx, art.URL, tmpPath); err != nil {
		return err
	}

	if err := art.Verify(tmpPath); err != nil {
		return fmt.Errorf("verifying server JAR: %w", err)
	}

	restore, err := backupExistingJAR(dest, output)
	if err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		restore()
		return fmt.Errorf("installing server JAR: %w", err)
	}

	if art.SHA256 == "" && art.SHA1 == "" {
		output.Warn("Server JAR downloaded (no upstream checksum published to verify against)")
		return nil
	}
	output.Success("Server JAR downloaded and checksum verified")
	return nil
}

func backupExistingJAR(path string, output *ui.UI) (func(), error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return func() {}, nil
	} else if err != nil {
		return nil, fmt.Errorf("checking existing server JAR: %w", err)
	}

	backupPath := path + ".bak"
	output.Warn("server.jar already exists. Backing up to server.jar.bak")
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("removing old server JAR backup: %w", err)
	}
	if err := os.Rename(path, backupPath); err != nil {
		return nil, fmt.Errorf("backing up existing server JAR: %w", err)
	}

	return func() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			output.Warn("Failed to remove incomplete server.jar: %v", err)
			return
		}
		if err := os.Rename(backupPath, path); err != nil {
			output.Warn("Failed to restore server.jar backup: %v", err)
		}
	}, nil
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
