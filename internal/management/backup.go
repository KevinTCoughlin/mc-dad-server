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
func Backup(ctx context.Context, serverDir string, maxBackups int, screen *ScreenManager, output *ui.UI) error {
	backupDir := filepath.Join(serverDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("creating backup dir: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("world_%s.tar.gz", timestamp))

	// Notify server and save
	if screen.IsRunning(ctx) {
		_ = screen.SendCommand(ctx, "say Backup starting...")
		_ = screen.SendCommand(ctx, "save-all")
		_ = Sleep(ctx, 3)
		_ = screen.SendCommand(ctx, "save-off")
		_ = Sleep(ctx, 1)
	}

	// Create backup
	output.Info("Creating backup: %s", backupFile)
	worlds := findWorldDirs(serverDir)
	if len(worlds) == 0 {
		output.Warn("No world directories found to backup")
		return nil
	}

	if err := createTarGz(backupFile, serverDir, worlds); err != nil {
		return fmt.Errorf("creating backup archive: %w", err)
	}

	// Re-enable auto-save
	if screen.IsRunning(ctx) {
		_ = screen.SendCommand(ctx, "save-on")
		_ = screen.SendCommand(ctx, "say Backup complete!")
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

func findWorldDirs(serverDir string) []string {
	candidates := []string{"world", "world_nether", "world_the_end"}
	var found []string
	for _, name := range candidates {
		path := filepath.Join(serverDir, name)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			found = append(found, name)
		}
	}
	return found
}

func createTarGz(dest, baseDir string, dirs []string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz := gzip.NewWriter(f)
	defer func() { _ = gz.Close() }()

	tw := tar.NewWriter(gz)
	defer func() { _ = tw.Close() }()

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
			defer func() { _ = file.Close() }()

			_, err = io.Copy(tw, file)
			return err
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
