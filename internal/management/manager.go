package management

import "context"

// ServerManager is the interface for managing a Minecraft server process,
// whether via a GNU screen session or a container runtime.
type ServerManager interface {
	// IsRunning reports whether the managed server process is active.
	IsRunning(ctx context.Context) bool

	// SendCommand sends a console command to the running server.
	SendCommand(ctx context.Context, cmd string) error

	// Launch starts the server. Each implementation owns its own startup
	// details (screen session + start script, podman start, etc.).
	Launch(ctx context.Context) error

	// Session returns the session or container name.
	Session() string
}
