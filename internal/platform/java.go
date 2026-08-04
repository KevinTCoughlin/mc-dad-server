package platform

import (
	"context"
	"fmt"
	"regexp"
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
		return fmt.Errorf("java installation verification failed")
	}
	output.Success("Java %d installed successfully", ver)
	return nil
}

// javaVersionPattern matches the version number in the first line of both
// `java --version` ("openjdk 21.0.2 2024-01-16") and the legacy quoted form
// (`openjdk version "21.0.2"` / `java version "1.8.0_392"`).
var javaVersionPattern = regexp.MustCompile(`(\d+)(?:\.(\d+))?(?:[._][\w.-]+)?`)

func javaVersion(ctx context.Context, runner CommandRunner) (int, error) {
	// `java -version` writes to stderr, which RunWithOutput discards. The
	// JDK 9+ `--version` form writes the same information to stdout.
	out, err := runner.RunWithOutput(ctx, "java", "--version")
	if err != nil {
		return 0, err
	}
	return parseJavaVersion(string(out))
}

// parseJavaVersion extracts the Java major version from `java --version` output.
// Legacy 1.x version strings (e.g. "1.8.0_392") map to their second component.
func parseJavaVersion(out string) (int, error) {
	line := strings.TrimSpace(strings.Split(out, "\n")[0])
	// Strip quotes so the legacy `java version "1.8.0_392"` form parses too.
	line = strings.ReplaceAll(line, `"`, " ")

	for _, field := range strings.Fields(line) {
		m := javaVersionPattern.FindStringSubmatch(field)
		if len(m) == 0 || m[0] != field {
			continue
		}
		major, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if major == 1 && m[2] != "" {
			// "1.8.0_392" — the real major version is the second component.
			return strconv.Atoi(m[2])
		}
		return major, nil
	}

	return 0, fmt.Errorf("cannot parse java version: %s", line)
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
