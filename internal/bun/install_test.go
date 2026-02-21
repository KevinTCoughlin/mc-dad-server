package bun

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// mockKey mirrors MockRunner.key() which is unexported.
func mockKey(name string, args ...string) string {
	return fmt.Sprintf("%s %v", name, args)
}

func TestBunVersion(t *testing.T) {
	runner := platform.NewMockRunner()
	runner.OutputMap[mockKey("bun", "--version")] = []byte("1.3.0\n")

	ver, err := bunVersion(context.Background(), runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.3.0" {
		t.Fatalf("expected 1.3.0, got %s", ver)
	}
}

func TestBunVersion_NotInstalled(t *testing.T) {
	runner := platform.NewMockRunner()
	runner.ErrorMap[mockKey("bun", "--version")] = fmt.Errorf("bun not found")

	_, err := bunVersion(context.Background(), runner)
	if err == nil {
		t.Fatal("expected error for missing bun")
	}
}

func TestInstallBun_AlreadyInstalled(t *testing.T) {
	runner := platform.NewMockRunner()
	runner.ExistsMap["bun"] = true
	runner.OutputMap[mockKey("bun", "--version")] = []byte("1.3.0\n")

	err := InstallBun(context.Background(), runner, nil, ui.New(false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not have run the installer
	for _, cmd := range runner.Commands {
		if cmd.Name == "bash" {
			t.Fatal("expected installer to be skipped when bun already exists")
		}
	}
}

func TestInstallBun_NotInstalled(t *testing.T) {
	runner := platform.NewMockRunner()
	runner.ExistsMap["bun"] = false
	// After install, bun --version should succeed
	runner.OutputMap[mockKey("bun", "--version")] = []byte("1.3.0\n")

	err := InstallBun(context.Background(), runner, nil, ui.New(false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have run the bash installer
	found := false
	for _, cmd := range runner.Commands {
		if cmd.Name == "bash" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected installer to run when bun is not installed")
	}
}

func setupTestEmbedFS(t *testing.T) {
	t.Helper()

	projRoot := findProjectRoot(t)
	embDir := filepath.Join(projRoot, "embedded")

	fs := make(fstest.MapFS)
	err := filepath.Walk(embDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(projRoot, path)
		key := filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fs[key] = &fstest.MapFile{Data: data}
		return nil
	})
	if err != nil {
		t.Fatalf("walking embedded dir: %v", err)
	}

	embeddedFS = fs
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

func TestDeployScripts(t *testing.T) {
	setupTestEmbedFS(t)

	tmpDir := t.TempDir()
	cfg := &config.ServerConfig{
		Dir:          tmpDir,
		RCONPassword: "testpass123",
	}

	err := DeployScripts(cfg)
	if err != nil {
		t.Fatalf("DeployScripts failed: %v", err)
	}

	// Verify runtime files exist
	runtimeFiles := []string{
		"runtime/types.ts", "runtime/events.ts", "runtime/rcon.ts",
		"runtime/log-parser.ts", "runtime/players.ts", "runtime/scheduler.ts",
		"runtime/webhooks.ts", "runtime/server.ts", "runtime/command-filter.ts",
		"runtime/rate-limiter.ts", "runtime/integrity.ts", "runtime/index.ts",
	}
	for _, f := range runtimeFiles {
		path := filepath.Join(tmpDir, "bun-scripts", f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}

	// Verify example script deployed
	examplePath := filepath.Join(tmpDir, "bun-scripts", "scripts", "example.ts")
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Error("expected example.ts to exist")
	}

	// Verify .env does NOT contain RCON password (passed at runtime instead)
	envData, err := os.ReadFile(filepath.Join(tmpDir, "bun-scripts", ".env"))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	envStr := string(envData)
	if strings.Contains(envStr, "testpass123") {
		t.Errorf("expected .env to NOT contain RCON password, got: %s", envStr)
	}
	if !strings.Contains(envStr, "RCON_PORT=") {
		t.Errorf("expected .env to contain RCON_PORT, got: %s", envStr)
	}

	// Verify package.json exists
	if _, err := os.Stat(filepath.Join(tmpDir, "bun-scripts", "package.json")); os.IsNotExist(err) {
		t.Error("expected package.json to exist")
	}

	// Verify tsconfig.json exists
	if _, err := os.Stat(filepath.Join(tmpDir, "bun-scripts", "tsconfig.json")); os.IsNotExist(err) {
		t.Error("expected tsconfig.json to exist")
	}
}

func TestDeployScripts_PreservesExistingScripts(t *testing.T) {
	setupTestEmbedFS(t)

	tmpDir := t.TempDir()
	cfg := &config.ServerConfig{
		Dir:          tmpDir,
		RCONPassword: "testpass",
	}

	// Create an existing user script
	scriptsDir := filepath.Join(tmpDir, "bun-scripts", "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	userScript := filepath.Join(scriptsDir, "my-script.ts")
	if err := os.WriteFile(userScript, []byte("// my custom script"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := DeployScripts(cfg)
	if err != nil {
		t.Fatalf("DeployScripts failed: %v", err)
	}

	// User script should still exist
	if _, err := os.Stat(userScript); os.IsNotExist(err) {
		t.Error("expected user script to be preserved")
	}

	// Example script should NOT be deployed (scripts/ not empty)
	examplePath := filepath.Join(scriptsDir, "example.ts")
	if _, err := os.Stat(examplePath); !os.IsNotExist(err) {
		t.Error("expected example.ts to NOT be deployed when scripts/ has existing files")
	}
}
