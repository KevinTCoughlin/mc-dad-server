package bun

import (
	"context"
	"fmt"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// InstallBun ensures Bun is installed on the system.
func InstallBun(ctx context.Context, runner platform.CommandRunner, plat *platform.Platform, output *ui.UI) error {
	output.Step("Installing Bun Runtime")

	if runner.CommandExists("bun") {
		ver, err := bunVersion(ctx, runner)
		if err == nil {
			output.Success("Bun %s already installed", ver)
			return nil
		}
	}

	output.Info("Installing Bun...")
	err := runner.Run(ctx, "bash", "-c", "curl -fsSL https://bun.sh/install | bash")
	if err != nil {
		return fmt.Errorf("installing Bun: %w", err)
	}

	// Verify installation
	ver, err := bunVersion(ctx, runner)
	if err != nil {
		return fmt.Errorf("bun installation verification failed: %w", err)
	}
	output.Success("Bun %s installed successfully", ver)
	return nil
}

// bunVersion returns the installed Bun version string.
func bunVersion(ctx context.Context, runner platform.CommandRunner) (string, error) {
	out, err := runner.RunWithOutput(ctx, "bun", "--version")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
