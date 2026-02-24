package container

import (
	"context"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

func TestManager_IsRunning(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{name: "running", output: "true", want: true},
		{name: "not running", output: "false", want: false},
		{name: "empty", output: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := platform.NewMockRunner()
			mock.OutputMap["podman [inspect --format {{.State.Running}} mc]"] = []byte(tt.output)
			m := NewManager(mock, "mc", "127.0.0.1:25575", "pass")

			if got := m.IsRunning(ctx); got != tt.want {
				t.Errorf("IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_Launch(t *testing.T) {
	mock := platform.NewMockRunner()
	m := NewManager(mock, "mc", "127.0.0.1:25575", "pass")

	if err := m.Launch(context.Background()); err != nil {
		t.Fatalf("Launch() error = %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if cmd.Name != "podman" {
		t.Errorf("command name = %q, want %q", cmd.Name, "podman")
	}
	wantArgs := []string{"start", "mc"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", cmd.Args, wantArgs)
	}
	for i, a := range wantArgs {
		if cmd.Args[i] != a {
			t.Errorf("arg[%d] = %q, want %q", i, cmd.Args[i], a)
		}
	}
}

func TestManager_Session(t *testing.T) {
	m := NewManager(platform.NewMockRunner(), "mycontainer", "", "")
	if got := m.Session(); got != "mycontainer" {
		t.Errorf("Session() = %q, want %q", got, "mycontainer")
	}
}

func TestManager_Stop(t *testing.T) {
	mock := platform.NewMockRunner()
	m := NewManager(mock, "mc", "", "")

	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if cmd.Name != "podman" {
		t.Errorf("command = %q, want %q", cmd.Name, "podman")
	}
}

func TestExists(t *testing.T) {
	ctx := context.Background()

	t.Run("container exists", func(t *testing.T) {
		mock := platform.NewMockRunner()
		if !Exists(ctx, mock, "mc") {
			t.Error("Exists() = false, want true")
		}
	})

	t.Run("container does not exist", func(t *testing.T) {
		mock := platform.NewMockRunner()
		mock.ErrorMap["podman [inspect --type container mc]"] = context.DeadlineExceeded
		if Exists(ctx, mock, "mc") {
			t.Error("Exists() = true, want false")
		}
	})
}

func TestManager_Health(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "healthy", output: "healthy", want: "healthy"},
		{name: "unhealthy", output: "unhealthy", want: "unhealthy"},
		{name: "starting", output: "starting", want: "starting"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := platform.NewMockRunner()
			mock.OutputMap["podman [inspect --format {{.State.Healthcheck.Status}} mc]"] = []byte(tt.output)
			m := NewManager(mock, "mc", "", "")

			if got := m.Health(ctx); got != tt.want {
				t.Errorf("Health() = %q, want %q", got, tt.want)
			}
		})
	}
}
