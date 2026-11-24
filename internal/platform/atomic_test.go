package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteAtomicReplacesAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.properties")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := WriteAtomic(path, []byte("eula=true\n"), 0o640); err != nil {
		t.Fatalf("WriteAtomic() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if string(got) != "eula=true\n" {
		t.Fatalf("content = %q", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.Contains(name, ".tmp-") || strings.HasSuffix(name, ".bak-rename") {
			t.Errorf("leftover file %s", name)
		}
	}

	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
}

func TestWriteAtomicCreatesParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "paper-global.yml")
	if err := WriteAtomic(path, []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("WriteAtomic() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if string(got) != "ok\n" {
		t.Fatalf("content = %q", got)
	}
}
