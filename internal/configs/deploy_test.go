package configs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
)

func setupTestFS(t *testing.T) {
	t.Helper()

	// Create a test embed.FS-compatible filesystem using the real files.
	// Since we can't use go:embed in test files, we'll read from disk.
	// The embedded directory is at ../../embedded relative to this package.
	projRoot := findProjectRoot(t)
	embDir := filepath.Join(projRoot, "embedded")

	fs := make(fstest.MapFS)
	err := filepath.Walk(embDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(projRoot, path)
		// Normalize to forward slashes for fs.FS compatibility on Windows.
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

func TestDeploy(t *testing.T) {
	setupTestFS(t)

	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.Port = 25565
	cfg.MOTD = "Test Server"
	cfg.Difficulty = "normal"
	cfg.GameMode = "survival"
	cfg.MaxPlayers = 10
	cfg.Whitelist = true
	cfg.RCONPassword = "testpass123"

	if err := Deploy(cfg); err != nil {
		t.Fatalf("Deploy() error: %v", err)
	}

	// Check server.properties was written with substitutions
	data, err := os.ReadFile(filepath.Join(dir, "server.properties"))
	if err != nil {
		t.Fatalf("reading server.properties: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "%%MC_PORT%%") {
		t.Error("server.properties still contains %%MC_PORT%% placeholder")
	}
	if !strings.Contains(content, "server-port=25565") {
		t.Error("server.properties missing server-port=25565")
	}
	if !strings.Contains(content, "motd=Test Server") {
		t.Error("server.properties missing motd substitution")
	}
	if !strings.Contains(content, "rcon.password=testpass123") {
		t.Error("server.properties missing rcon password")
	}

	// Check paper configs in config/ subdir
	for _, name := range []string{"paper-global.yml", "paper-world-defaults.yml"} {
		if _, err := os.Stat(filepath.Join(dir, "config", name)); err != nil {
			t.Errorf("missing config/%s: %v", name, err)
		}
	}

	// Check bukkit.yml and spigot.yml
	for _, name := range []string{"bukkit.yml", "spigot.yml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}

	// Check parkour config
	parkourCfg := filepath.Join(dir, "plugins", "Parkour", "config.yml")
	if _, err := os.Stat(parkourCfg); err != nil {
		t.Errorf("missing parkour config: %v", err)
	}
}

func TestDeployStartScript(t *testing.T) {
	setupTestFS(t)

	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.Memory = "4G"
	cfg.GCType = "zgc"

	if err := DeployStartScript(cfg); err != nil {
		t.Fatalf("DeployStartScript() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "start.sh"))
	if err != nil {
		t.Fatalf("reading start.sh: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "4G") {
		t.Error("start.sh missing memory setting")
	}
}
