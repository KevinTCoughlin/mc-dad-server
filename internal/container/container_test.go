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
	tests := []struct {
		name       string
		runtime    string
		output     string
		err        bool
		wantResult bool
	}{
		{
			name:       "podman container running",
			runtime:    "podman",
			output:     "true\n",
			err:        false,
			wantResult: true,
		},
		{
			name:       "docker container running",
			runtime:    "docker",
			output:     "true\n",
			err:        false,
			wantResult: true,
		},
		{
			name:       "container stopped",
			runtime:    "podman",
			output:     "false\n",
			err:        false,
			wantResult: false,
		},
		{
			name:       "container not found",
			runtime:    "docker",
			output:     "",
			err:        true,
			wantResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := platform.NewMockRunner()
			key := tc.runtime + " [inspect --format {{.State.Running}} minecraft]"
			m.OutputMap[key] = []byte(tc.output)
			if tc.err {
				m.ErrorMap[key] = errors.New("mock error")
			}

			mgr := NewManager(m, tc.runtime, "minecraft", "localhost:25575", "testpass")
			got := mgr.IsRunning(context.Background())
			if got != tc.wantResult {
				t.Errorf("IsRunning() = %v, want %v", got, tc.wantResult)
			}
		})
	}
}

func TestManager_Launch(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
	}{
		{
			name:    "launch with podman",
			runtime: "podman",
		},
		{
			name:    "launch with docker",
			runtime: "docker",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := platform.NewMockRunner()
			mgr := NewManager(m, tc.runtime, "minecraft", "localhost:25575", "testpass")

			err := mgr.Launch(context.Background())
			if err != nil {
				t.Errorf("Launch() error = %v", err)
			}

			// Verify the correct runtime and command were used
			if len(m.Commands) != 1 {
				t.Fatalf("expected 1 command, got %d", len(m.Commands))
			}
			if m.Commands[0].Name != tc.runtime {
				t.Errorf("expected runtime %q, got %q", tc.runtime, m.Commands[0].Name)
			}
		})
	}
}

func TestManager_Stop(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
	}{
		{
			name:    "stop with podman",
			runtime: "podman",
		},
		{
			name:    "stop with docker",
			runtime: "docker",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := platform.NewMockRunner()
			mgr := NewManager(m, tc.runtime, "minecraft", "localhost:25575", "testpass")

			err := mgr.Stop(context.Background())
			if err != nil {
				t.Errorf("Stop() error = %v", err)
			}
		})
	}
}

func TestExists(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
		exists  bool
	}{
		{
			name:    "podman container exists",
			runtime: "podman",
			exists:  true,
		},
		{
			name:    "docker container exists",
			runtime: "docker",
			exists:  true,
		},
		{
			name:    "podman container not found",
			runtime: "podman",
			exists:  false,
		},
		{
			name:    "docker container not found",
			runtime: "docker",
			exists:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := platform.NewMockRunner()
			if !tc.exists {
				key := tc.runtime + " [inspect --type container minecraft]"
				m.ErrorMap[key] = errors.New("mock error")
			}

			got := Exists(context.Background(), m, tc.runtime, "minecraft")
			if got != tc.exists {
				t.Errorf("Exists() = %v, want %v", got, tc.exists)
			}
		})
	}
}

func TestManager_Session(t *testing.T) {
	mgr := NewManager(platform.NewMockRunner(), "podman", "my-container", "localhost:25575", "testpass")
	if got := mgr.Session(); got != "my-container" {
		t.Errorf("Session() = %q, want %q", got, "my-container")
	}
}

func TestManager_Health(t *testing.T) {
	tests := []struct {
		name          string
		runtime       string
		inspectOutput string
		inspectErr    bool
		runningOutput string
		wantHealth    string
	}{
		{
			name:          "podman healthy",
			runtime:       "podman",
			inspectOutput: "healthy\n",
			wantHealth:    "healthy",
		},
		{
			name:          "docker healthy",
			runtime:       "docker",
			inspectOutput: "healthy\n",
			wantHealth:    "healthy",
		},
		{
			name:          "podman unhealthy",
			runtime:       "podman",
			inspectOutput: "unhealthy\n",
			wantHealth:    "unhealthy",
		},
		{
			name:          "docker starting",
			runtime:       "docker",
			inspectOutput: "starting\n",
			wantHealth:    "starting",
		},
		{
			name:          "podman no healthcheck running",
			runtime:       "podman",
			inspectOutput: "<no value>\n",
			runningOutput: "true\n",
			wantHealth:    "running",
		},
		{
			name:          "docker no healthcheck stopped",
			runtime:       "docker",
			inspectOutput: "<no value>\n",
			runningOutput: "false\n",
			wantHealth:    "stopped",
		},
		{
			name:          "podman inspect error",
			runtime:       "podman",
			inspectOutput: "",
			inspectErr:    true,
			wantHealth:    "unknown",
		},
		{
			name:          "docker inspect error",
			runtime:       "docker",
			inspectOutput: "",
			inspectErr:    true,
			wantHealth:    "unknown",
		},
		{
			name:          "podman empty status running error",
			runtime:       "podman",
			inspectOutput: "\n",
			wantHealth:    "stopped",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := platform.NewMockRunner()

			// Setup inspect mock based on runtime
			var inspectFormat string
			if tc.runtime == "docker" {
				inspectFormat = "{{.State.Health.Status}}"
			} else {
				inspectFormat = "{{.State.Healthcheck.Status}}"
			}
			inspectKey := tc.runtime + " [inspect --format " + inspectFormat + " minecraft]"
			m.OutputMap[inspectKey] = []byte(tc.inspectOutput)
			if tc.inspectErr {
				m.ErrorMap[inspectKey] = errors.New("mock error")
			}

			// Setup IsRunning mock for fallback cases
			if tc.inspectOutput == "<no value>\n" {
				runningKey := tc.runtime + " [inspect --format {{.State.Running}} minecraft]"
				m.OutputMap[runningKey] = []byte(tc.runningOutput)
			}

			mgr := NewManager(m, tc.runtime, "minecraft", "localhost:25575", "testpass")
			got := mgr.Health(context.Background())
			if got != tc.wantHealth {
				t.Errorf("Health() = %q, want %q", got, tc.wantHealth)
			}
		})
	}
}

