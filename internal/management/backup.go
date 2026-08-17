package management

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// Backup creates a compressed backup of world directories with rotation.
func Backup(ctx context.Context, serverDir string, maxBackups int, mgr ServerManager, output *ui.UI) error {
	backupDir := filepath.Join(serverDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("creating backup dir: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("world_%s.tar.gz", timestamp))

	// Notify server and save. Auto-save is re-enabled from a defer so that a
	// failed or aborted backup can never leave the live server with
	// save-off still in effect.
	if mgr.IsRunning(ctx) {
		_ = mgr.SendCommand(ctx, "say Backup starting...")
		_ = mgr.SendCommand(ctx, "save-all")
		_ = Sleep(ctx, 3)
		_ = mgr.SendCommand(ctx, "save-off")
		_ = Sleep(ctx, 1)

		defer func() {
			// Use a fresh context: the caller's may already be cancelled,
			// and re-enabling auto-save must still be attempted.
			restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			_ = mgr.SendCommand(restoreCtx, "save-on")
		}()
	}

	// Create backup
	output.Info("Creating backup: %s", backupFile)
	worlds := findWorldDirs(serverDir)
	if len(worlds) == 0 {
		output.Warn("No world directories found to backup")
		return nil
	}

	if err := createTarGz(backupFile, serverDir, worlds); err != nil {
		// Don't leave a half-written archive behind — a later run would
		// otherwise rotate good backups out in favour of a corrupt one.
		_ = os.Remove(backupFile)
		return fmt.Errorf("creating backup archive: %w", err)
	}

	if mgr.IsRunning(ctx) {
		_ = mgr.SendCommand(ctx, "say Backup complete!")
	}

	// Rotate old backups
	rotateBackups(backupDir, maxBackups, output)

	// Print size
	info, err := os.Stat(backupFile)
	if err == nil {
		output.Success("Backup complete: %s (%s)", backupFile, formatSize(info.Size()))
	}

	return nil
}

// worldDirs are the Minecraft world directories to include in backups.
var worldDirs = []string{"world", "world_nether", "world_the_end"}

func findWorldDirs(serverDir string) []string {
	candidates := worldDirs
	var found []string
	for _, name := range candidates {
		path := filepath.Join(serverDir, name)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			found = append(found, name)
		}
	}
	return found
}

// createTarGz writes dirs (relative to baseDir) into a gzipped tar at dest.
// The writers are closed explicitly so that a flush failure surfaces as an
// error instead of silently producing a truncated archive.
func createTarGz(dest, baseDir string, dirs []string) (err error) {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing backup archive: %w", closeErr)
		}
	}()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	if err := writeTarEntries(tw, baseDir, dirs); err != nil {
		_ = tw.Close()
		_ = gz.Close()
		return err
	}

	if err := tw.Close(); err != nil {
		_ = gz.Close()
		return fmt.Errorf("finalizing tar: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("finalizing gzip: %w", err)
	}

	return nil
}

func writeTarEntries(tw *tar.Writer, baseDir string, dirs []string) error {
	for _, dir := range dirs {
		dirPath := filepath.Join(baseDir, dir)
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = relPath

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}

			_, copyErr := io.Copy(tw, file)
			closeErr := file.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func rotateBackups(backupDir string, maxBackups int, output *ui.UI) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}

	var backups []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "world_") && strings.HasSuffix(e.Name(), ".tar.gz") {
			backups = append(backups, filepath.Join(backupDir, e.Name()))
		}
	}

	if len(backups) <= maxBackups {
		return
	}

	sort.Strings(backups) // Sorted by timestamp in name
	toRemove := backups[:len(backups)-maxBackups]
	for _, f := range toRemove {
		_ = os.Remove(f)
	}
	output.Info("Rotated old backups (keeping %d)", maxBackups)
}

func formatSize(bytes int64) string {
	const mb = 1024 * 1024
	if bytes >= mb {
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	}
	return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
}
