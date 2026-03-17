package bun

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

const (
	// minBunVersion is the minimum Bun version required for performance improvements.
	// Bun 1.3+ includes: structuredClone 25x faster, Buffer.slice 1.8x faster,
	// path.parse 7x faster, updated JavaScriptCore, and other optimizations.
	minBunVersion = "1.3"
)

// InstallBun ensures Bun is installed on the system.
func InstallBun(ctx context.Context, runner platform.CommandRunner, plat *platform.Platform, output *ui.UI) error {
	output.Step("Installing Bun Runtime")

	if runner.CommandExists("bun") {
		ver, err := bunVersion(ctx, runner)
		if err == nil {
			if isBunVersionSupported(ver) {
				output.Success("Bun %s already installed", ver)
				return nil
			}
			output.Info("Bun %s is installed but version %s+ is recommended for performance improvements", ver, minBunVersion)
			output.Info("Upgrading Bun...")
		}
	} else {
		output.Info("Installing Bun...")
	}

	err := runner.Run(ctx, "bash", "-c", "curl -fsSL https://bun.sh/install | bash")
	if err != nil {
		return fmt.Errorf("installing Bun: %w", err)
	}

	// Verify installation
	ver, err := bunVersion(ctx, runner)
	if err != nil {
		return fmt.Errorf("bun installation verification failed: %w", err)
	}
	if !isBunVersionSupported(ver) {
		output.Warn("Bun %s installed, but %s+ recommended for best performance", ver, minBunVersion)
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

// isBunVersionSupported checks if the installed Bun version meets the minimum requirement.
// It parses semantic versions like "1.3.10" and compares major.minor against minBunVersion.
func isBunVersionSupported(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	minParts := strings.Split(minBunVersion, ".")
	minMajor, _ := strconv.Atoi(minParts[0])
	minMinor, _ := strconv.Atoi(minParts[1])

	if major > minMajor {
		return true
	}
	if major == minMajor && minor >= minMinor {
		return true
	}
	return false
}