func TestManager_Stats(t *testing.T) {
	tests := []struct {
		name      string
		runtime   string
		output    string
		err       bool
		wantStats string
		wantErr   bool
	}{
		{
			name:      "podman stats success",
			runtime:   "podman",
			output:    "CPU: 5.23%  MEM: 512MiB / 16GiB\n",
			wantStats: "CPU: 5.23%  MEM: 512MiB / 16GiB",
			wantErr:   false,
		},
		{
			name:      "docker stats success",
			runtime:   "docker",
			output:    "CPU: 10.50%  MEM: 1GiB / 32GiB\n",
			wantStats: "CPU: 10.50%  MEM: 1GiB / 32GiB",
			wantErr:   false,
		},
		{
			name:      "podman stats error",
			runtime:   "podman",
			output:    "",
			err:       true,
			wantStats: "",
			wantErr:   true,
		},
		{
			name:      "docker stats error",
			runtime:   "docker",
			output:    "",
			err:       true,
			wantStats: "",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := platform.NewMockRunner()
			key := tc.runtime + " [stats --no-stream --format CPU: {{.CPUPerc}}  MEM: {{.MemUsage}} minecraft]"
			m.OutputMap[key] = []byte(tc.output)
			if tc.err {
				m.ErrorMap[key] = errors.New("mock error")
			}

			mgr := NewManager(m, tc.runtime, "minecraft", "localhost:25575", "testpass")
			got, err := mgr.Stats(context.Background())
			if (err != nil) != tc.wantErr {
				t.Errorf("Stats() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if got != tc.wantStats {
				t.Errorf("Stats() = %q, want %q", got, tc.wantStats)
			}
		})
	}
}

func TestManager_Start_Error(t *testing.T) {
	m := platform.NewMockRunner()
	key := "podman [start minecraft]"
	m.ErrorMap[key] = errors.New("container not found")
	mgr := NewManager(m, "podman", "minecraft", "", "")

	err := mgr.Start(context.Background(), "ignored")
	if err == nil {
		t.Fatal("Start() expected error, got nil")
	}
}

func TestManager_Stop_Error(t *testing.T) {
	m := platform.NewMockRunner()
	key := "podman [stop -t 60 minecraft]"
	m.ErrorMap[key] = errors.New("podman error")
	mgr := NewManager(m, "podman", "minecraft", "", "")

	err := mgr.Stop(context.Background())
	if err == nil {
		t.Fatal("Stop() expected error, got nil")
	}
}

func TestManager_Stats_Error(t *testing.T) {
	m := platform.NewMockRunner()
	key := "podman [stats --no-stream --format CPU: {{.CPUPerc}}  MEM: {{.MemUsage}} minecraft]"
	m.ErrorMap[key] = errors.New("container not running")
	mgr := NewManager(m, "podman", "minecraft", "", "")

	_, err := mgr.Stats(context.Background())
	if err == nil {
		t.Fatal("Stats() expected error, got nil")
	}
}

func TestManager_SendCommand(t *testing.T) {
	// Start a test RCON server.
	srv := newRCONTestServer(t, "rconpass", func(cmd string) string {
		return "done:" + cmd
	})
	defer srv.Close()
	go srv.Serve(t)

	m := platform.NewMockRunner()
	mgr := NewManager(m, "podman", "minecraft", srv.Addr(), "rconpass")

	if err := mgr.SendCommand(context.Background(), "say hello"); err != nil {
		t.Fatalf("SendCommand() error = %v", err)
	}
}

func TestManager_SendCommand_ConnectFailure(t *testing.T) {
	m := platform.NewMockRunner()
	// Allocate an ephemeral port, then close it so that dialing will fail.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("listener Close() error = %v", err)
	}

	mgr := NewManager(m, "podman", "minecraft", addr, "pass")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = mgr.SendCommand(ctx, "list")
	if err == nil {
		t.Fatal("SendCommand() expected error, got nil")
	}
}
