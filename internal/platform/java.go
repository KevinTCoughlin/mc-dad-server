package platform

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// minJavaVersion is the runtime Paper 26.3 requires.
const minJavaVersion = 25

// temurin25URL is the manual install page named in the installer message.
const temurin25URL = "https://adoptium.net/temurin/releases/?version=25"

// previousJavaReleaseURL is the last published release that installs Java 21
// for Minecraft 1.21-era servers. Paper 26.3 does not run on that JDK.
const previousJavaReleaseURL = "https://github.com/KevinTCoughlin/mc-dad-server/releases/tag/v0.2.0"

// paperJavaRequirement explains why Java 25 is required and how to stay on
// the previous release. foundOK is false when java is missing or unreadable.
func paperJavaRequirement(found int, foundOK bool) string {
	have := "Java was not found on PATH."
	if foundOK {
		have = fmt.Sprintf("This machine has Java %d.", found)
	}
	return fmt.Sprintf(
		"Paper 26.3 requires Java %d (Eclipse Temurin). %s The installer will install Temurin %d from %s. To stay on Java 21 and Minecraft 1.21, use the previous release v0.2.0: %s",
		minJavaVersion, have, minJavaVersion, temurin25URL, previousJavaReleaseURL,
	)
}

// InstallJava ensures Java 25+ is available, installing Temurin 25 when it is not.
func InstallJava(ctx context.Context, runner CommandRunner, plat *Platform, output *ui.UI) error {
	output.Step("Installing Java (Adoptium Temurin)")

	found, foundOK := 0, false
	if runner.CommandExists("java") {
		ver, err := javaVersion(ctx, runner)
		if err == nil && ver >= minJavaVersion {
			output.Success("Java %d already installed (Paper 26.3 requires Eclipse Temurin %d+)", ver, minJavaVersion)
			return nil
		}
		if err == nil {
			found, foundOK = ver, true
		}
		output.Warn("%s", paperJavaRequirement(found, foundOK))
	} else {
		output.Warn("%s", paperJavaRequirement(0, false))
	}

	output.Info("Installing Adoptium Temurin JDK %d...", minJavaVersion)
	var err error
	switch plat.PkgMgr {
	case "apt":
		err = installJavaAPT(ctx, runner, output)
	case "dnf":
		err = installJavaDNF(ctx, runner, output)
	case "pacman":
		err = runner.RunSudo(ctx, "pacman", "-S", "--noconfirm", "jre-openjdk-headless")
	case "brew":
		err = runner.Run(ctx, "brew", "install", "--cask", "temurin@25")
	default:
		err = installJavaSDKMAN(ctx, runner, output)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", paperJavaRequirement(found, foundOK), err)
	}

	// Verify the JDK that is now first on PATH. A leftover Java 21 install
	// must not be treated as success.
	ver, verErr := javaVersion(ctx, runner)
	if verErr != nil || ver < minJavaVersion {
		got := "an unreadable java version"
		if verErr == nil {
			got = fmt.Sprintf("Java %d", ver)
		}
		return fmt.Errorf("java installation verification failed: Paper 26.3 requires Java %d (Eclipse Temurin) but found %s. Install Temurin %d from %s, or stay on Java 21 and Minecraft 1.21 with v0.2.0: %s", minJavaVersion, got, minJavaVersion, temurin25URL, previousJavaReleaseURL)
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

var osReleaseCodename = regexp.MustCompile(`(?m)^VERSION_CODENAME=["']?([A-Za-z0-9._-]+)["']?\s*$`)

func installJavaAPT(ctx context.Context, runner CommandRunner, output *ui.UI) error {
	output.Info("Adding the Adoptium APT repository...")
	if err := runner.RunSudo(ctx, "apt-get", "update", "-qq"); err != nil {
		return err
	}
	if err := runner.RunSudo(ctx, "apt-get", "install", "-y", "-qq", "wget", "ca-certificates", "gpg"); err != nil {
		return err
	}

	codename, err := debianCodename(ctx, runner)
	if err != nil {
		output.Warn("Could not read the Debian codename (%v); falling back to distro OpenJDK 25", err)
		return runner.RunSudo(ctx, "apt-get", "install", "-y", "-qq", "openjdk-25-jre-headless")
	}

	// The signed Adoptium repo is required on Debian 12: temurin-25-jdk is not
	// in the default archives, and OpenJDK 25 is not either.
	if err := runner.RunSudo(ctx, "bash", "-c", adoptiumAptScript(codename)); err != nil {
		output.Warn("Adoptium repository install failed (%v); falling back to distro OpenJDK 25", err)
		return runner.RunSudo(ctx, "apt-get", "install", "-y", "-qq", "openjdk-25-jre-headless")
	}
	output.Success("Adoptium Temurin 25 installed via APT")
	return nil
}

func debianCodename(ctx context.Context, runner CommandRunner) (string, error) {
	out, err := runner.RunWithOutput(ctx, "cat", "/etc/os-release")
	if err != nil {
		return "", err
	}
	m := osReleaseCodename.FindSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("VERSION_CODENAME missing from /etc/os-release")
	}
	return string(m[1]), nil
}

// adoptiumAptScript adds the signed Adoptium repository and installs Temurin 25.
// codename must already be restricted to a single release token.
func adoptiumAptScript(codename string) string {
	return fmt.Sprintf(`set -euo pipefail
rm -f /etc/apt/trusted.gpg.d/adoptium.gpg
wget -qO - https://packages.adoptium.net/artifactory/api/gpg/key/public | gpg --dearmor -o /etc/apt/trusted.gpg.d/adoptium.gpg
printf 'deb https://packages.adoptium.net/artifactory/deb %s main\n' > /etc/apt/sources.list.d/adoptium.list
apt-get update -qq
apt-get install -y -qq temurin-25-jdk
`, codename)
}

func installJavaDNF(ctx context.Context, runner CommandRunner, output *ui.UI) error {
	if err := runner.RunSudo(ctx, "dnf", "install", "-y", "-q", "temurin-25-jdk"); err != nil {
		output.Warn("Adoptium repo unavailable, falling back to distro OpenJDK")
		return runner.RunSudo(ctx, "dnf", "install", "-y", "-q", "java-25-openjdk-headless")
	}
	output.Success("Adoptium Temurin 25 installed via DNF")
	return nil
}

func installJavaSDKMAN(ctx context.Context, runner CommandRunner, output *ui.UI) error {
	output.Warn("Using SDKMAN to install Temurin Java...")
	if err := runner.Run(ctx, "bash", "-c",
		`curl -fsSL "https://get.sdkman.io" | bash && source "$HOME/.sdkman/bin/sdkman-init.sh" && sdk install java 25.0.5-tem`); err != nil {
		return fmt.Errorf("SDKMAN install failed: %w", err)
	}
	return nil
}
