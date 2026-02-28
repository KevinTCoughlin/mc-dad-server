package container

import (
	"context"
	"errors"
	"testing"

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

func TestManager_Start(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
	}{
		{
			name:    "start with podman",
			runtime: "podman",
		},
		{
			name:    "start with docker",
			runtime: "docker",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := platform.NewMockRunner()
			mgr := NewManager(m, tc.runtime, "minecraft", "localhost:25575", "testpass")

			err := mgr.Start(context.Background(), "ignored", "args")
			if err != nil {
				t.Errorf("Start() error = %v", err)
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
