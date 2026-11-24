package platform

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// serviceUser is the dedicated account the systemd unit runs as.
const serviceUser = "minecraft"

// ServiceManager handles platform-specific service installation and management.
type ServiceManager interface {
	Install(cfg *config.ServerConfig) error
	Enable() error
	Start() error
	Stop() error
	Status() (string, error)
}

// NewServiceManager returns the appropriate service manager for the platform.
func NewServiceManager(plat *Platform, runner CommandRunner, cfg *config.ServerConfig) ServiceManager {
	switch {
	case plat.InitSystem == "systemd" && runtime.GOOS == "linux":
		return &systemdManager{runner: runner, cfg: cfg}
	case plat.InitSystem == "launchd" && runtime.GOOS == "darwin":
		home, _ := os.UserHomeDir()
		return &launchdManager{
			runner:    runner,
			cfg:       cfg,
			plistPath: filepath.Join(home, "Library", "LaunchAgents", "com.mc-dad-server.minecraft.plist"),
		}
	default:
		return nil
	}
}

// --- systemd ---

type systemdManager struct {
	runner CommandRunner
	cfg    *config.ServerConfig
}

func (m *systemdManager) Install(cfg *config.ServerConfig) error {
	output := ui.Default()
	output.Step("Setting Up Systemd Service")

	if err := ensureServicePathTraversable(cfg.Dir); err != nil {
		return err
	}

	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	ctx := context.Background()
	if err := ensureServiceUser(ctx, m.runner); err != nil {
		return err
	}

	unit := systemdUnit(serviceUser, cfg)
	unitPath := "/etc/systemd/system/minecraft.service"
	// Staged in a private temp file rather than a predictable /tmp path: the
	// file is handed to `sudo cp`, so a pre-planted symlink at a guessable
	// name would let a local user steer what root installs.
	tmpFile, err := writePrivateTemp("minecraft-*.service", []byte(unit), 0o644)
	if err != nil {
		return fmt.Errorf("writing temp service file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	if err := m.runner.RunSudo(ctx, "cp", tmpFile, unitPath); err != nil {
		return fmt.Errorf("installing service file: %w", err)
	}
	if err := m.runner.RunSudo(ctx, "chown", "-R", serviceUser+":"+serviceUser, cfg.Dir); err != nil {
		return fmt.Errorf("giving %s ownership of %s: %w", serviceUser, cfg.Dir, err)
	}
	// Group-writable so the operator, once in the minecraft group, can still
	// edit configs. Other users get nothing, including the RCON password.
	if err := m.runner.RunSudo(ctx, "chmod", "-R", "u+rwX,g+rwX,o-rwx", cfg.Dir); err != nil {
		return fmt.Errorf("setting permissions on %s: %w", cfg.Dir, err)
	}
	if err := m.runner.RunSudo(ctx, "find", cfg.Dir, "-type", "d", "-exec", "chmod", "2770", "{}", "+"); err != nil {
		return fmt.Errorf("setting directory permissions on %s: %w", cfg.Dir, err)
	}
	addedToGroup := false
	if u.Username != serviceUser {
		if err := m.runner.RunSudo(ctx, "usermod", "-aG", serviceUser, u.Username); err != nil {
			return fmt.Errorf("adding %s to the %s group: %w. The service user was created, but this account cannot edit server files until it is in that group", u.Username, serviceUser, err)
		}
		addedToGroup = true
	}

	if err := m.runner.RunSudo(ctx, "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	output.Success("Systemd service installed for user %s", serviceUser)
	output.Info("Control with: sudo systemctl start|stop|restart|status minecraft")
	if addedToGroup {
		output.Info("%s was added to the %s group. Log in again before editing server files from this account.", u.Username, serviceUser)
	}
	return nil
}

// systemdUnit renders the hardened unit. dir may contain spaces; paths are
// quoted so systemd does not split them.
func systemdUnit(account string, cfg *config.ServerConfig) string {
	dir := systemdQuotePath(cfg.Dir)
	script := systemdQuotePath(filepath.Join(cfg.Dir, "start.sh"))
	return fmt.Sprintf(`[Unit]
Description=Minecraft Server (MC Dad Server)
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=%s
Group=%s
WorkingDirectory=%s
ExecStart=/usr/bin/bash %s
ExecStop=/usr/bin/bash -c "screen -S %s -p 0 -X stuff 'stop\r'"
Restart=on-failure
RestartSec=30
StandardInput=null
StandardOutput=journal
StandardError=journal

# Hardened dedicated service account. ReadWritePaths is the only tree the
# process may modify; ProtectSystem=strict makes the rest read-only.
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, account, account, dir, script, cfg.SessionName, dir)
}

func systemdQuotePath(s string) string {
	s = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "%", "%%", "\n", " ").Replace(s)
	return `"` + s + `"`
}

func ensureServiceUser(ctx context.Context, runner CommandRunner) error {
	if _, err := runner.RunWithOutput(ctx, "id", serviceUser); err == nil {
		return nil
	}
	if err := runner.RunSudo(ctx, "useradd", "--system", "--user-group", "--no-create-home", "--shell", "/usr/sbin/nologin", serviceUser); err != nil {
		return fmt.Errorf("creating dedicated system user %s: %w. Install it manually with: sudo useradd --system --user-group --no-create-home --shell /usr/sbin/nologin %s", serviceUser, err, serviceUser)
	}
	return nil
}

// ensureServicePathTraversable rejects a server directory the dedicated user
// cannot reach. Parents are not chowned: a mode of 0750 on a home directory
// would hide the server from the minecraft user, and chmod on that home
// directory would change more than the server.
func ensureServicePathTraversable(dir string) error {
	path := filepath.Clean(dir)
	for {
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		info, err := os.Stat(parent)
		if err != nil {
			return fmt.Errorf("checking whether user %s can reach %s: %w", serviceUser, parent, err)
		}
		if info.Mode().Perm()&0o001 == 0 {
			return fmt.Errorf("the systemd service runs as user %s and cannot enter %s (mode %o). Install with --dir /var/lib/minecraft, or grant other-execute on each parent directory", serviceUser, parent, info.Mode().Perm())
		}
		path = parent
	}
}

func (m *systemdManager) Enable() error {
	return m.runner.RunSudo(context.Background(), "systemctl", "enable", "minecraft.service")
}

func (m *systemdManager) Start() error {
	return m.runner.RunSudo(context.Background(), "systemctl", "start", "minecraft.service")
}

func (m *systemdManager) Stop() error {
	return m.runner.RunSudo(context.Background(), "systemctl", "stop", "minecraft.service")
}

func (m *systemdManager) Status() (string, error) {
	out, err := m.runner.RunWithOutput(context.Background(), "systemctl", "is-active", "minecraft.service")
	return string(out), err
}

// --- launchd ---

type launchdManager struct {
	runner    CommandRunner
	cfg       *config.ServerConfig
	plistPath string
}

func (m *launchdManager) Install(cfg *config.ServerConfig) error {
	output := ui.Default()
	output.Step("Setting Up LaunchAgent (macOS)")

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.mc-dad-server.minecraft</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>%s/start.sh</string>
    </array>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>RunAtLoad</key>
    <false/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardOutPath</key>
    <string>%s/logs/launchd-stdout.log</string>
    <key>StandardErrorPath</key>
    <string>%s/logs/launchd-stderr.log</string>
</dict>
</plist>
`, cfg.Dir, cfg.Dir, cfg.Dir, cfg.Dir)

	dir := filepath.Dir(m.plistPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating LaunchAgents dir: %w", err)
	}

	if err := os.WriteFile(m.plistPath, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}

	output.Success("LaunchAgent created")
	output.Info("Load with: launchctl load %s", m.plistPath)
	output.Info("Start with: launchctl start com.mc-dad-server.minecraft")
	return nil
}

func (m *launchdManager) Enable() error {
	return m.runner.Run(context.Background(), "launchctl", "load", m.plistPath)
}

func (m *launchdManager) Start() error {
	return m.runner.Run(context.Background(), "launchctl", "start", "com.mc-dad-server.minecraft")
}

func (m *launchdManager) Stop() error {
	return m.runner.Run(context.Background(), "launchctl", "stop", "com.mc-dad-server.minecraft")
}

func (m *launchdManager) Status() (string, error) {
	out, err := m.runner.RunWithOutput(context.Background(), "launchctl", "list", "com.mc-dad-server.minecraft")
	return string(out), err
}
