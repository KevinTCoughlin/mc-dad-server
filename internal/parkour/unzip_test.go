package parkour

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeZip builds a zip archive on disk from name -> content pairs.
func writeZip(t *testing.T, entries map[string]string) string {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "map.zip")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUnzipExtractsFiles(t *testing.T) {
	t.Parallel()

	src := writeZip(t, map[string]string{
		"world/level.dat":        "leveldata",
		"world/region/r.0.0.mca": "regiondata",
	})
	dest := filepath.Join(t.TempDir(), "out")

	if err := unzip(src, dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "world", "level.dat"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "leveldata" {
		t.Fatalf("content = %q", got)
	}
}

func TestUnzipRejectsZipSlip(t *testing.T) {
	t.Parallel()

	src := writeZip(t, map[string]string{"../escaped.txt": "pwned"})
	dest := filepath.Join(t.TempDir(), "out")

	err := unzip(src, dest)
	if err == nil {
		t.Fatal("expected a zip-slip rejection")
	}
	if !strings.Contains(err.Error(), "illegal file path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnzipRejectsOversizedEntry(t *testing.T) {
	t.Parallel()

	// Highly compressible content: small on disk, large when expanded — the
	// shape of a zip bomb.
	src := writeZip(t, map[string]string{"world/big.bin": strings.Repeat("A", 4096)})
	dest := filepath.Join(t.TempDir(), "out")

	// Squeeze the budget so the modest fixture trips the same guard a real
	// bomb would, without materialising gigabytes in the test.
	if err := unzipWithLimit(src, dest, 100); err == nil {
		t.Fatal("expected the extraction limit to be enforced")
	}
}
