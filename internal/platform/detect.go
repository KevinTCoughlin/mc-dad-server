package platform

import (
	"context"
	"runtime"
	"strings"
)

// Platform holds detected OS, distro, package manager, and init system.
type Platform struct {
	OS               string // linux, macos, wsl, windows
	Distro           string // debian, fedora, arch, suse, unknown
	PkgMgr           string // apt, dnf, pacman, zypper, brew, unknown
	InitSystem       string // systemd, launchd, unknown
	Arch             string // amd64, arm64, armv7
	ContainerRuntime string // podman, docker, unknown
}

// Detect probes the runtime environment and returns platform info.
func Detect(ctx context.Context, runner CommandRunner) Platform {
	p := Platform{
		OS:               "unknown",
		Distro:           "unknown",
		PkgMgr:           "unknown",
		InitSystem:       "unknown",
		Arch:             normalizeArch(runtime.GOARCH),
		ContainerRuntime: "unknown",
	}

	switch runtime.GOOS {
	case "linux":
		p.OS = "linux"
		// Check for WSL
		out, err := runner.RunWithOutput(ctx, "cat", "/proc/version")
		if err == nil && containsCI(string(out), "microsoft") {
			p.OS = "wsl"
		}
		p.detectLinuxDistro(runner)
		if runner.CommandExists("systemctl") {
			p.InitSystem = "systemd"
		}
	case "darwin":
		p.OS = "macos"
		p.PkgMgr = "brew"
		p.InitSystem = "launchd"
	case "windows":
		p.OS = "windows"
	}

	// Detect container runtime (podman preferred over docker)
	p.ContainerRuntime = detectContainerRuntime(runner)

	return p
}

func (p *Platform) detectLinuxDistro(runner CommandRunner) {
	switch {
	case runner.CommandExists("apt-get"):
		p.Distro = "debian"
		p.PkgMgr = "apt"
	case runner.CommandExists("dnf"):
		p.Distro = "fedora"
		p.PkgMgr = "dnf"
	case runner.CommandExists("pacman"):
		p.Distro = "arch"
		p.PkgMgr = "pacman"
	case runner.CommandExists("zypper"):
		p.Distro = "suse"
		p.PkgMgr = "zypper"
	}
}

func normalizeArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "arm":
		return "armv7"
	default:
		return goarch
	}
}

func containsCI(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// IsLinux returns true if the platform is linux or WSL.
func (p *Platform) IsLinux() bool {
	return p.OS == "linux" || p.OS == "wsl"
}

// detectContainerRuntime detects available container runtime (podman or docker).
// Prefers podman if both are available.
func detectContainerRuntime(runner CommandRunner) string {
	if runner.CommandExists("podman") {
		return "podman"
	}
	if runner.CommandExists("docker") {
		return "docker"
	}
	return "unknown"
}
