package server

import (
	"context"
	"errors"
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

const downloadAttempts = 3

var downloadRetryDelay = 500 * time.Millisecond

// fetchAndVerify downloads the artifact to dest and checks it against the
// checksum published by the upstream API, removing the file if it fails.
func fetchAndVerify(ctx context.Context, art Artifact, dest string, output *ui.UI) error {
	output.Info("Downloading from: %s", art.URL)
	f, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".*.download")
	if err != nil {
		return fmt.Errorf("creating temporary file for %s: %w", dest, err)
	}
	tmpPath := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("closing %s: %w", tmpPath, err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("setting temporary download permissions: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if err := downloadFile(ctx, art.URL, tmpPath); err != nil {
		return err
	}

	if err := art.Verify(tmpPath); err != nil {
		return fmt.Errorf("verifying server JAR: %w", err)
	}
	if err := renameOverwrite(tmpPath, dest); err != nil {
		return fmt.Errorf("installing %s: %w", dest, err)
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
	var lastErr error
	for attempt := 1; attempt <= downloadAttempts; attempt++ {
		err := downloadFileOnce(ctx, url, dest)
		if err == nil {
			return nil
		}

		lastErr = err
		if attempt == downloadAttempts || !isRetryableDownloadError(err) {
			break
		}
		if err := sleepForRetry(ctx); err != nil {
			return err
		}
	}
	return lastErr
}

func sleepForRetry(ctx context.Context) error {
	if downloadRetryDelay <= 0 {
		return nil
	}
	timer := time.NewTimer(downloadRetryDelay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func downloadFileOnce(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return retryableError{err: fmt.Errorf("downloading %s: %w", url, err)}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, url)
		if isRetryableStatus(resp.StatusCode) {
			return retryableError{err: err}
		}
		return err
	}

	f, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".*.part")
	if err != nil {
		return fmt.Errorf("creating temporary file for %s: %w", dest, err)
	}
	tmpPath := f.Name()
	closed := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return retryableError{err: fmt.Errorf("writing %s: %w", tmpPath, err)}
	}
	if err := f.Close(); err != nil {
		return retryableError{err: fmt.Errorf("closing %s: %w", tmpPath, err)}
	}
	closed = true
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return fmt.Errorf("setting permissions on %s: %w", tmpPath, err)
	}
	if err := renameOverwrite(tmpPath, dest); err != nil {
		return fmt.Errorf("installing %s: %w", dest, err)
	}

	return nil
}

// renameOverwrite renames oldpath to newpath, overwriting newpath if it
// already exists. os.Rename silently overwrites an existing file on POSIX
// systems but fails with an "already exists" error on Windows, so newpath is
// first moved aside and only removed once oldpath has been renamed
// successfully; on failure the original newpath is restored.
func renameOverwrite(oldpath, newpath string) error {
	backupPath := newpath + ".bak-rename"
	// Clear out any stale backup left behind by a previous crashed run;
	// os.Rename below would otherwise fail on Windows if backupPath already
	// exists.
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing stale backup %s: %w", backupPath, err)
	}

	hasBackup := false
	if err := os.Rename(newpath, backupPath); err == nil {
		hasBackup = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("backing up existing %s: %w", newpath, err)
	}

	if err := os.Rename(oldpath, newpath); err != nil {
		if hasBackup {
			if restoreErr := os.Rename(backupPath, newpath); restoreErr != nil {
				return errors.Join(err, fmt.Errorf("restoring %s from backup %s: %w", newpath, backupPath, restoreErr))
			}
		}
		return err
	}

	if hasBackup {
		_ = os.Remove(backupPath)
	}
	return nil
}

type retryableError struct {
	err error
}

func (e retryableError) Error() string {
	return e.err.Error()
}

func (e retryableError) Unwrap() error {
	return e.err
}

func isRetryableDownloadError(err error) bool {
	var retryable retryableError
	return errors.As(err, &retryable)
}

func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}
