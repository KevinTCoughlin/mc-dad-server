package management

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

func TestFindWorldDirs(t *testing.T) {
	dir := t.TempDir()

	// No world dirs yet
	if dirs := findWorldDirs(dir); len(dirs) != 0 {
		t.Errorf("expected no dirs, got %v", dirs)
	}

	// Create some world dirs
	if err := os.Mkdir(filepath.Join(dir, "world"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "world_nether"), 0o755); err != nil {
		t.Fatal(err)
	}

	dirs := findWorldDirs(dir)
	if len(dirs) != 2 {
		t.Fatalf("expected 2 dirs, got %d: %v", len(dirs), dirs)
	}
	if dirs[0] != "world" || dirs[1] != "world_nether" {
		t.Errorf("unexpected dirs: %v", dirs)
	}
}

func TestFindWorldDirs_IgnoresFiles(t *testing.T) {
	dir := t.TempDir()
	// Create "world" as a file, not a directory
	if err := os.WriteFile(filepath.Join(dir, "world"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	if dirs := findWorldDirs(dir); len(dirs) != 0 {
		t.Errorf("expected no dirs (file should be ignored), got %v", dirs)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0.0 KB"},
		{1024, "1.0 KB"},
		{512 * 1024, "512.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{5 * 1024 * 1024, "5.0 MB"},
		{1536 * 1024, "1.5 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatSize(tt.bytes); got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestRotateBackups(t *testing.T) {
	dir := t.TempDir()

	// Create 5 backup files
	for _, name := range []string{
		"world_20250101_000000.tar.gz",
		"world_20250102_000000.tar.gz",
		"world_20250103_000000.tar.gz",
		"world_20250104_000000.tar.gz",
		"world_20250105_000000.tar.gz",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Keep 3 — should remove oldest 2
	rotateBackups(dir, 3, ui.New(false))

	entries, _ := os.ReadDir(dir)
	if len(entries) != 3 {
		t.Fatalf("expected 3 files after rotation, got %d", len(entries))
	}

	// Oldest files should be gone
	for _, removed := range []string{"world_20250101_000000.tar.gz", "world_20250102_000000.tar.gz"} {
		if _, err := os.Stat(filepath.Join(dir, removed)); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed", removed)
		}
	}
}
