package management

import (
	"context"
	"fmt"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// StartServer starts the Minecraft server if not already running.
// It prints status messages to output. It returns true if the server was
// already running (no action taken), false if it was freshly started.
// Returns an error if the start attempt fails.
func StartServer(ctx context.Context, mgr ServerManager, runner platform.CommandRunner, port int, sessionName string, output *ui.UI) (bool, error) {
	if IsServerRunning(ctx, mgr, runner, port) {
		output.Warn("Server is already running!")
		return true, nil
	}

	output.Info("Starting Minecraft server...")
	if err := mgr.Launch(ctx); err != nil {
		return false, fmt.Errorf("starting server: %w", err)
	}
	output.Success("Server started!")
	return false, nil
}

// StopServer gracefully stops the Minecraft server.
// It attempts a graceful shutdown via SendCommand, falling back to
// mgr.Stop() if the console path is unavailable.
func StopServer(ctx context.Context, mgr ServerManager, runner platform.CommandRunner, port int, output *ui.UI) error {
	if !IsServerRunning(ctx, mgr, runner, port) {
		output.Info("No running Minecraft server found.")
		return nil
	}

	output.Info("Sending shutdown command...")

	// Try graceful in-game countdown + stop command.
	sayErr := mgr.SendCommand(ctx, "say Server shutting down in 10 seconds...")
	if sayErr == nil {
		if err := Sleep(ctx, 10); err != nil {
			return err
		}
		if err := mgr.SendCommand(ctx, "stop"); err == nil {
			output.Success("Stop command sent. Server shutting down...")
			return nil
		}
	}

	// Fallback to manager-level stop (e.g. podman stop).
	output.Info("Console unavailable, using manager stop...")
	if err := mgr.Stop(ctx); err != nil {
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
