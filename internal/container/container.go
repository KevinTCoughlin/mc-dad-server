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

	mu   sync.Mutex
	rcon *RCONClient
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

// SendCommand sends a console command to the server via a persistent RCON
// connection. The connection is lazily established on the first call and
// reused for subsequent calls. If the connection is broken, it is
// automatically re-established.
func (c *Manager) SendCommand(ctx context.Context, cmd string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnectedLocked(ctx); err != nil {
		return fmt.Errorf("rcon: %w", err)
	}

	_, err := c.rcon.Command(ctx, cmd)
	if err != nil && isConnectionError(err) {
		// Connection is broken — close and retry once.
		_ = c.rcon.Close()
		c.rcon = nil

		if err := c.ensureConnectedLocked(ctx); err != nil {
			return fmt.Errorf("rcon reconnect: %w", err)
		}
		_, err = c.rcon.Command(ctx, cmd)
	}
	return err
}

// Close tears down the persistent RCON connection, if any.
func (c *Manager) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.rcon == nil {
		return nil
	}
	err := c.rcon.Close()
	c.rcon = nil
	return err
}

// Start starts the container. The command and args parameters are ignored;
// the container is started via podman.
func (c *Manager) Start(ctx context.Context, _ string, _ ...string) error {
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

// ensureConnectedLocked lazily connects the persistent RCON client.
// The caller must hold c.mu.
func (c *Manager) ensureConnectedLocked(ctx context.Context) error {
	if c.rcon != nil {
		return nil
	}
	client := NewRCONClient(c.rconAddr, c.rconPass)
	if err := client.Connect(ctx); err != nil {
		return err
	}
	c.rcon = client
	return nil
}

// isConnectionError reports whether err indicates a broken TCP connection
// that should trigger a reconnect attempt.
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "not connected") ||
		strings.Contains(msg, "use of closed network connection")
}
