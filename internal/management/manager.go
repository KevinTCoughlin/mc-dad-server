package management

import "context"

// ServerManager is the interface for managing a Minecraft server process,
// whether via a GNU screen session or a container runtime.
type ServerManager interface {
	// IsRunning reports whether the managed server process is active.
	IsRunning(ctx context.Context) bool

	// SendCommand sends a console command to the running server.
	SendCommand(ctx context.Context, cmd string) error

	// Start launches the server. In screen mode the arguments are the
	// command and its args; in container mode they may be ignored.
	Start(ctx context.Context, command string, args ...string) error

	// Session returns the session or container name.
	Session() string
}
