package platform

import (
	"context"
	"fmt"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// InstallPackage installs a system package using the detected package manager.
func InstallPackage(ctx context.Context, runner CommandRunner, plat Platform, pkg string, output *ui.UI) error {
	if runner.CommandExists(pkg) {
		output.Success("%s already installed", pkg)
		return nil
	}

	output.Info("Installing %s...", pkg)
	var err error
	switch plat.PkgMgr {
	case "apt":
		err = runner.RunSudo(ctx, "apt-get", "update", "-qq")
		if err == nil {
			err = runner.RunSudo(ctx, "apt-get", "install", "-y", "-qq", pkg)
		}
	case "dnf":
		err = runner.RunSudo(ctx, "dnf", "install", "-y", "-q", pkg)
	case "pacman":
		err = runner.RunSudo(ctx, "pacman", "-S", "--noconfirm", "--quiet", pkg)
	case "zypper":
		err = runner.RunSudo(ctx, "zypper", "install", "-y", pkg)
	case "brew":
		err = runner.Run(ctx, "brew", "install", pkg)
	default:
		return fmt.Errorf("cannot install %s: unknown package manager %s", pkg, plat.PkgMgr)
	}
	if err != nil {
		return fmt.Errorf("installing %s: %w", pkg, err)
	}
	output.Success("%s installed", pkg)
	return nil
}
