package management

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// StartServer starts the Minecraft server if not already running.
// It prints status messages to output. It returns true if the server was
// already running (no action taken), false if it was freshly started.
// Returns an error if the start attempt fails.
func StartServer(ctx context.Context, mgr ServerManager, runner platform.CommandRunner, port int, dir, sessionName string, output *ui.UI) (bool, error) {
	if IsServerRunning(ctx, mgr, runner, port) {
		output.Warn("Server is already running!")
		return true, nil
	}

	output.Info("Starting Minecraft server in screen session '%s'...", sessionName)
	if err := mgr.Start(ctx, "bash", filepath.Join(dir, "start.sh")); err != nil {
		return false, fmt.Errorf("starting server: %w", err)
	}
	output.Success("Server started!")
	return false, nil
}

// shutdownStep defines a countdown message and the pause before it.
type shutdownStep struct {
	message string
	delay   int // seconds to sleep after sending
}

// shutdownCountdown is the multi-step graceful shutdown sequence.
var shutdownCountdown = []shutdownStep{
	{message: "say [SERVER] Shutting down in 30 seconds...", delay: 20},
	{message: "say [SERVER] Shutting down in 10 seconds...", delay: 5},
	{message: "say [SERVER] Shutting down in 5 seconds...", delay: 3},
	{message: "say [SERVER] Shutting down in 2 seconds...", delay: 1},
	{message: "say [SERVER] Shutting down in 1 second...", delay: 1},
	{message: "say [SERVER] Goodbye!", delay: 0},
}

// StopServer gracefully stops the Minecraft server with a multi-step countdown.
// It prints status messages to output and returns any error encountered.
func StopServer(ctx context.Context, mgr ServerManager, runner platform.CommandRunner, port int, output *ui.UI) error {
	if !IsServerRunning(ctx, mgr, runner, port) {
		output.Info("No running Minecraft server found.")
		return nil
	}

	output.Info("Starting graceful shutdown (30s countdown)...")
	for _, step := range shutdownCountdown {
		if err := mgr.SendCommand(ctx, step.message); err != nil {
			output.Warn("Failed to send countdown message: %s", err)
			break
		}
		if step.delay > 0 {
			if err := Sleep(ctx, step.delay); err != nil {
				return err
			}
		}
	}

	output.Info("Sending stop command...")
	if err := mgr.SendCommand(ctx, "stop"); err != nil {
		return err
	}
	output.Success("Stop command sent. Server shutting down...")
	return nil
}

// PrintStatus prints the server status and resource usage to output.
func PrintStatus(ctx context.Context, mgr ServerManager, runner platform.CommandRunner, port int, sessionName string, output *ui.UI) {
	output.Step("Minecraft Server Status")

	stats, err := GetProcessStats(ctx, runner)

	switch {
	case mgr.IsRunning(ctx):
		output.Info("  Status:  RUNNING")
		output.Info("  Session: screen -r %s", sessionName)
	case err == nil && stats.PID > 0:
		output.Info("  Status:  RUNNING (pid %d)", stats.PID)
	case IsPortListening(port):
		output.Info("  Status:  RUNNING (port %d)", port)
	default:
		output.Info("  Status:  STOPPED")
	}
	output.Info("")

	if err == nil && stats.PID > 0 {
		output.Info("  PID:     %d", stats.PID)
		output.Info("  Memory:  %s", stats.Memory)
		output.Info("  CPU:     %s", stats.CPU)
	}
}
