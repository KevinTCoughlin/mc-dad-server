// Package serverctl resolves which backend manages a Minecraft server —
// a GNU screen session or a container runtime — and builds the matching
// management.ServerManager.
//
// It exists so the CLI commands and the interactive console share one
// implementation: the two previously carried byte-identical copies of this
// logic, which drifted apart and had to be re-fixed in both places.
package serverctl

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/container"
	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// Server modes.
const (
	ModeAuto      = "auto"
	ModeScreen    = "screen"
	ModeContainer = "container"
)

// DefaultRCONAddr is the address container mode uses to reach the server's
// RCON listener, which the shipped configs bind to loopback.
const DefaultRCONAddr = "127.0.0.1:25575"

// Target identifies the server to manage.
type Target struct {
	// Mode is "auto", "screen", or "container".
	Mode string
	// Dir is the server directory.
	Dir string
	// Session is the screen session or container name.
	Session string
}

// Resolved is a manager plus the context needed to report on it.
type Resolved struct {
	// Manager operates the server. Callers should release it with Close
	// when finished — container managers hold a persistent RCON connection.
	Manager management.ServerManager

	// Mode is the concrete mode that was selected ("screen" or "container").
	Mode string

	// MissingRCONPassword is true when container mode was selected but no
	// RCON password could be found, so console commands will fail.
	MissingRCONPassword bool
}

// Close releases any resources held by the manager. Safe to call on a
// zero-value Resolved.
func (r Resolved) Close() error {
	if c, ok := r.Manager.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// Resolve returns a ServerManager for the target.
func Resolve(ctx context.Context, t Target, runner platform.CommandRunner) Resolved {
	mode := ResolveMode(ctx, t, runner)
	if mode == ModeContainer {
		rconPass := ReadRCONPassword(t.Dir)
		return Resolved{
			Manager:             container.NewManager(runner, DetectRuntime(runner), t.Session, DefaultRCONAddr, rconPass),
			Mode:                mode,
			MissingRCONPassword: rconPass == "",
		}
	}
	return Resolved{
		Manager: management.NewScreenManager(runner, t.Session, filepath.Join(t.Dir, "start.sh")),
		Mode:    mode,
	}
}

// ResolveMode determines the server mode from the target's mode setting,
// auto-detecting when it is unset or "auto".
func ResolveMode(ctx context.Context, t Target, runner platform.CommandRunner) string {
	switch t.Mode {
	case ModeScreen:
		return ModeScreen
	case ModeContainer:
		return ModeContainer
	default:
		return detectMode(ctx, t, runner)
	}
}

// detectMode selects container mode only when a container with the session
// name is currently running; otherwise it defaults to screen mode.
func detectMode(ctx context.Context, t Target, runner platform.CommandRunner) string {
	runtime := DetectRuntime(runner)
	if runtime != "unknown" {
		cm := container.NewManager(runner, runtime, t.Session, "", "")
		if cm.IsRunning(ctx) {
			return ModeContainer
		}
	}
	return ModeScreen
}

// DetectRuntime returns the available container runtime ("podman", "docker",
// or "unknown"). Podman is preferred when both are installed.
func DetectRuntime(runner platform.CommandRunner) string {
	return platform.DetectContainerRuntime(runner)
}

// ReadRCONPassword reads the RCON password, checking the RCON_PASSWORD env
// var first, then falling back to server.properties in the server dir.
func ReadRCONPassword(serverDir string) string {
	if pass := os.Getenv("RCON_PASSWORD"); pass != "" {
		return pass
	}
	data, err := os.ReadFile(filepath.Join(serverDir, "server.properties"))
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if after, ok := strings.CutPrefix(line, "rcon.password="); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

// Config returns a ServerConfig seeded with defaults for the target.
func Config(t Target) *config.ServerConfig {
	cfg := config.DefaultConfig()
	cfg.Dir = t.Dir
	cfg.SessionName = t.Session
	return cfg
}
