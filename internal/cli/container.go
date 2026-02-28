package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/configs"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// SetupContainerCmd deploys configs, .env, and Quadlet unit for container mode.
type SetupContainerCmd struct {
	Port       int    `help:"Server port" default:"25565"`
	Memory     string `help:"RAM allocation (e.g., 2G, 4G)" default:"2G"`
	Type       string `help:"Server type" default:"paper" enum:"paper,fabric,vanilla" name:"type"`
	MOTD       string `help:"Message of the day" default:"Dads Minecraft Server" name:"motd"`
	Players    int    `help:"Max players" default:"20" name:"players"`
	Difficulty string `help:"Difficulty" default:"normal" enum:"peaceful,easy,normal,hard"`
	Gamemode   string `help:"Game mode" default:"survival" enum:"survival,creative,adventure" name:"gamemode"`
	GC         string `help:"Garbage collector" default:"g1gc" enum:"g1gc,zgc,G1GC,ZGC" name:"gc"`
	Whitelist  bool   `help:"Enable whitelist" default:"true" negatable:""`
	MCVersion  string `help:"Minecraft version" default:"latest" name:"mc-version"`
}

func (cmd *SetupContainerCmd) toConfig() *config.ServerConfig {
	return &config.ServerConfig{
		Edition:    "java",
		Dir:        ".",
		Port:       cmd.Port,
		Memory:     cmd.Memory,
		ServerType: cmd.Type,
		MOTD:       cmd.MOTD,
		MaxPlayers: cmd.Players,
		Difficulty: cmd.Difficulty,
		GameMode:   cmd.Gamemode,
		GCType:     strings.ToLower(cmd.GC),
		Whitelist:  cmd.Whitelist,
		Version:    cmd.MCVersion,
		MaxBackups: 5,
	}
}

// Run deploys container configs, .env, and Quadlet unit.
func (cmd *SetupContainerCmd) Run(_ *Globals, runner platform.CommandRunner, output *ui.UI) error {
	cfg := cmd.toConfig()
	if err := cfg.Validate(); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("detecting home directory: %w", err)
	}

	baseDir := filepath.Join(home, ".config", "mc-dad-server")
	configDir := filepath.Join(baseDir, "configs")
	quadletDir := filepath.Join(home, ".config", "containers", "systemd")
	envFile := filepath.Join(baseDir, ".env")

	// Generate RCON password
	cfg.RCONPassword = generateRCONPassword()

	// Deploy server configs
	output.Step("Deploying server configs")
	if err := configs.DeployContainerConfigs(cfg, configDir); err != nil {
		return fmt.Errorf("deploying container configs: %w", err)
	}
	output.Success("Configs written to %s", configDir)

	// Deploy .env file
	output.Step("Creating .env file")
	if err := configs.DeployContainerEnv(cfg, baseDir); err != nil {
		return fmt.Errorf("deploying .env: %w", err)
	}
	output.Success(".env written to %s", envFile)

	// Deploy Quadlet unit
	output.Step("Installing Quadlet unit")
	if err := configs.DeployQuadlet(cfg, configDir, envFile, quadletDir); err != nil {
		return fmt.Errorf("deploying quadlet unit: %w", err)
	}
	output.Success("Quadlet unit written to %s", filepath.Join(quadletDir, "minecraft.container"))

	// Detect container runtime
	runtime := "podman"
	if runner.CommandExists("podman") {
		runtime = "podman"
	} else if runner.CommandExists("docker") {
		runtime = "docker"
	}

	// Print next steps
	output.Info("")
	output.Info("Container setup complete! Next steps:")
	output.Info("  1. Build the container image:")
	output.Info("       %s build -t mc-dad-server:latest .", runtime)
	output.Info("  2. Reload systemd and start the service:")
	output.Info("       systemctl --user daemon-reload")
	output.Info("       systemctl --user start minecraft")
	output.Info("  3. Check status:")
	output.Info("       systemctl --user status minecraft")
	output.Info("")

	return nil
}
