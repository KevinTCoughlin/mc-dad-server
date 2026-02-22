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
func StartServer(ctx context.Context, screen *ScreenManager, runner platform.CommandRunner, port int, dir, sessionName string, output *ui.UI) (bool, error) {
	if IsServerRunning(ctx, screen, runner, port) {
		output.Warn("Server is already running!")
		return true, nil
	}

	output.Info("Starting Minecraft server in screen session '%s'...", sessionName)
	if err := screen.Start(ctx, "bash", dir+"/start.sh"); err != nil {
		return false, fmt.Errorf("starting server: %w", err)
	}
	output.Success("Server started!")
	return false, nil
}

// StopServer gracefully stops the Minecraft server.
// It prints status messages to output and returns any error encountered.
func StopServer(ctx context.Context, screen *ScreenManager, runner platform.CommandRunner, port int, output *ui.UI) error {
	if !IsServerRunning(ctx, screen, runner, port) {
		output.Info("No running Minecraft server found.")
		return nil
	}

	output.Info("Sending shutdown command...")
	if err := screen.SendCommand(ctx, "say Server shutting down in 10 seconds..."); err != nil {
		return err
	}
	if err := Sleep(ctx, 10); err != nil {
		return err
	}
	if err := screen.SendCommand(ctx, "stop"); err != nil {
		return err
	}
	output.Success("Stop command sent. Server shutting down...")
	return nil
}

// PrintStatus prints the server status and resource usage to output.
func PrintStatus(ctx context.Context, screen *ScreenManager, runner platform.CommandRunner, port int, sessionName string, output *ui.UI) {
	output.Step("Minecraft Server Status")

	stats, err := GetProcessStats(ctx, runner)

	if screen.IsRunning(ctx) {
		output.Info("  Status:  RUNNING")
		output.Info("  Session: screen -r %s", sessionName)
	} else if err == nil && stats.PID > 0 {
		output.Info("  Status:  RUNNING (pid %d)", stats.PID)
	} else if IsPortListening(port) {
		output.Info("  Status:  RUNNING (port %d)", port)
	} else {
		output.Info("  Status:  STOPPED")
	}
	output.Info("")

	if err == nil && stats.PID > 0 {
		output.Info("  PID:     %d", stats.PID)
		output.Info("  Memory:  %s", stats.Memory)
		output.Info("  CPU:     %s", stats.CPU)
	}
}
