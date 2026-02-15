package management

import (
	"context"
	"strings"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// ScreenManager wraps GNU screen session operations.
type ScreenManager struct {
	runner  platform.CommandRunner
	session string
}

// NewScreenManager creates a ScreenManager for the named session.
func NewScreenManager(runner platform.CommandRunner, session string) *ScreenManager {
	return &ScreenManager{runner: runner, session: session}
}

// IsRunning checks if the named screen session exists.
func (s *ScreenManager) IsRunning(ctx context.Context) bool {
	out, err := s.runner.RunWithOutput(ctx, "screen", "-list")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "."+s.session)
}

// SendCommand sends a command string to the screen session.
func (s *ScreenManager) SendCommand(ctx context.Context, cmd string) error {
	return s.runner.Run(ctx, "screen", "-S", s.session, "-p", "0", "-X", "stuff", cmd+"\r")
}

// Start launches a command in a new detached screen session.
func (s *ScreenManager) Start(ctx context.Context, command string, args ...string) error {
	screenArgs := []string{"-dmS", s.session, command}
	screenArgs = append(screenArgs, args...)
	return s.runner.Run(ctx, "screen", screenArgs...)
}

// Sleep pauses for the given number of seconds, respecting context cancellation.
func Sleep(ctx context.Context, seconds int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(seconds) * time.Second):
		return nil
	}
}

// Session returns the session name.
func (s *ScreenManager) Session() string {
	return s.session
}

