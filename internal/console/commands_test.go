package console

import (
	"context"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

func TestDispatch_ContainerMode_UsesContainerManager(t *testing.T) {
	ctx := context.Background()
	runner := platform.NewMockRunner()
	opts := &Options{Dir: t.TempDir(), Session: "minecraft", Mode: "container"}

	_, _ = dispatch(ctx, "status", opts, runner)

	if len(runner.Commands) == 0 {
		t.Fatal("expected command execution")
	}
	first := runner.Commands[0]
	if first.Name != "container" && first.Name != "docker" && first.Name != "podman" {
		// In forced container mode with unknown runtime, command name becomes "unknown".
		// This test ensures screen is not used.
		if first.Name == "screen" {
			t.Fatalf("expected non-screen manager in container mode, got %q", first.Name)
		}
	}
}

func TestDispatch_AutoMode_PrefersRunningContainer(t *testing.T) {
	ctx := context.Background()
	runner := platform.NewMockRunner()
	runner.ExistsMap["docker"] = true
	runner.OutputMap["docker [inspect --format {{.State.Running}} minecraft]"] = []byte("true")
	opts := &Options{Dir: t.TempDir(), Session: "minecraft", Mode: "auto"}

	_, _ = dispatch(ctx, "status", opts, runner)

	if len(runner.Commands) < 2 {
		t.Fatalf("expected at least 2 commands, got %d", len(runner.Commands))
	}
	// First command is auto-detect inspect; second status check should also use docker manager.
	if runner.Commands[1].Name != "docker" {
		t.Fatalf("expected docker command for status, got %q", runner.Commands[1].Name)
	}
}
