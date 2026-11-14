package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/configs"
	"github.com/KevinTCoughlin/mc-dad-server/internal/container"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// SetupContainerCmd deploys configs, .env, and the service definition for
// container mode: a Podman Quadlet unit on Linux, or a launchd agent and run
// script for Apple's `container` runtime on macOS.
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
	MCVersion  string `help:"Minecraft version. 'latest' is the newest stable Paper release at or below 26.3; 26.4 is refused." default:"26.3" name:"mc-version"`
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
		// The generated unit and compose file both name the container
		// "minecraft"; Validate requires the field to be set.
		SessionName: "minecraft",
	}
}

// Run deploys container configs, .env, and the runtime's service definition.
func (cmd *SetupContainerCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI, deployer *configs.Deployer) error {
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
	if err := deployer.DeployContainerConfigs(cfg, configDir); err != nil {
		return fmt.Errorf("deploying container configs: %w", err)
	}
	output.Success("Configs written to %s", configDir)

	// Deploy .env file
	output.Step("Creating .env file")
	if err := deployer.DeployContainerEnv(cfg, baseDir); err != nil {
		return fmt.Errorf("deploying .env: %w", err)
	}
	output.Success(".env written to %s", envFile)

	rt := globals.Runtime
	if rt == "" || rt == "auto" {
		rt = platform.DetectContainerRuntime(runner)
	}
	if rt == platform.RuntimeApple {
		return setupAppleContainer(cfg, home, envFile, output, deployer)
	}

	// Deploy Quadlet unit
	output.Step("Installing Quadlet unit")
	if err := deployer.DeployQuadlet(cfg, configDir, envFile, quadletDir); err != nil {
		return fmt.Errorf("deploying quadlet unit: %w", err)
	}
	output.Success("Quadlet unit written to %s", filepath.Join(quadletDir, "minecraft.container"))

	buildRuntime := rt
	if buildRuntime == platform.RuntimeUnknown {
		buildRuntime = platform.RuntimePodman
	}

	// Print next steps
	output.Info("")
	output.Info("Container setup complete! Next steps:")
	output.Info("  1. Build the container image:")
	output.Info("       %s build -t mc-dad-server:latest .", buildRuntime)
	output.Info("  2. Reload systemd and start the service:")
	output.Info("       systemctl --user daemon-reload")
	output.Info("       systemctl --user start minecraft")
	output.Info("  3. Check status:")
	output.Info("       systemctl --user status minecraft")
	output.Info("")

	return nil
}

// setupAppleContainer writes the launchd agent and run script that manage the
// server under Apple's `container` runtime, then prints next steps.
func setupAppleContainer(cfg *config.ServerConfig, home, envFile string, output *ui.UI, deployer *configs.Deployer) error {
	paths := configs.AppleContainerPaths{
		EnvFile:    envFile,
		ScriptPath: filepath.Join(home, ".config", "mc-dad-server", "run-container.sh"),
		PlistPath:  filepath.Join(home, "Library", "LaunchAgents", container.LaunchdLabel+".plist"),
		LogDir:     filepath.Join(home, "Library", "Logs", "mc-dad-server"),
	}

	output.Step("Installing launchd agent for Apple container")
	if err := deployer.DeployAppleContainer(cfg, &paths, container.LaunchdLabel); err != nil {
		return fmt.Errorf("deploying launchd agent: %w", err)
	}
	output.Success("Run script written to %s", paths.ScriptPath)
	output.Success("launchd agent written to %s", paths.PlistPath)

	output.Info("")
	output.Info("Container setup complete! Next steps:")
	output.Info("  1. Build the container image (from the mc-dad-server checkout):")
	output.Info("       container build -t %s .", configs.AppleContainerImage)
	output.Info("  2. Load and start the launchd agent:")
	output.Info("       launchctl bootstrap gui/$(id -u) %s", paths.PlistPath)
	output.Info("  3. Check status:")
	output.Info("       mc-dad-server status --mode container")
	output.Info("     Logs: %s", filepath.Join(paths.LogDir, "minecraft-container.log"))
	output.Info("")
	return nil
}
