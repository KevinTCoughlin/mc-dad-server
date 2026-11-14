package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// Verify Manager satisfies management.ServerManager at compile time.
var _ management.ServerManager = (*Manager)(nil)

// LaunchdLabel is the launchd agent that supervises the server container
// under Apple's `container` runtime (written by `setup-container` on macOS).
const LaunchdLabel = "com.mc-dad-server.minecraft-container"

// Manager manages a Minecraft server running in a container (Podman, Docker,
// or Apple's `container`). It implements management.ServerManager and
// management.HealthChecker.
type Manager struct {
	runner    platform.CommandRunner
	runtime   string // platform.RuntimePodman, RuntimeDocker, or RuntimeApple
	container string
	rconAddr  string
	rconPass  string
	rcon      *RCONClient
	rconMu    sync.Mutex
}

// NewManager creates a Manager for the named container using the specified runtime.
func NewManager(runner platform.CommandRunner, runtime, container, rconAddr, rconPass string) *Manager {
	return &Manager{
		runner:    runner,
		runtime:   runtime,
		container: container,
		rconAddr:  rconAddr,
		rconPass:  rconPass,
	}
}

// isApple reports whether this manager drives Apple's `container` CLI, whose
// inspect/stats output is JSON rather than Go-template formatted.
func (c *Manager) isApple() bool {
	return c.runtime == platform.RuntimeApple
}

// appleInspect is the subset of `container inspect` JSON the manager reads.
type appleInspect struct {
	Status struct {
		State string `json:"state"`
	} `json:"status"`
}

// appleState returns the container's state ("running", "stopped", ...) as
// reported by `container inspect`, or "" if it cannot be determined.
func (c *Manager) appleState(ctx context.Context) string {
	out, err := c.runner.RunWithOutput(ctx, c.runtime, "inspect", c.container)
	if err != nil {
		return ""
	}
	var items []appleInspect
	if err := json.Unmarshal(out, &items); err != nil || len(items) == 0 {
		return ""
	}
	return items[0].Status.State
}

