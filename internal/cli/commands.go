package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
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
	screen := management.NewScreenManager(runner, cfg.SessionName)

	if management.IsServerRunning(ctx, screen, runner, cfg.Port) {
		output.Warn("Server is already running!")
		return nil
	}

	output.Info("Starting Minecraft server in screen session '%s'...", cfg.SessionName)
	if err := screen.Start(ctx, "bash", cfg.Dir+"/start.sh"); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}
	output.Success("Server started!")
	output.Info("")
	output.Info("  Attach to console:  screen -r %s", cfg.SessionName)
	output.Info("  Detach from console: Ctrl+A then D")
	output.Info("  Stop server:         mc-dad-server stop")
	output.Info("  Server status:       mc-dad-server status")
	output.Info("")
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
	screen := management.NewScreenManager(runner, cfg.SessionName)

	if !management.IsServerRunning(ctx, screen, runner, cfg.Port) {
		output.Info("No running Minecraft server found.")
		return nil
	}

	output.Info("Sending shutdown command...")
	if err := screen.SendCommand(ctx, "say Server shutting down in 10 seconds..."); err != nil {
		return err
	}
	if err := management.Sleep(ctx, 10); err != nil {
		return err
	}
	if err := screen.SendCommand(ctx, "stop"); err != nil {
		return err
	}
	output.Success("Stop command sent. Server shutting down...")
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
	screen := management.NewScreenManager(runner, cfg.SessionName)

	output.Step("Minecraft Server Status")

	stats, err := management.GetProcessStats(ctx, runner)

	if screen.IsRunning(ctx) {
		output.Info("  Status:  RUNNING")
		output.Info("  Session: screen -r %s", cfg.SessionName)
	} else if err == nil && stats.PID > 0 {
		output.Info("  Status:  RUNNING (pid %d)", stats.PID)
	} else if management.IsPortListening(cfg.Port) {
		output.Info("  Status:  RUNNING (port %d)", cfg.Port)
	} else {
		output.Info("  Status:  STOPPED")
	}
	output.Info("")

	if err == nil && stats.PID > 0 {
		output.Info("  PID:     %d", stats.PID)
		output.Info("  Memory:  %s", stats.Memory)
		output.Info("  CPU:     %s", stats.CPU)
	}
	output.Info("")

	nagInfo := nag.Resolve(ctx, cfg.Dir)
	output.Info("  License: %s", nag.StatusLabel(nagInfo))
	output.Info("")
	nag.MaybeNag(output, nagInfo)
	return nil
}

// BackupCmd backs up world data with rotation.
type BackupCmd struct{}

// Run performs a backup.
func (cmd *BackupCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := globalsToConfig(globals)
	screen := management.NewScreenManager(runner, cfg.SessionName)
	return management.Backup(ctx, cfg.Dir, cfg.MaxBackups, screen, output)
}

// SetupParkourCmd sets up the parkour world (first-time setup).
type SetupParkourCmd struct{}

// Run sets up the parkour world.
func (cmd *SetupParkourCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := globalsToConfig(globals)
	screen := management.NewScreenManager(runner, cfg.SessionName)

	if !management.IsServerRunning(ctx, screen, runner, cfg.Port) {
		return fmt.Errorf("server not running — start it first with: mc-dad-server start")
	}

	output.Info("Setting up parkour world...")
	cmds := []string{
		"mv create parkour normal --world-type flat --no-structures",
	}
	for _, c := range cmds {
		if err := screen.SendCommand(ctx, c); err != nil {
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
		if err := screen.SendCommand(ctx, c); err != nil {
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
	screen := management.NewScreenManager(runner, cfg.SessionName)

	if !management.IsServerRunning(ctx, screen, runner, cfg.Port) {
		output.Info("Server not running, skipping rotation")
		return nil
	}

	return management.RotateParkour(ctx, cfg.Dir, screen, output)
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
	screen := management.NewScreenManager(runner, cfg.SessionName)

	if !management.IsServerRunning(ctx, screen, runner, cfg.Port) {
		return fmt.Errorf("server not running — start it first with: mc-dad-server start")
	}

	result, err := vote.RunVote(ctx, vote.Config{
		Maps:       management.ParkourMaps,
		Duration:   time.Duration(cmd.Duration) * time.Second,
		MaxChoices: cmd.Choices,
		ServerDir:  cfg.Dir,
		Screen:     screen,
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
