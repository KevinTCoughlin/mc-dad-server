package platform

import (
	"context"
	"fmt"
	"net"
)

// CheckJava confirms the java on PATH is new enough for Paper 26.3.
// The error names the version that was found and how to fix it.
func CheckJava(ctx context.Context, runner CommandRunner) (int, error) {
	if !runner.CommandExists("java") {
		return 0, fmt.Errorf("java check failed: %s", paperJavaRequirement(0, false))
	}
	ver, err := javaVersion(ctx, runner)
	if err != nil {
		return 0, fmt.Errorf("could not read the Java version: %w. %s", err, paperJavaRequirement(0, false))
	}
	if ver < MinJavaVersion {
		return ver, fmt.Errorf("java check failed: %s", paperJavaRequirement(ver, true))
	}
	return ver, nil
}

// CheckPortAvailable reports when TCP port is already bound.
func CheckPortAvailable(ctx context.Context, port int) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("TCP port %d is already in use (%w). Stop the program listening on it, or rerun with --port set to a free port", port, err)
	}
	if err := ln.Close(); err != nil {
		return fmt.Errorf("closing the port %d probe: %w", port, err)
	}
	return nil
}
