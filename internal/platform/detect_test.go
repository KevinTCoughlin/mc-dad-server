package platform

import (
	"context"
	"runtime"
	"testing"
)

func TestDetect_CurrentPlatform(t *testing.T) {
	m := NewMockRunner()
	p := Detect(context.Background(), m)

	switch runtime.GOOS {
	case "darwin":
		if p.OS != "macos" {
			t.Fatalf("expected macos, got %s", p.OS)
		}
		if p.PkgMgr != "brew" {
			t.Fatalf("expected brew, got %s", p.PkgMgr)
		}
		if p.InitSystem != "launchd" {
			t.Fatalf("expected launchd, got %s", p.InitSystem)
		}
	case "linux":
		if p.OS != "linux" {
			t.Fatalf("expected linux, got %s", p.OS)
		}
	}

	if p.Arch == "" {
		t.Fatal("arch should not be empty")
	}
}

func TestDetect_LinuxDistroAPT(t *testing.T) {
	m := NewMockRunner()
	m.ExistsMap["apt-get"] = true
	m.ExistsMap["systemctl"] = true
	m.OutputMap[m.key("cat", "/proc/version")] = []byte("Linux version 5.15.0 (gcc)")

	if runtime.GOOS != "linux" {
		t.Skip("skipping linux-specific test on non-linux")
	}

	p := Detect(context.Background(), m)
	if p.Distro != "debian" {
		t.Fatalf("expected debian, got %s", p.Distro)
	}
	if p.PkgMgr != "apt" {
		t.Fatalf("expected apt, got %s", p.PkgMgr)
	}
	if p.InitSystem != "systemd" {
		t.Fatalf("expected systemd, got %s", p.InitSystem)
	}
}

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"amd64", "amd64"},
		{"arm64", "arm64"},
		{"arm", "armv7"},
		{"386", "386"},
	}
	for _, tc := range tests {
		got := normalizeArch(tc.in)
		if got != tc.want {
			t.Errorf("normalizeArch(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPlatform_IsLinux(t *testing.T) {
	tests := []struct {
		os   string
		want bool
	}{
		{"linux", true},
		{"wsl", true},
		{"macos", false},
		{"windows", false},
	}
	for _, tc := range tests {
		p := Platform{OS: tc.os}
		if got := p.IsLinux(); got != tc.want {
			t.Errorf("Platform{OS: %q}.IsLinux() = %v, want %v", tc.os, got, tc.want)
		}
	}
}

func TestDetectContainerRuntime(t *testing.T) {
	tests := []struct {
		name        string
		hasPodman   bool
		hasDocker   bool
		wantRuntime string
	}{
		{
			name:        "podman only",
			hasPodman:   true,
			hasDocker:   false,
			wantRuntime: "podman",
		},
		{
			name:        "docker only",
			hasPodman:   false,
			hasDocker:   true,
			wantRuntime: "docker",
		},
		{
			name:        "both available (prefers podman)",
			hasPodman:   true,
			hasDocker:   true,
			wantRuntime: "podman",
		},
		{
			name:        "neither available",
			hasPodman:   false,
			hasDocker:   false,
			wantRuntime: "unknown",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewMockRunner()
			if tc.hasPodman {
				m.ExistsMap["podman"] = true
			}
			if tc.hasDocker {
				m.ExistsMap["docker"] = true
			}

			got := detectContainerRuntime(m)
			if got != tc.wantRuntime {
				t.Errorf("detectContainerRuntime() = %q, want %q", got, tc.wantRuntime)
			}
		})
	}
}
