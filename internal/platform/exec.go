package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
)

// CommandRunner abstracts shell-out operations for testability.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
	RunWithOutput(ctx context.Context, name string, args ...string) ([]byte, error)
	RunSudo(ctx context.Context, name string, args ...string) error
	CommandExists(name string) bool
}

// OSCommandRunner executes real system commands.
type OSCommandRunner struct{}

// NewOSCommandRunner returns a CommandRunner that executes real system commands.
func NewOSCommandRunner() *OSCommandRunner {
	return &OSCommandRunner{}
}

// Run executes a system command and returns any error.
func (r *OSCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	slog.Debug("exec", "cmd", name, "args", args)
	cmd := exec.CommandContext(ctx, name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w: %s", name, args, err, stderr.String())
	}
	return nil
}

// RunWithOutput executes a system command and returns its stdout output.
func (r *OSCommandRunner) RunWithOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	slog.Debug("exec", "cmd", name, "args", args)
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("%s %v: %w: %s", name, args, err, exitErr.Stderr)
		}
		return nil, fmt.Errorf("%s %v: %w", name, args, err)
	}
	return out, nil
}

// RunSudo executes a system command with sudo privileges.
func (r *OSCommandRunner) RunSudo(ctx context.Context, name string, args ...string) error {
	sudoArgs := append([]string{name}, args...)
	return r.Run(ctx, "sudo", sudoArgs...)
}

// CommandExists checks whether a command is available on the system PATH.
func (r *OSCommandRunner) CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// MockRunner records commands for testing without executing them.
type MockRunner struct {
	Commands  []MockCommand
	OutputMap map[string][]byte
	ErrorMap  map[string]error
	ExistsMap map[string]bool
}

// MockCommand records a single command invocation.
type MockCommand struct {
	Name string
	Args []string
	Sudo bool
}

// NewMockRunner creates a MockRunner with empty state.
func NewMockRunner() *MockRunner {
	return &MockRunner{
		OutputMap: make(map[string][]byte),
		ErrorMap:  make(map[string]error),
		ExistsMap: make(map[string]bool),
	}
}

// Key returns the map key used for OutputMap / ErrorMap lookups.
func (m *MockRunner) Key(name string, args ...string) string {
	return fmt.Sprintf("%s %v", name, args)
}

// Run records the command and returns any preconfigured error.
func (m *MockRunner) Run(_ context.Context, name string, args ...string) error {
	m.Commands = append(m.Commands, MockCommand{Name: name, Args: args})
	if err, ok := m.ErrorMap[m.Key(name, args...)]; ok {
		return err
	}
	return nil
}

// RunWithOutput records the command and returns preconfigured output or error.
func (m *MockRunner) RunWithOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	m.Commands = append(m.Commands, MockCommand{Name: name, Args: args})
	if err, ok := m.ErrorMap[m.Key(name, args...)]; ok {
		return nil, err
	}
	if out, ok := m.OutputMap[m.Key(name, args...)]; ok {
		return out, nil
	}
	return nil, nil
}

// RunSudo records the command as a sudo invocation and returns any preconfigured error.
func (m *MockRunner) RunSudo(_ context.Context, name string, args ...string) error {
	m.Commands = append(m.Commands, MockCommand{Name: name, Args: args, Sudo: true})
	if err, ok := m.ErrorMap[m.Key(name, args...)]; ok {
		return err
	}
	return nil
}

// CommandExists returns the preconfigured existence value for the given command.
func (m *MockRunner) CommandExists(name string) bool {
	if exists, ok := m.ExistsMap[name]; ok {
		return exists
	}
	return false
}
