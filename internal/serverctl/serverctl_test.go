package serverctl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/container"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

func TestResolveModeExplicit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		mode string
		want string
	}{
		{mode: ModeScreen, want: ModeScreen},
		{mode: ModeContainer, want: ModeContainer},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			t.Parallel()

			// An explicit mode must win without probing the system at all.
			runner := platform.NewMockRunner()
			got := ResolveMode(t.Context(), Target{Mode: tt.mode, Session: "minecraft"}, runner)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			if len(runner.Commands) != 0 {
				t.Fatalf("expected no probing, ran %v", runner.Commands)
			}
		})
	}
}

func TestResolveModeAutoFallsBackToScreen(t *testing.T) {
	t.Parallel()

	// No container runtime installed.
	runner := platform.NewMockRunner()
	if got := ResolveMode(t.Context(), Target{Mode: ModeAuto, Session: "minecraft"}, runner); got != ModeScreen {
		t.Fatalf("got %q, want %q", got, ModeScreen)
	}

	// Runtime installed, but no container of that name is running.
	runner = platform.NewMockRunner()
	runner.ExistsMap["podman"] = true
	if got := ResolveMode(t.Context(), Target{Mode: ModeAuto, Session: "minecraft"}, runner); got != ModeScreen {
		t.Fatalf("got %q, want %q", got, ModeScreen)
	}
}

func TestResolveModeAutoDetectsRunningContainer(t *testing.T) {
	t.Parallel()

	runner := platform.NewMockRunner()
	runner.ExistsMap["podman"] = true
	runner.OutputMap["podman [inspect --format {{.State.Running}} minecraft]"] = []byte("true\n")

	if got := ResolveMode(t.Context(), Target{Mode: ModeAuto, Session: "minecraft"}, runner); got != ModeContainer {
		t.Fatalf("got %q, want %q", got, ModeContainer)
	}
}

func TestResolveContainerReportsMissingRCONPassword(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RCON_PASSWORD", "")

	runner := platform.NewMockRunner()
	runner.ExistsMap["podman"] = true

	res := Resolve(t.Context(), Target{Mode: ModeContainer, Dir: dir, Session: "minecraft"}, runner)
	defer func() { _ = res.Close() }()

	if !res.MissingRCONPassword {
		t.Fatal("expected MissingRCONPassword for a server dir without server.properties")
	}
	if res.Mode != ModeContainer {
		t.Fatalf("mode = %q, want %q", res.Mode, ModeContainer)
	}
}

func TestResolveCloseIsSafeForScreen(t *testing.T) {
	t.Parallel()

	res := Resolve(t.Context(), Target{Mode: ModeScreen, Dir: t.TempDir(), Session: "minecraft"}, platform.NewMockRunner())
	if err := res.Close(); err != nil {
		t.Fatalf("unexpected error closing screen manager: %v", err)
	}
}

func TestReadRCONPassword(t *testing.T) {
	dir := t.TempDir()
	props := "server-port=25565\nrcon.password=hunter2  \nrcon.port=25575\n"
	if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte(props), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	t.Setenv("RCON_PASSWORD", "")
	if got := ReadRCONPassword(dir); got != "hunter2" {
		t.Fatalf("got %q, want %q", got, "hunter2")
	}

	// The env var takes precedence over the file.
	t.Setenv("RCON_PASSWORD", "from-env")
	if got := ReadRCONPassword(dir); got != "from-env" {
		t.Fatalf("got %q, want %q", got, "from-env")
	}
}

func TestReadRCONPasswordMissingFile(t *testing.T) {
	t.Setenv("RCON_PASSWORD", "")
	if got := ReadRCONPassword(t.TempDir()); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

// appleRunning is `container inspect` output for a running container.
const appleRunning = `[{"id":"minecraft","status":{"state":"running"}}]`

func runtimeOf(t *testing.T, r Resolved) string {
	t.Helper()
	cm, ok := r.Manager.(*container.Manager)
	if !ok {
		t.Fatalf("expected *container.Manager, got %T", r.Manager)
	}
	return cm.Runtime()
}

func TestResolvePicksRuntimeHostingTheContainer(t *testing.T) {
	t.Setenv("RCON_PASSWORD", "x")

	// Apple's container and podman both installed (as on a Mac); the server
	// runs under podman, so podman must be chosen even though `container`
	// is preferred on darwin.
	runner := platform.NewMockRunner()
	runner.ExistsMap["podman"] = true
	runner.ExistsMap["container"] = true
	runner.OutputMap["podman [inspect --format {{.State.Running}} minecraft]"] = []byte("true\n")

	res := Resolve(t.Context(), Target{Mode: ModeAuto, Session: "minecraft", Dir: t.TempDir()}, runner)
	if res.Mode != ModeContainer {
		t.Fatalf("mode = %q, want container", res.Mode)
	}
	if got := runtimeOf(t, res); got != platform.RuntimePodman {
		t.Fatalf("runtime = %q, want podman", got)
	}
}

func TestResolveExplicitAppleRuntime(t *testing.T) {
	t.Setenv("RCON_PASSWORD", "x")

	runner := platform.NewMockRunner()
	runner.ExistsMap["podman"] = true
	runner.OutputMap["container [inspect minecraft]"] = []byte(appleRunning)

	res := Resolve(t.Context(), Target{Mode: ModeAuto, Session: "minecraft", Dir: t.TempDir(), Runtime: "container"}, runner)
	if res.Mode != ModeContainer {
		t.Fatalf("mode = %q, want container", res.Mode)
	}
	if got := runtimeOf(t, res); got != platform.RuntimeApple {
		t.Fatalf("runtime = %q, want container", got)
	}
	for _, c := range runner.Commands {
		if c.Name == "podman" {
			t.Fatalf("explicit --runtime container must not probe podman, ran %v", runner.Commands)
		}
	}
}

func TestResolveContainerModeFallsBackToPreferredRuntime(t *testing.T) {
	t.Setenv("RCON_PASSWORD", "x")

	// Container mode requested, nothing running yet: use the first candidate.
	runner := platform.NewMockRunner()
	runner.ExistsMap["docker"] = true
	res := Resolve(t.Context(), Target{Mode: ModeContainer, Session: "minecraft", Dir: t.TempDir()}, runner)
	if got := runtimeOf(t, res); got != platform.RuntimeDocker {
		t.Fatalf("runtime = %q, want docker", got)
	}
}
