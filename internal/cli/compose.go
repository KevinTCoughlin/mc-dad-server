package cli

import (
	"fmt"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/configs"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// GenerateComposeCmd generates a compose.yml for Docker / Podman Compose.
type GenerateComposeCmd struct {
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
	Output     string `help:"Output directory for compose.yml" default:"." name:"output"`
}

func (cmd *GenerateComposeCmd) toConfig() *config.ServerConfig {
	return &config.ServerConfig{
		Edition:    "java",
		Dir:        cmd.Output,
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

// Run generates a compose.yml file.
func (cmd *GenerateComposeCmd) Run(_ *Globals, _ platform.CommandRunner, output *ui.UI) error {
	cfg := cmd.toConfig()
	if err := cfg.Validate(); err != nil {
		return err
	}

	if err := configs.DeployCompose(cfg, cmd.Output); err != nil {
		return fmt.Errorf("generating compose.yml: %w", err)
	}

	output.Success("compose.yml written to %s", cmd.Output)
	output.Info("")
	output.Info("Start with:  docker compose up -d")
	output.Info("       or:   podman compose up -d")
	output.Info("Stop with:   docker compose down")
	output.Info("")
	return nil
}
