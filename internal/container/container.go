package container

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// Manager manages a Minecraft server running in a Podman container.
// It implements management.ServerManager.
type Manager struct {
	runner    platform.CommandRunner
	container string
	rconAddr  string
	rconPass  string
	rcon      *RCONClient
	rconMu    sync.Mutex
}

// NewManager creates a Manager for the named container.
func NewManager(runner platform.CommandRunner, container, rconAddr, rconPass string) *Manager {
	return &Manager{
		runner:    runner,
		container: container,
		rconAddr:  rconAddr,
		rconPass:  rconPass,
	}
}

// IsRunning reports whether the container is running.
func (c *Manager) IsRunning(ctx context.Context) bool {
	out, err := c.runner.RunWithOutput(ctx, "podman", "inspect", "--format", "{{.State.Running}}", c.container)
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

// Launch starts the container.
func (c *Manager) Launch(ctx context.Context) error {
	return c.runner.Run(ctx, "podman", "start", c.container)
}

// Stop stops the container with a 60-second grace period so the entrypoint
// can perform a graceful shutdown.
func (c *Manager) Stop(ctx context.Context) error {
	return c.runner.Run(ctx, "podman", "stop", "-t", "60", c.container)
}

// Session returns the container name.
func (c *Manager) Session() string {
	return c.container
}

// Health returns the container health status string (e.g. "healthy", "unhealthy", "starting").
func (c *Manager) Health(ctx context.Context) string {
	out, err := c.runner.RunWithOutput(ctx, "podman", "inspect", "--format", "{{.State.Healthcheck.Status}}", c.container)
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
	out, err := c.runner.RunWithOutput(ctx, "podman", "stats", "--no-stream", "--format",
		"CPU: {{.CPUPerc}}  MEM: {{.MemUsage}}", c.container)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Exists checks if a container with the given name exists (running or stopped).
func Exists(ctx context.Context, runner platform.CommandRunner, name string) bool {
	err := runner.Run(ctx, "podman", "inspect", "--type", "container", name)
	return err == nil
}

// isConnectionError reports whether err looks like a broken or closed
// network connection that warrants a reconnect attempt.
func isConnectionError(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
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
