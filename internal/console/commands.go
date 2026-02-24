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
	var mgr management.ServerManager = management.NewScreenManager(runner, cfg.SessionName)

	var buf bytes.Buffer
	output := ui.NewWriter(&buf, false)

	running := management.IsServerRunning(ctx, mgr, runner, cfg.Port)

	switch cmd {
	case "start":
		if _, err := management.StartServer(ctx, mgr, runner, cfg.Port, cfg.Dir, cfg.SessionName, output); err != nil {
			output.Warn("Starting server: %s", err)
		}

	case "stop":
		if err := management.StopServer(ctx, mgr, runner, cfg.Port, output); err != nil {
			output.Warn("%s", err)
		}

	case "status":
		management.PrintStatus(ctx, mgr, runner, cfg.Port, cfg.SessionName, output)

	case "backup":
		if err := management.Backup(ctx, cfg.Dir, cfg.MaxBackups, mgr, output); err != nil {
			output.Warn("Backup failed: %s", err)
		}

	case "rotate-parkour":
		if !running {
			output.Info("Server not running, skipping rotation")
		} else if err := management.RotateParkour(ctx, cfg.Dir, mgr, output); err != nil {
			output.Warn("Rotation failed: %s", err)
		}

	case "vote-map":
		if !running {
			output.Warn("Server not running — start it first")
		} else {
			result, err := vote.RunVote(ctx, &vote.Config{
				Maps:       management.ParkourMaps,
				Duration:   time.Duration(cfg.VoteDuration) * time.Second,
				MaxChoices: cfg.VoteChoices,
				ServerDir:  cfg.Dir,
				Screen:     mgr,
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
			if err := mgr.SendCommand(ctx, "say "+msg); err != nil {
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
			if err := mgr.SendCommand(ctx, raw); err != nil {
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
