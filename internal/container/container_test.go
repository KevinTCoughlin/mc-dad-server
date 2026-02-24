package container

import (
	"context"
	"fmt"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// Compile-time checks that Manager satisfies both interfaces.
var (
	_ management.ServerManager = (*Manager)(nil)
	_ management.HealthChecker = (*Manager)(nil)
)

func TestManager_IsRunning(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.OutputMap["podman [inspect --format {{.State.Running}} minecraft]"] = []byte("true\n")
	mgr := NewManager(mock, "minecraft", "127.0.0.1:25575", "secret")

	if !mgr.IsRunning(context.Background()) {
		t.Error("IsRunning() = false, want true")
	}
}

func TestManager_IsRunning_NotRunning(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.OutputMap["podman [inspect --format {{.State.Running}} minecraft]"] = []byte("false\n")
	mgr := NewManager(mock, "minecraft", "127.0.0.1:25575", "secret")

	if mgr.IsRunning(context.Background()) {
		t.Error("IsRunning() = true, want false")
	}
}

func TestManager_Session(t *testing.T) {
	mock := platform.NewMockRunner()
	mgr := NewManager(mock, "mycontainer", "127.0.0.1:25575", "secret")

	if got := mgr.Session(); got != "mycontainer" {
		t.Errorf("Session() = %q, want %q", got, "mycontainer")
	}
}

func TestManager_Health(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.OutputMap["podman [inspect --format {{.State.Healthcheck.Status}} minecraft]"] = []byte("healthy\n")
	mgr := NewManager(mock, "minecraft", "127.0.0.1:25575", "secret")

	if got := mgr.Health(context.Background()); got != "healthy" {
		t.Errorf("Health() = %q, want %q", got, "healthy")
	}
}

func TestManager_Health_Error(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.ErrorMap["podman [inspect --format {{.State.Healthcheck.Status}} minecraft]"] = fmt.Errorf("not found")
	mgr := NewManager(mock, "minecraft", "127.0.0.1:25575", "secret")

	if got := mgr.Health(context.Background()); got != "unknown" {
		t.Errorf("Health() = %q, want %q", got, "unknown")
	}
}

func TestManager_Stats(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.OutputMap["podman [stats --no-stream --format CPU: {{.CPUPerc}}  MEM: {{.MemUsage}} minecraft]"] = []byte("CPU: 5.2%  MEM: 1.2GiB / 3GiB\n")
	mgr := NewManager(mock, "minecraft", "127.0.0.1:25575", "secret")

	got, err := mgr.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	want := "CPU: 5.2%  MEM: 1.2GiB / 3GiB"
	if got != want {
		t.Errorf("Stats() = %q, want %q", got, want)
	}
}

func TestManager_Stats_Error(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.ErrorMap["podman [stats --no-stream --format CPU: {{.CPUPerc}}  MEM: {{.MemUsage}} minecraft]"] = fmt.Errorf("not found")
	mgr := NewManager(mock, "minecraft", "127.0.0.1:25575", "secret")

	_, err := mgr.Stats(context.Background())
	if err == nil {
		t.Error("Stats() error = nil, want error")
	}
}
