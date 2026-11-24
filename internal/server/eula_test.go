package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureEULAWritesAcceptance(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureEULA(dir, true); err != nil {
		t.Fatalf("EnsureEULA() error = %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "eula.txt"))
	if err != nil {
		t.Fatalf("reading eula: %v", err)
	}
	if !strings.Contains(string(body), "eula=true") || !strings.Contains(string(body), EULAURL) {
		t.Fatalf("eula.txt = %q", body)
	}
	if err := EnsureEULA(dir, false); err != nil {
		t.Fatalf("existing acceptance should satisfy --accept-eula=false: %v", err)
	}
}

func TestEnsureEULARefusesWithoutAgreement(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "eula.txt"), []byte("eula=false\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	err := EnsureEULA(dir, false)
	if err == nil {
		t.Fatal("expected an error when the EULA is declined")
	}
	if !strings.Contains(err.Error(), EULAURL) || !strings.Contains(err.Error(), "--accept-eula") {
		t.Fatalf("error = %q", err)
	}

	body, err := os.ReadFile(filepath.Join(dir, "eula.txt"))
	if err != nil {
		t.Fatalf("reading eula: %v", err)
	}
	if string(body) != "eula=false\n" {
		t.Fatalf("declined eula was rewritten: %q", body)
	}
}

func TestEnsureEULAOverwritesFalseWhenAccepted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "eula.txt")
	if err := os.WriteFile(path, []byte("eula=false\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := EnsureEULA(dir, true); err != nil {
		t.Fatalf("EnsureEULA() error = %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading eula: %v", err)
	}
	if !strings.Contains(string(body), "eula=true") {
		t.Fatalf("eula.txt = %q", body)
	}
}
