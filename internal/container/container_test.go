package container

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

func TestManager_IsRunning(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		output string
		err    error
		want   bool
	}{
		{
			name:   "running",
			output: "true\n",
			want:   true,
		},
		{
			name:   "stopped",
			output: "false\n",
			want:   false,
		},
		{
			name:   "no whitespace",
			output: "true",
			want:   true,
		},
		{
			name: "error",
			err:  errors.New("no such container"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := platform.NewMockRunner()
			key := mock.Key("podman", "inspect", "--format", "{{.State.Running}}", "minecraft")
			if tt.err != nil {
				mock.ErrorMap[key] = tt.err
			} else {
				mock.OutputMap[key] = []byte(tt.output)
			}
			m := NewManager(mock, "minecraft", "127.0.0.1:25575", "pass")

			if got := m.IsRunning(ctx); got != tt.want {
				t.Errorf("IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_Start(t *testing.T) {
	mock := platform.NewMockRunner()
	m := NewManager(mock, "minecraft", "", "")

	if err := m.Start(context.Background(), "ignored"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if cmd.Name != "podman" {
		t.Errorf("command name = %q, want %q", cmd.Name, "podman")
	}
	if len(cmd.Args) < 2 || cmd.Args[0] != "start" || cmd.Args[1] != "minecraft" {
		t.Errorf("args = %v, want [start minecraft]", cmd.Args)
	}
}

func TestManager_Start_Error(t *testing.T) {
	mock := platform.NewMockRunner()
	key := mock.Key("podman", "start", "minecraft")
	mock.ErrorMap[key] = errors.New("container not found")
	m := NewManager(mock, "minecraft", "", "")

	err := m.Start(context.Background(), "ignored")
	if err == nil {
		t.Fatal("Start() expected error, got nil")
	}
}

func TestManager_Stop(t *testing.T) {
	mock := platform.NewMockRunner()
	m := NewManager(mock, "minecraft", "", "")

	if err := m.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if cmd.Name != "podman" {
		t.Errorf("command name = %q, want %q", cmd.Name, "podman")
	}
	if len(cmd.Args) < 4 || cmd.Args[0] != "stop" || cmd.Args[1] != "-t" || cmd.Args[2] != "60" || cmd.Args[3] != "minecraft" {
		t.Errorf("args = %v, want [stop -t 60 minecraft]", cmd.Args)
	}
}

func TestManager_Stop_Error(t *testing.T) {
	mock := platform.NewMockRunner()
	key := mock.Key("podman", "stop", "-t", "60", "minecraft")
	mock.ErrorMap[key] = errors.New("podman error")
	m := NewManager(mock, "minecraft", "", "")

	err := m.Stop(context.Background())
	if err == nil {
		t.Fatal("Stop() expected error, got nil")
	}
}

func TestManager_Session(t *testing.T) {
	m := NewManager(nil, "mycontainer", "", "")
	if got := m.Session(); got != "mycontainer" {
		t.Errorf("Session() = %q, want %q", got, "mycontainer")
	}
}

func TestManager_Health(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		output     string
		err        error
		isRunning  bool
		runningErr error
		want       string
	}{
		{
			name:   "healthy",
			output: "healthy\n",
			want:   "healthy",
		},
		{
			name:   "unhealthy",
			output: "unhealthy\n",
			want:   "unhealthy",
		},
		{
			name:   "starting",
			output: "starting\n",
			want:   "starting",
		},
		{
			name: "inspect error",
			err:  errors.New("no such container"),
			want: "unknown",
		},
		{
			name:      "empty status running",
			output:    "\n",
			isRunning: true,
			want:      "running",
		},
		{
			name:      "no value running",
			output:    "<no value>\n",
			isRunning: true,
			want:      "running",
		},
		{
			name:   "empty status stopped",
			output: "\n",
			want:   "stopped",
		},
		{
			name:       "empty status running error",
			output:     "\n",
			runningErr: errors.New("inspect failed"),
			want:       "stopped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := platform.NewMockRunner()
			healthKey := mock.Key("podman", "inspect", "--format", "{{.State.Healthcheck.Status}}", "minecraft")
			if tt.err != nil {
				mock.ErrorMap[healthKey] = tt.err
			} else {
				mock.OutputMap[healthKey] = []byte(tt.output)
			}

			// For the IsRunning fallback when status is empty/<no value>.
			runningKey := mock.Key("podman", "inspect", "--format", "{{.State.Running}}", "minecraft")
			if tt.isRunning {
				mock.OutputMap[runningKey] = []byte("true")
			} else if tt.runningErr != nil {
				mock.ErrorMap[runningKey] = tt.runningErr
			}

			m := NewManager(mock, "minecraft", "", "")
			if got := m.Health(ctx); got != tt.want {
				t.Errorf("Health() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestManager_Stats(t *testing.T) {
	mock := platform.NewMockRunner()
	key := mock.Key("podman", "stats", "--no-stream", "--format", "CPU: {{.CPUPerc}}  MEM: {{.MemUsage}}", "minecraft")
	mock.OutputMap[key] = []byte("CPU: 12.5%  MEM: 1.2GiB / 3GiB\n")
	m := NewManager(mock, "minecraft", "", "")

	got, err := m.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	want := "CPU: 12.5%  MEM: 1.2GiB / 3GiB"
	if got != want {
		t.Errorf("Stats() = %q, want %q", got, want)
	}
}

func TestManager_Stats_Error(t *testing.T) {
	mock := platform.NewMockRunner()
	key := mock.Key("podman", "stats", "--no-stream", "--format", "CPU: {{.CPUPerc}}  MEM: {{.MemUsage}}", "minecraft")
	mock.ErrorMap[key] = errors.New("container not running")
	m := NewManager(mock, "minecraft", "", "")

	_, err := m.Stats(context.Background())
	if err == nil {
		t.Fatal("Stats() expected error, got nil")
	}
}

func TestExists(t *testing.T) {
	ctx := context.Background()

	t.Run("exists", func(t *testing.T) {
		mock := platform.NewMockRunner()
		if !Exists(ctx, mock, "minecraft") {
			t.Error("Exists() = false, want true")
		}
	})

	t.Run("not exists", func(t *testing.T) {
		mock := platform.NewMockRunner()
		key := mock.Key("podman", "inspect", "--type", "container", "minecraft")
		mock.ErrorMap[key] = errors.New("no such container")
		if Exists(ctx, mock, "minecraft") {
			t.Error("Exists() = true, want false")
		}
	})
}

func TestManager_SendCommand(t *testing.T) {
	// Start a test RCON server.
	srv := newRCONTestServer(t, "rconpass", func(cmd string) string {
		return "done:" + cmd
	})
	defer srv.Close()
	go srv.Serve(t)

	mock := platform.NewMockRunner()
	m := NewManager(mock, "minecraft", srv.Addr(), "rconpass")

	if err := m.SendCommand(context.Background(), "say hello"); err != nil {
		t.Fatalf("SendCommand() error = %v", err)
	}
}

func TestManager_SendCommand_ConnectFailure(t *testing.T) {
	mock := platform.NewMockRunner()
	// Allocate an ephemeral port, then close it so that dialing will fail.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("listener Close() error = %v", err)
	}

	m := NewManager(mock, "minecraft", addr, "pass")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = m.SendCommand(ctx, "list")
	if err == nil {
		t.Fatal("SendCommand() expected error, got nil")
	}
}
