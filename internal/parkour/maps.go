package parkour

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// MapEntry describes a parkour map to download.
type MapEntry struct {
	Name string
	URL  string
}

// DefaultMaps returns the curated parkour map catalog.
func DefaultMaps() []MapEntry {
	return []MapEntry{
		{Name: "parkour-spiral", URL: "https://hielkemaps.com/downloads/Parkour Spiral.zip"},
		{Name: "parkour-spiral-3", URL: "https://hielkemaps.com/downloads/Parkour Spiral 3.zip"},
		{Name: "parkour-volcano", URL: "https://hielkemaps.com/downloads/Parkour Volcano.zip"},
		{Name: "parkour-pyramid", URL: "https://hielkemaps.com/downloads/Parkour Pyramid.zip"},
		{Name: "parkour-paradise", URL: "https://hielkemaps.com/downloads/Parkour Paradise.zip"},
	}
}

// ParkourWorldYML is the paper-world.yml content for parkour worlds.
const ParkourWorldYML = `# Paper world config for parkour world
# Optimized for parkour: no mobs, no weather, no explosions

_version: 31

entities:
  spawning:
    spawn-limits:
      ambient: 0
      axolotls: 0
      creature: 0
      monster: 0
      underground_water_creature: 0
      water_ambient: 0
      water_creature: 0

environment:
  disable-explosion-knockback: true
  disable-ice-and-snow: true
  disable-thunder: true
  optimize-explosions: true
`

// DownloadMaps downloads and installs all maps that don't already exist.
func DownloadMaps(ctx context.Context, serverDir string, screen *management.ScreenManager, output *ui.UI, dryRun bool) error {
	output.Info("Parkour map setup starting...")
	output.Info("Server dir: %s", serverDir)

	if dryRun {
		output.Info("DRY RUN - no files will be modified")
	}

	maps := DefaultMaps()
	installed := 0
	skipped := 0

	for _, m := range maps {
		dest := filepath.Join(serverDir, m.Name)

		if info, err := os.Stat(dest); err == nil && info.IsDir() {
			output.Info("SKIP: %s (already exists)", m.Name)
			skipped++
			continue
		}

		if dryRun {
			output.Info("WOULD INSTALL: %s from %s", m.Name, m.URL)
			continue
		}

		output.Info("INSTALLING: %s", m.Name)

		if err := downloadAndExtractMap(ctx, m, serverDir, screen, output); err != nil {
			output.Warn("Failed to install %s: %v", m.Name, err)
			continue
		}

		installed++
		output.Success("Done: %s", m.Name)
	}

	output.Success("Setup complete: %d installed, %d skipped", installed, skipped)
	return nil
}

func downloadAndExtractMap(ctx context.Context, m MapEntry, serverDir string, screen *management.ScreenManager, output *ui.UI) error {
	tmpDir, err := os.MkdirTemp("", "parkour-map-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "map.zip")

	// Download
	output.Info("  Downloading from %s...", m.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d downloading map", resp.StatusCode)
	}

	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// Extract
	output.Info("  Extracting...")
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := unzip(zipPath, extractDir); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Find world folder (contains level.dat)
	worldDir, err := findLevelDat(extractDir)
	if err != nil {
		return err
	}

	// Move to server directory
	dest := filepath.Join(serverDir, m.Name)
	output.Info("  Installing to %s...", dest)
	if err := os.Rename(worldDir, dest); err != nil {
		return fmt.Errorf("moving world: %w", err)
	}

	// Write paper-world.yml
	if err := os.WriteFile(filepath.Join(dest, "paper-world.yml"), []byte(ParkourWorldYML), 0o644); err != nil {
		return fmt.Errorf("writing paper-world.yml: %w", err)
	}
	output.Info("  Created paper-world.yml")

	// Import into Multiverse if server is running
	if screen != nil && screen.IsRunning(ctx) {
		output.Info("  Importing into Multiverse...")
		screen.SendCommand(ctx, fmt.Sprintf("mv import %s normal", m.Name))
	} else {
		output.Info("  Server not running; import with: mv import %s normal", m.Name)
	}

	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)

		// Prevent zip slip
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0o755)
			continue
		}

		os.MkdirAll(filepath.Dir(path), 0o755)

		outFile, err := os.Create(path)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func findLevelDat(dir string) (string, error) {
	var found string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "level.dat" && !info.IsDir() {
			found = filepath.Dir(path)
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && found == "" {
		return "", fmt.Errorf("searching for level.dat: %w", err)
	}
	if found == "" {
		return "", fmt.Errorf("no level.dat found in zip")
	}
	return found, nil
}
