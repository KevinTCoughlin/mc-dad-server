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

func TestDeployCompose(t *testing.T) {
	setupTestFS(t)

	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.Port = 25565
	cfg.Memory = "4G"
	cfg.ServerType = "paper"
	cfg.MOTD = "My Test Server"
	cfg.MaxPlayers = 15
	cfg.Difficulty = "hard"
	cfg.GameMode = "creative"
	cfg.GCType = "g1gc"
	cfg.Whitelist = true
	cfg.Version = "latest"

	if err := DeployCompose(cfg, dir); err != nil {
		t.Fatalf("DeployCompose() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "compose.yml"))
	if err != nil {
		t.Fatalf("reading compose.yml: %v", err)
	}
	content := string(data)

	checks := []struct {
		desc string
		want string
	}{
		{"port mapping", "25565:25565"},
		{"bedrock UDP port", "19132:19132/udp"},
		{"server type uppercase", `TYPE: "PAPER"`},
		{"memory", `MEMORY: "4G"`},
		{"MOTD", `MOTD: "My Test Server"`},
		{"max players", `MAX_PLAYERS: "15"`},
		{"difficulty", `DIFFICULTY: "hard"`},
		{"game mode", `MODE: "creative"`},
		{"whitelist enabled", `ENABLE_WHITELIST: "true"`},
		{"aikar flags", `USE_AIKAR_FLAGS: "true"`},
		{"volume mount", "minecraft_data:/data"},
		{"volume definition", "minecraft_data:"},
		{"restart policy", "restart: unless-stopped"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.want) {
			t.Errorf("compose.yml missing %s (%q)", c.desc, c.want)
		}
	}
}

func TestDeployComposeZGC(t *testing.T) {
	setupTestFS(t)

	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.GCType = "zgc"

	if err := DeployCompose(cfg, dir); err != nil {
		t.Fatalf("DeployCompose() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "compose.yml"))
	if err != nil {
		t.Fatalf("reading compose.yml: %v", err)
	}

	// With ZGC, Aikar flags should be disabled.
	if !strings.Contains(string(data), `USE_AIKAR_FLAGS: "false"`) {
		t.Error("compose.yml should have USE_AIKAR_FLAGS false for ZGC")
	}
}

func TestDeployContainerConfigs(t *testing.T) {
	setupTestFS(t)

	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.Port = 25565
	cfg.MOTD = "Container Test"
	cfg.Difficulty = "hard"
	cfg.GameMode = "creative"
	cfg.MaxPlayers = 8
	cfg.Whitelist = false
	cfg.RCONPassword = "containerpass"

	if err := DeployContainerConfigs(cfg, dir); err != nil {
		t.Fatalf("DeployContainerConfigs() error: %v", err)
	}

	// All five config files should be in a flat directory
	for _, name := range []string{
		"server.properties",
		"bukkit.yml",
		"spigot.yml",
		"paper-global.yml",
		"paper-world-defaults.yml",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}

	// Verify server.properties substitution
	data, err := os.ReadFile(filepath.Join(dir, "server.properties"))
	if err != nil {
		t.Fatalf("reading server.properties: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "%%MC_PORT%%") {
		t.Error("server.properties still contains %%MC_PORT%% placeholder")
	}
	if !strings.Contains(content, "motd=Container Test") {
		t.Error("server.properties missing motd substitution")
	}
	if !strings.Contains(content, "rcon.password=containerpass") {
		t.Error("server.properties missing rcon password")
	}
}

func TestDeployContainerEnv(t *testing.T) {
	setupTestFS(t)

	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.Port = 25565
	cfg.RCONPassword = "secret123"
	cfg.Version = "1.21.4"

	if err := DeployContainerEnv(cfg, dir); err != nil {
		t.Fatalf("DeployContainerEnv() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	content := string(data)

	checks := []struct {
		desc string
		want string
	}{
		{"rcon password", "RCON_PASSWORD=secret123"},
		{"port", "PORT=25565"},
		{"bedrock port", "BEDROCK_PORT=19132"},
		{"version", "MC_VERSION=1.21.4"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.want) {
			t.Errorf(".env missing %s (%q)", c.desc, c.want)
		}
	}

	// Verify template placeholders are not present
	if strings.Contains(content, "{{") {
		t.Error(".env still contains unsubstituted template placeholders")
	}
}

func TestDeployQuadlet(t *testing.T) {
	setupTestFS(t)

	dir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Dir = dir
	cfg.Port = 25565
	cfg.Memory = "4G"
	cfg.GCType = "g1gc"

	configDir := "/home/user/.config/mc-dad-server/configs"
	envFile := "/home/user/.config/mc-dad-server/.env"

	if err := DeployQuadlet(cfg, configDir, envFile, dir); err != nil {
		t.Fatalf("DeployQuadlet() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "minecraft.container"))
	if err != nil {
		t.Fatalf("reading minecraft.container: %v", err)
	}
	content := string(data)

	checks := []struct {
		desc string
		want string
	}{
		{"java port", "25565:25565/tcp"},
		{"bedrock port", "19132:19132/udp"},
		{"memory env", "MEMORY=4G"},
		{"memory max", "MemoryMax=5G"},
		{"gc type env", "GC_TYPE=g1gc"},
		{"config volume", configDir + "/server.properties:/minecraft/server.properties"},
		{"env file", "EnvironmentFile=" + envFile},
		{"unit description", "Minecraft Paper Server"},
		{"cpu quota comment", "CPUQuota: 200% = 2 CPU cores"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.want) {
			t.Errorf("minecraft.container missing %s (%q)", c.desc, c.want)
		}
	}
}

func TestComputeMemoryMax(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2G", "3G"},
		{"4G", "5G"},
		{"8G", "9G"},
		{"512M", "1536M"},
		{"1024M", "2048M"},
		{"", "3G"},
		{"bad", "3G"},
	}
	for _, tt := range tests {
		got := computeMemoryMax(tt.input)
		if got != tt.want {
			t.Errorf("computeMemoryMax(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
