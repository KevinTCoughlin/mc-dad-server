package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/container"
	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/nag"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
	"github.com/KevinTCoughlin/mc-dad-server/internal/vote"
)

// StartCmd starts the Minecraft server in a screen session.
type StartCmd struct{}

// Run starts the server.
func (cmd *StartCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := globalsToConfig(globals)
	mgr := resolveManager(ctx, globals, runner, cfg)

	alreadyRunning, err := management.StartServer(ctx, mgr, runner, cfg.Port, cfg.Dir, cfg.SessionName, output)
	if err != nil {
		return err
	}
	if !alreadyRunning {
		mode := resolveMode(ctx, globals, runner)
		if mode == "container" {
			output.Info("")
			output.Info("  View logs:    mc-dad-server --mode container status")
			output.Info("  Stop server:  mc-dad-server stop")
			output.Info("")
		} else {
			output.Info("")
			output.Info("  Attach to console:  screen -r %s", cfg.SessionName)
			output.Info("  Detach from console: Ctrl+A then D")
			output.Info("  Stop server:         mc-dad-server stop")
			output.Info("  Server status:       mc-dad-server status")
			output.Info("")
		}
	}
	nagInfo := nag.Resolve(ctx, cfg.Dir)
	nag.MaybeNag(output, nagInfo)
	return nil
}

// StopCmd gracefully stops the Minecraft server.
type StopCmd struct{}

// Run stops the server.
func (cmd *StopCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := globalsToConfig(globals)
	mgr := resolveManager(ctx, globals, runner, cfg)

	if err := management.StopServer(ctx, mgr, runner, cfg.Port, output); err != nil {
		return err
	}
	nagInfo := nag.Resolve(ctx, cfg.Dir)
	nag.MaybeNag(output, nagInfo)
	return nil
}

// StatusCmd shows server status and resource usage.
type StatusCmd struct{}

// Run shows server status.
func (cmd *StatusCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := globalsToConfig(globals)
	mgr := resolveManager(ctx, globals, runner, cfg)

	mode := resolveMode(ctx, globals, runner)
	if mode == "container" {
		printContainerStatus(ctx, mgr, cfg, output)
	} else {
		management.PrintStatus(ctx, mgr, runner, cfg.Port, cfg.SessionName, output)
	}
	output.Info("")

	nagInfo := nag.Resolve(ctx, cfg.Dir)
	output.Info("  License: %s", nag.StatusLabel(nagInfo))
	output.Info("")
	nag.MaybeNag(output, nagInfo)
	return nil
}

// printContainerStatus shows container-specific status information.
func printContainerStatus(ctx context.Context, mgr management.ServerManager, cfg *config.ServerConfig, output *ui.UI) {
	output.Step("Minecraft Server Status (container)")

	cm, ok := mgr.(*container.ContainerManager)
	if !ok {
		output.Info("  Status:  UNKNOWN (not a container manager)")
		return
	}

	if cm.IsRunning(ctx) {
		health := cm.Health(ctx)
		output.Info("  Status:    RUNNING (%s)", health)
		output.Info("  Container: %s", cm.Session())
		if stats, err := cm.Stats(ctx); err == nil {
			output.Info("  Resources: %s", stats)
		}
	} else if management.IsPortListening(cfg.Port) {
		output.Info("  Status:  RUNNING (port %d)", cfg.Port)
	} else {
		output.Info("  Status:  STOPPED")
	}
}

// BackupCmd backs up world data with rotation.
type BackupCmd struct{}

// Run performs a backup.
func (cmd *BackupCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := globalsToConfig(globals)
	mgr := resolveManager(ctx, globals, runner, cfg)
	return management.Backup(ctx, cfg.Dir, cfg.MaxBackups, mgr, output)
}

// SetupParkourCmd sets up the parkour world (first-time setup).
type SetupParkourCmd struct{}

