package tunnel

import (
	"context"
	"fmt"
	"runtime"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// InstallPlayit downloads and installs the playit.gg agent binary.
func InstallPlayit(ctx context.Context, runner platform.CommandRunner, plat platform.Platform, output *ui.UI) error {
	output.Step("Setting Up playit.gg (No Port Forwarding Needed!)")

	fmt.Println("playit.gg lets your kids' friends connect WITHOUT port forwarding.")
	fmt.Println("This is the safest and easiest way to make your server accessible.")
	fmt.Println()

	if runner.CommandExists("playit") {
		output.Success("playit.gg already installed")
		return nil
	}

	output.Info("Installing playit.gg agent...")

	arch, err := playitArch()
	if err != nil {
		output.Warn("%v", err)
		return nil
	}

	osName := "linux"
	if plat.OS == "macos" {
		osName = "darwin"
	}

	url := fmt.Sprintf("https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-%s-%s",
		osName, arch)

	// Download to temp location
	if err := runner.Run(ctx, "curl", "-fsSL", url, "-o", "/tmp/playit"); err != nil {
		output.Warn("Could not auto-install playit.gg. Visit https://playit.gg to install manually.")
		return nil
	}

	if err := runner.RunSudo(ctx, "install", "-m", "755", "/tmp/playit", "/usr/local/bin/playit"); err != nil {
		output.Warn("Could not install playit to /usr/local/bin. Visit https://playit.gg")
		return nil
	}

	runner.Run(ctx, "rm", "-f", "/tmp/playit")
	output.Success("playit.gg installed")
	return nil
}

func playitArch() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "aarch64", nil
	case "arm":
		return "armv7", nil
	default:
		return "", fmt.Errorf("unsupported architecture for playit.gg: %s", runtime.GOARCH)
	}
}
