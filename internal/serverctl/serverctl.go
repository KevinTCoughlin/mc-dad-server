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
	// Runtime is the container runtime to use in container mode: "podman",
	// "docker", "container" (Apple), or "" / "auto" to detect.
	Runtime string
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

// RuntimeAuto selects the container runtime automatically.
const RuntimeAuto = "auto"

// Resolve returns a ServerManager for the target.
func Resolve(ctx context.Context, t Target, runner platform.CommandRunner) Resolved {
	mode, runtime := resolve(ctx, t, runner)
	if mode == ModeContainer {
		rconPass := ReadRCONPassword(t.Dir)
		return Resolved{
			Manager:             container.NewManager(runner, runtime, t.Session, DefaultRCONAddr, rconPass),
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
	if t.Mode == ModeScreen || t.Mode == ModeContainer {
		return t.Mode // explicit mode wins without probing the system
	}
	mode, _ := resolve(ctx, t, runner)
	return mode
}

// resolve picks the server mode and, for container mode, the runtime.
//
// Auto mode selects container mode only when a container with the session
// name is running under one of the candidate runtimes, and then uses that
// runtime — so a Mac with both Apple's `container` and Podman installed
// manages whichever one actually hosts the server. Explicit container mode
// with no running container falls back to the most preferred runtime.
func resolve(ctx context.Context, t Target, runner platform.CommandRunner) (mode, runtime string) {
	if t.Mode == ModeScreen {
		return ModeScreen, ""
	}
	candidates := candidateRuntimes(t, runner)
	for _, rt := range candidates {
		if container.NewManager(runner, rt, t.Session, "", "").IsRunning(ctx) {
			return ModeContainer, rt
		}
	}
	if t.Mode == ModeContainer {
		if len(candidates) > 0 {
			return ModeContainer, candidates[0]
		}
		return ModeContainer, platform.RuntimeUnknown
	}
	return ModeScreen, ""
}

// candidateRuntimes returns the runtimes to consider: the explicitly
// requested one, or every installed runtime in preference order.
func candidateRuntimes(t Target, runner platform.CommandRunner) []string {
	if t.Runtime != "" && t.Runtime != RuntimeAuto {
		return []string{t.Runtime}
	}
	return platform.AvailableContainerRuntimes(runner)
}

// DetectRuntime returns the most preferred installed container runtime
// ("podman", "docker", "container", or "unknown"). See
// platform.DetectContainerRuntime for the preference order.
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
