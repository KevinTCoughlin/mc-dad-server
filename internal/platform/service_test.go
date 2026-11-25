package platform

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
)

func TestSystemdUnitHardening(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Dir = "/var/lib/minecraft"
	cfg.SessionName = "minecraft"

	unit := systemdUnit(serviceUser, cfg)
	for _, want := range []string{
		"User=minecraft",
		"Group=minecraft",
		"NoNewPrivileges=true",
		"PrivateTmp=true",
		"ProtectSystem=strict",
		"Restart=on-failure",
		`ReadWritePaths="/var/lib/minecraft"`,
		`WorkingDirectory="/var/lib/minecraft"`,
		`ExecStart=/usr/bin/bash "/var/lib/minecraft/start.sh"`,
	} {
		if !strings.Contains(unit, want) {
			t.Errorf("unit missing %s\n%s", want, unit)
		}
	}
}

func TestSystemdUnitQuotesSpaces(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	cfg.Dir = "/srv/dad's server"
	cfg.SessionName = "minecraft"
	unit := systemdUnit("minecraft", cfg)
	if !strings.Contains(unit, `WorkingDirectory="/srv/dad's server"`) {
		t.Fatalf("unit did not quote the directory:\n%s", unit)
	}
}

func TestEnsureServicePathTraversable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not enforce Unix directory modes; the systemd unit is Linux-only")
	}
	root := t.TempDir()
	chmodAncestorsTraversable(t, root)
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatalf("chmod root: %v", err)
	}

	blocked := filepath.Join(root, "private")
	if err := os.Mkdir(blocked, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dir := filepath.Join(blocked, "server")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir server: %v", err)
	}

	err := ensureServicePathTraversable(dir)
	if err == nil {
		t.Fatal("expected a traversal error")
	}
	if !strings.Contains(err.Error(), "minecraft") || !strings.Contains(err.Error(), "--dir") {
		t.Fatalf("error = %q", err)
	}

	open := filepath.Join(root, "open")
	if err := os.Mkdir(open, 0o755); err != nil {
		t.Fatalf("mkdir open: %v", err)
	}
	// Mkdir's mode is cut down by the umask: under 0077 this directory would
	// be 0700 and the check would be right to refuse it.
	if err := os.Chmod(open, 0o755); err != nil {
		t.Fatalf("chmod open: %v", err)
	}
	if err := ensureServicePathTraversable(filepath.Join(open, "server")); err != nil {
		t.Fatalf("traversable path rejected: %v", err)
	}
}

// chmodAncestorsTraversable gives other-execute to parents of dir that this
// test created as 0700. Go's TempDir is private, and the dedicated minecraft
// user would not be able to walk into it either.
func chmodAncestorsTraversable(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Clean(dir)
	for {
		parent := filepath.Dir(path)
		if parent == path {
			return
		}
		info, err := os.Stat(parent)
		if err != nil {
			t.Fatalf("stat %s: %v", parent, err)
		}
		if info.Mode().Perm()&0o001 != 0 {
			return
		}
		if err := os.Chmod(parent, 0o755); err != nil {
			t.Fatalf("chmod %s: %v", parent, err)
		}
		path = parent
	}
}

func TestSystemdInstallCreatesDedicatedUser(t *testing.T) {
	runner := NewMockRunner()
	runner.ErrorMap["id [minecraft]"] = errors.New("no such user")

	cfg := config.DefaultConfig()
	cfg.Dir = t.TempDir()
	cfg.SessionName = "minecraft"
	chmodAncestorsTraversable(t, cfg.Dir)
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	mgr := &systemdManager{runner: runner, cfg: cfg}
	if err := mgr.Install(cfg); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	var sawUseradd, sawChown, sawReload bool
	for _, cmd := range runner.Commands {
		if !cmd.Sudo {
			continue
		}
		switch cmd.Name {
		case "useradd":
			sawUseradd = true
			joined := strings.Join(cmd.Args, " ")
			if !strings.Contains(joined, "--system") || !strings.Contains(joined, "minecraft") {
				t.Fatalf("useradd args = %v", cmd.Args)
			}
		case "chown":
			sawChown = true
			if len(cmd.Args) < 2 || cmd.Args[1] != "minecraft:minecraft" {
				t.Fatalf("chown args = %v", cmd.Args)
			}
		case "systemctl":
			if len(cmd.Args) > 0 && cmd.Args[0] == "daemon-reload" {
				sawReload = true
			}
		}
	}
	if !sawUseradd || !sawChown || !sawReload {
		t.Fatalf("useradd=%v chown=%v reload=%v commands=%v", sawUseradd, sawChown, sawReload, runner.Commands)
	}
}

func TestSystemdTemplatesKeepHardening(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"../../embedded/templates/systemd.service.tmpl",
		"../../cmd/mc-dad-server/embedded/templates/systemd.service.tmpl",
	} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		text := string(body)
		for _, want := range []string{
			"User={{.User}}",
			"Group={{.Group}}",
			"NoNewPrivileges=true",
			"PrivateTmp=true",
			"ProtectSystem=strict",
			"Restart=on-failure",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing %s", path, want)
			}
		}
	}
}