// Run sets up the parkour world.
func (cmd *SetupParkourCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := globalsToConfig(globals)
	mgr := resolveManager(ctx, globals, runner, cfg)

	if !management.IsServerRunning(ctx, mgr, runner, cfg.Port) {
		return fmt.Errorf("server not running — start it first with: mc-dad-server start")
	}

	output.Info("Setting up parkour world...")
	cmds := []string{
		"mv create parkour normal --world-type flat --no-structures",
	}
	for _, c := range cmds {
		if err := mgr.SendCommand(ctx, c); err != nil {
			return err
		}
		if err := management.Sleep(ctx, 2); err != nil {
			return err
		}
	}
	if err := management.Sleep(ctx, 3); err != nil {
		return err
	}

	gamerules := []string{
		"mv modify parkour set gamemode adventure",
		"mv modify parkour set difficulty peaceful",
		"mv gamerule set minecraft:spawn_mobs false parkour",
		"mv gamerule set minecraft:advance_weather false parkour",
		"mv gamerule set minecraft:advance_time false parkour",
		"mv gamerule set minecraft:fire_damage false parkour",
		"mv gamerule set minecraft:spawn_monsters false parkour",
		"mv gamerule set minecraft:spawn_phantoms false parkour",
		"mv gamerule set minecraft:mob_griefing false parkour",
	}
	for _, c := range gamerules {
		if err := mgr.SendCommand(ctx, c); err != nil {
			return err
		}
		if err := management.Sleep(ctx, 2); err != nil {
			return err
		}
	}

	output.Success("Parkour world created!")
	output.Info("")
	output.Info("Next steps:")
	output.Info("  1. Join the server and run: /mv tp parkour")
	output.Info("  2. Fly to where you want the parkour lobby")
	output.Info("  3. Run: /pa setlobby")
	output.Info("  4. Start building courses with: /pa create <name>")
	return nil
}

// RotateParkourCmd rotates the featured parkour map.
type RotateParkourCmd struct{}

// Run rotates the featured parkour map.
func (cmd *RotateParkourCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := globalsToConfig(globals)
	mgr := resolveManager(ctx, globals, runner, cfg)

	if !management.IsServerRunning(ctx, mgr, runner, cfg.Port) {
		output.Info("Server not running, skipping rotation")
		return nil
	}

	return management.RotateParkour(ctx, cfg.Dir, mgr, output)
}

// VoteMapCmd starts a map vote (CS:GO style).
type VoteMapCmd struct {
	Duration int `help:"Vote duration in seconds" default:"30"`
	Choices  int `help:"Number of maps to vote on" default:"5" name:"choices"`
}

// Run starts a map vote.
func (cmd *VoteMapCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := globalsToConfig(globals)
	mgr := resolveManager(ctx, globals, runner, cfg)

	if !management.IsServerRunning(ctx, mgr, runner, cfg.Port) {
		return fmt.Errorf("server not running — start it first with: mc-dad-server start")
	}

	result, err := vote.RunVote(ctx, vote.Config{
		Maps:       management.ParkourMaps,
		Duration:   time.Duration(cmd.Duration) * time.Second,
		MaxChoices: cmd.Choices,
		ServerDir:  cfg.Dir,
		Screen:     mgr,
		Output:     output,
	})
	if err != nil {
		return err
	}

	output.Success("Map vote complete: %s (%d voters)", result.Winner, result.Voters)
	return nil
}

// globalsToConfig creates a minimal ServerConfig from the global flags.
func globalsToConfig(g *Globals) *config.ServerConfig {
	cfg := config.DefaultConfig()
	cfg.Dir = g.Dir
	cfg.SessionName = g.Session
	return cfg
}

// resolveManager returns a ServerManager based on the resolved mode.
func resolveManager(ctx context.Context, globals *Globals, runner platform.CommandRunner, cfg *config.ServerConfig) management.ServerManager {
	mode := resolveMode(ctx, globals, runner)
	if mode == "container" {
		rconPass := readRCONPassword(cfg.Dir)
		return container.NewContainerManager(runner, cfg.SessionName, "127.0.0.1:25575", rconPass)
	}
	return management.NewScreenManager(runner, cfg.SessionName)
}

// resolveMode determines the server mode from the --mode flag or auto-detection.
func resolveMode(ctx context.Context, globals *Globals, runner platform.CommandRunner) string {
	switch globals.Mode {
	case "screen":
		return "screen"
	case "container":
		return "container"
	default:
		return detectMode(ctx, globals, runner)
	}
}

// detectMode auto-detects whether to use container or screen mode.
// Priority: running container > running screen session > default screen.
func detectMode(ctx context.Context, globals *Globals, runner platform.CommandRunner) string {
	if container.ContainerExists(ctx, runner, globals.Session) {
		return "container"
	}
	return "screen"
}

// readRCONPassword reads the RCON password from server.properties in the server dir.
func readRCONPassword(serverDir string) string {
	data, err := os.ReadFile(filepath.Join(serverDir, "server.properties"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "rcon.password=") {
			return strings.TrimPrefix(line, "rcon.password=")
		}
	}
	return ""
}
