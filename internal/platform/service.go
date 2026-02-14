package platform

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// ServiceManager handles platform-specific service installation and management.
type ServiceManager interface {
	Install(cfg *config.ServerConfig) error
	Enable() error
	Start() error
	Stop() error
	Status() (string, error)
}

// NewServiceManager returns the appropriate service manager for the platform.
func NewServiceManager(plat Platform, runner CommandRunner, cfg *config.ServerConfig) ServiceManager {
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

	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	unit := fmt.Sprintf(`[Unit]
Description=Minecraft Server (MC Dad Server)
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=%s
WorkingDirectory=%s
ExecStart=/usr/bin/bash %s/start.sh
ExecStop=/usr/bin/bash -c "screen -S %s -p 0 -X stuff 'stop\r'"
Restart=on-failure
RestartSec=30
StandardInput=null
StandardOutput=journal
StandardError=journal

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=%s

[Install]
WantedBy=multi-user.target
`, u.Username, cfg.Dir, cfg.Dir, cfg.SessionName, cfg.Dir)

	unitPath := "/etc/systemd/system/minecraft.service"
	tmpFile := "/tmp/minecraft.service"
	if err := os.WriteFile(tmpFile, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("writing temp service file: %w", err)
	}
	defer os.Remove(tmpFile)

	ctx := context.Background()
	if err := m.runner.RunSudo(ctx, "cp", tmpFile, unitPath); err != nil {
		return fmt.Errorf("installing service file: %w", err)
	}

	if err := m.runner.RunSudo(ctx, "systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	output.Success("Systemd service installed")
	output.Info("Control with: sudo systemctl start|stop|restart|status minecraft")
	return nil
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