// IsRunning reports whether the container is running.
func (c *Manager) IsRunning(ctx context.Context) bool {
	if c.isApple() {
		return c.appleState(ctx) == "running"
	}
	out, err := c.runner.RunWithOutput(ctx, c.runtime, "inspect", "--format", "{{.State.Running}}", c.container)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

// ensureRCON lazily initialises and returns the persistent RCON connection,
// reconnecting if the previous connection was lost.
// Must be called with c.rconMu held.
func (c *Manager) ensureRCON(ctx context.Context) (*RCONClient, error) {
	if c.rcon != nil {
		return c.rcon, nil
	}
	rc := NewRCONClient(c.rconAddr, c.rconPass)
	if err := rc.Connect(ctx); err != nil {
		return nil, err
	}
	c.rcon = rc
	return rc, nil
}

// SendCommand sends a console command to the server via RCON.
// It reuses a persistent connection and reconnects once on failure.
func (c *Manager) SendCommand(ctx context.Context, cmd string) error {
	c.rconMu.Lock()
	defer c.rconMu.Unlock()

	rc, err := c.ensureRCON(ctx)
	if err != nil {
		return fmt.Errorf("rcon: %w", err)
	}

	_, err = rc.Command(ctx, cmd)
	if err == nil {
		return nil
	}

	// Reconnect once on connection errors.
	if !isConnectionError(err) {
		return err
	}
	_ = rc.Close()
	c.rcon = nil

	rc, err = c.ensureRCON(ctx)
	if err != nil {
		return fmt.Errorf("rcon reconnect: %w", err)
	}
	_, err = rc.Command(ctx, cmd)
	return err
}

// Close tears down the persistent RCON connection, if any.
func (c *Manager) Close() error {
	c.rconMu.Lock()
	defer c.rconMu.Unlock()

	if c.rcon == nil {
		return nil
	}
	err := c.rcon.Close()
	c.rcon = nil
	return err
}

// Launch starts the container via the configured runtime. Under Apple's
// `container`, the launchd agent written by setup-container owns the
// container's lifecycle, so Launch asks launchd to start it and only falls
// back to `container start` when no agent is loaded.
func (c *Manager) Launch(ctx context.Context) error {
	if c.isApple() {
		agent := fmt.Sprintf("gui/%d/%s", os.Getuid(), LaunchdLabel)
		if err := c.runner.Run(ctx, "launchctl", "kickstart", agent); err == nil {
			return nil
		}
	}
	return c.runner.Run(ctx, c.runtime, "start", c.container)
}

// Stop stops the container with a 60-second grace period so the entrypoint
// can perform a graceful shutdown.
func (c *Manager) Stop(ctx context.Context) error {
	return c.runner.Run(ctx, c.runtime, "stop", "-t", "60", c.container)
}

// Runtime returns the container runtime this manager drives.
func (c *Manager) Runtime() string {
	return c.runtime
}

// Session returns the container name.
func (c *Manager) Session() string {
	return c.container
}

// Health returns the container health status string (e.g. "healthy", "unhealthy", "starting").
func (c *Manager) Health(ctx context.Context) string {
	// Apple's `container` has no HEALTHCHECK support, so report run state.
	if c.isApple() {
		switch c.appleState(ctx) {
		case "":
			return "unknown"
		case "running":
			return "running"
		default:
			return "stopped"
		}
	}

	// Docker and Podman use different health status paths
	format := "{{.State.Healthcheck.Status}}" // Podman
	if c.runtime == "docker" {
		format = "{{.State.Health.Status}}" // Docker
	}

	out, err := c.runner.RunWithOutput(ctx, c.runtime, "inspect", "--format", format, c.container)
	if err != nil {
		return "unknown"
	}
	status := strings.TrimSpace(string(out))
	if status == "" || status == "<no value>" {
		if c.IsRunning(ctx) {
			return "running"
		}
		return "stopped"
	}
	return status
}

// Stats returns a formatted string with container resource usage.
func (c *Manager) Stats(ctx context.Context) (string, error) {
	if c.isApple() {
		return c.appleStats(ctx)
	}
	out, err := c.runner.RunWithOutput(ctx, c.runtime, "stats", "--no-stream", "--format",
		"CPU: {{.CPUPerc}}  MEM: {{.MemUsage}}", c.container)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// appleStats formats `container stats --format json` output to match the
// "CPU: x%  MEM: used / limit" line podman and docker produce.
func (c *Manager) appleStats(ctx context.Context) (string, error) {
	out, err := c.runner.RunWithOutput(ctx, c.runtime, "stats", "--no-stream", "--format", "json", c.container)
	if err != nil {
		return "", err
	}
	var items []struct {
		CPUUtilization   float64 `json:"cpuUtilization"`
		MemoryUsageBytes int64   `json:"memoryUsageBytes"`
		MemoryLimitBytes int64   `json:"memoryLimitBytes"`
	}
	if err := json.Unmarshal(out, &items); err != nil {
		return "", fmt.Errorf("parsing container stats: %w", err)
	}
	if len(items) == 0 {
		return "", errors.New("container stats: no data")
	}
	s := items[0]
	return fmt.Sprintf("CPU: %.2f%%  MEM: %s / %s", s.CPUUtilization, formatMiB(s.MemoryUsageBytes), formatMiB(s.MemoryLimitBytes)), nil
}

// formatMiB renders a byte count the way podman's stats do (e.g. "512MiB",
// "1.5GiB").
func formatMiB(b int64) string {
	const mib = 1 << 20
	if b >= 1<<30 {
		return fmt.Sprintf("%.2fGiB", float64(b)/(1<<30))
	}
	return fmt.Sprintf("%.1fMiB", float64(b)/mib)
}

// Exists checks if a container with the given name exists (running or stopped).
func Exists(ctx context.Context, runner platform.CommandRunner, runtime, name string) bool {
	if runtime == platform.RuntimeApple {
		// `container inspect` has no --type flag; it exits non-zero when the
		// container is not found.
		return runner.Run(ctx, runtime, "inspect", name) == nil
	}
	err := runner.Run(ctx, runtime, "inspect", "--type", "container", name)
	return err == nil
}

// isConnectionError reports whether err looks like a broken, closed, or
// unusable connection that warrants a reconnect attempt.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// A desynchronized stream is not recoverable in place — the client has
	// already dropped the socket, so reconnecting is the only way forward.
	if errors.Is(err, ErrDesynchronized) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "not connected") ||
		strings.Contains(msg, "use of closed network connection")
}
