package console

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
	"github.com/KevinTCoughlin/mc-dad-server/internal/vote"
)

// sentinel value returned by dispatch to signal a viewport clear.
const clearSentinel = "\x00CLEAR"

// dispatch parses input and runs the corresponding command, capturing output
// into a string. Returns the output text and whether the console should quit.
func dispatch(ctx context.Context, input string, opts *Options, runner platform.CommandRunner) (string, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", false
	}

	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	cfg := optsToConfig(opts)
	screen := management.NewScreenManager(runner, cfg.SessionName)

	var buf bytes.Buffer
	output := ui.NewWriter(&buf, false)

	switch cmd {
	case "start":
		if screen.IsRunning(ctx) {
			output.Warn("Server is already running! Use: screen -r %s", cfg.SessionName)
		} else {
			output.Info("Starting Minecraft server in screen session '%s'...", cfg.SessionName)
			if err := screen.Start(ctx, "bash", cfg.Dir+"/start.sh"); err != nil {
				output.Warn("Starting server: %s", err)
			} else {
				output.Success("Server started!")
			}
		}

	case "stop":
		if !screen.IsRunning(ctx) {
			output.Info("No running Minecraft server found.")
		} else {
			output.Info("Sending shutdown command...")
			if err := screen.SendCommand(ctx, "say Server shutting down in 10 seconds..."); err != nil {
				output.Warn("%s", err)
				break
			}
			if err := management.Sleep(ctx, 10); err != nil {
				output.Warn("%s", err)
				break
			}
			if err := screen.SendCommand(ctx, "stop"); err != nil {
				output.Warn("%s", err)
				break
			}
			output.Success("Stop command sent. Server shutting down...")
		}

	case "status":
		output.Step("Minecraft Server Status")
		if screen.IsRunning(ctx) {
			output.Info("  Status:  RUNNING")
			output.Info("  Session: screen -r %s", cfg.SessionName)
		} else if management.IsPortListening(cfg.Port) {
			output.Info("  Status:  RUNNING (port %d)", cfg.Port)
		} else {
			output.Info("  Status:  STOPPED")
		}
		output.Info("")
		stats, err := management.GetProcessStats(ctx, runner)
		if err == nil && stats.PID > 0 {
			output.Info("  PID:     %d", stats.PID)
			output.Info("  Memory:  %s", stats.Memory)
			output.Info("  CPU:     %s", stats.CPU)
		}

	case "backup":
		if err := management.Backup(ctx, cfg.Dir, cfg.MaxBackups, screen, output); err != nil {
			output.Warn("Backup failed: %s", err)
		}

	case "rotate-parkour":
		if !screen.IsRunning(ctx) {
			output.Info("Server not running, skipping rotation")
		} else if err := management.RotateParkour(ctx, cfg.Dir, screen, output); err != nil {
			output.Warn("Rotation failed: %s", err)
		}

	case "vote-map":
		if !screen.IsRunning(ctx) {
			output.Warn("Server not running — start it first")
		} else {
			result, err := vote.RunVote(ctx, vote.Config{
				Maps:       management.ParkourMaps,
				Duration:   time.Duration(cfg.VoteDuration) * time.Second,
				MaxChoices: cfg.VoteChoices,
				ServerDir:  cfg.Dir,
				Screen:     screen,
				Output:     output,
			})
			if err != nil {
				output.Warn("Vote failed: %s", err)
			} else {
				output.Success("Map vote complete: %s (%d voters)", result.Winner, result.Voters)
			}
		}

	case "say":
		if len(args) == 0 {
			output.Warn("Usage: say <message>")
		} else {
			msg := strings.Join(args, " ")
			if err := screen.SendCommand(ctx, "say "+msg); err != nil {
				output.Warn("%s", err)
			} else {
				output.Success("Sent: say %s", msg)
			}
		}

	case "cmd":
		if len(args) == 0 {
			output.Warn("Usage: cmd <raw minecraft command>")
		} else {
			raw := strings.Join(args, " ")
			if err := screen.SendCommand(ctx, raw); err != nil {
				output.Warn("%s", err)
			} else {
				output.Success("Sent: %s", raw)
			}
		}

	case "help":
		return helpText(), false

	case "clear":
		return clearSentinel, false

	case "exit", "quit":
		return "", true

	default:
		return fmt.Sprintf("Unknown command: %s (type 'help' for available commands)", cmd), false
	}

	return strings.TrimRight(buf.String(), "\n"), false
}

func helpText() string {
	return `Available commands:
  start           Start the Minecraft server
  stop            Gracefully stop the server
  status          Show server status and resource usage
  backup          Backup world data
  rotate-parkour  Rotate the featured parkour map
  vote-map        Start a map vote
  say <msg>       Broadcast a message to players
  cmd <raw>       Send a raw command to the server console
  clear           Clear the console
  help            Show this help
  exit / quit     Exit the console`
}

func optsToConfig(o *Options) *config.ServerConfig {
	cfg := config.DefaultConfig()
	cfg.Dir = o.Dir
	cfg.SessionName = o.Session
	return cfg
}
