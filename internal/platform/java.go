package platform

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// InstallJava ensures Java 21+ is available, installing if necessary.
func InstallJava(ctx context.Context, runner CommandRunner, plat *Platform, output *ui.UI) error {
	output.Step("Installing Java (Adoptium Temurin)")

	if runner.CommandExists("java") {
		ver, err := javaVersion(ctx, runner)
		if err == nil && ver >= 21 {
			output.Success("Java %d already installed", ver)
			return nil
		}
		if err == nil {
			output.Warn("Java %d found, but 21+ required", ver)
		}
	}

	output.Info("Installing Adoptium Temurin JDK 21...")
	var err error
	switch plat.PkgMgr {
	case "apt":
		err = installJavaAPT(ctx, runner, output)
	case "dnf":
		err = installJavaDNF(ctx, runner, output)
	case "pacman":
		err = runner.RunSudo(ctx, "pacman", "-S", "--noconfirm", "jre-openjdk-headless")
	case "brew":
		err = runner.Run(ctx, "brew", "install", "--cask", "temurin@21")
	default:
		err = installJavaSDKMAN(ctx, runner, output)
	}
	if err != nil {
		return fmt.Errorf("installing Java: %w", err)
	}

	// Verify
	ver, verErr := javaVersion(ctx, runner)
	if verErr != nil || ver < 21 {
		return fmt.Errorf("Java installation verification failed")
	}
	output.Success("Java %d installed successfully", ver)
	return nil
}

func javaVersion(ctx context.Context, runner CommandRunner) (int, error) {
	out, err := runner.RunWithOutput(ctx, "java", "-version")
	if err != nil {
		return 0, err
	}
	// java -version outputs to stderr, but RunWithOutput captures stdout
	// Some implementations capture both. Parse the version string.
	line := strings.Split(string(out), "\n")[0]
	// Look for quoted version string like "21.0.2"
	parts := strings.Split(line, "\"")
	if len(parts) < 2 {
		return 0, fmt.Errorf("cannot parse java version: %s", line)
	}
	verStr := parts[1]
	major := strings.Split(verStr, ".")[0]
	return strconv.Atoi(major)
}

func installJavaAPT(ctx context.Context, runner CommandRunner, output *ui.UI) error {
	// Try Adoptium repo first
	output.Info("Adding Adoptium APT repository...")
	err := runner.RunSudo(ctx, "apt-get", "update", "-qq")
	if err != nil {
		return err
	}
	err = runner.RunSudo(ctx, "apt-get", "install", "-y", "-qq", "wget", "apt-transport-https", "gpg")
	if err != nil {
		return err
	}

	if err := runner.RunSudo(ctx, "apt-get", "install", "-y", "-qq", "temurin-21-jdk"); err != nil {
		output.Warn("Adoptium repo unavailable, falling back to distro OpenJDK")
		return runner.RunSudo(ctx, "apt-get", "install", "-y", "-qq", "openjdk-21-jre-headless")
	}
	output.Success("Adoptium Temurin 21 installed via APT")
	return nil
}

func installJavaDNF(ctx context.Context, runner CommandRunner, output *ui.UI) error {
	if err := runner.RunSudo(ctx, "dnf", "install", "-y", "-q", "temurin-21-jdk"); err != nil {
		output.Warn("Adoptium repo unavailable, falling back to distro OpenJDK")
		return runner.RunSudo(ctx, "dnf", "install", "-y", "-q", "java-21-openjdk-headless")
	}
	output.Success("Adoptium Temurin 21 installed via DNF")
	return nil
}

func installJavaSDKMAN(ctx context.Context, runner CommandRunner, output *ui.UI) error {
	output.Warn("Using SDKMAN to install Temurin Java...")
	if err := runner.Run(ctx, "bash", "-c",
		`curl -fsSL "https://get.sdkman.io" | bash && source "$HOME/.sdkman/bin/sdkman-init.sh" && sdk install java 21.0.2-tem`); err != nil {
		return fmt.Errorf("SDKMAN install failed: %w", err)
	}
	return nil
}
