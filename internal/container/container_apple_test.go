package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

const appleInspectKey = "container [inspect minecraft]"

func appleInspectJSON(state string) []byte {
	return fmt.Appendf(nil, `[{"id":"minecraft","status":{"state":%q,"networks":[]}}]`, state)
}

func TestAppleManager_IsRunning(t *testing.T) {
	tests := []struct {
		name string
		out  []byte
		err  error
		want bool
	}{
		{"running", appleInspectJSON("running"), nil, true},
		{"stopped", appleInspectJSON("stopped"), nil, false},
		{"not found", nil, errors.New("container not found"), false},
		{"malformed json", []byte("not json"), nil, false},
		{"empty array", []byte("[]"), nil, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := platform.NewMockRunner()
			if tc.err != nil {
				m.ErrorMap[appleInspectKey] = tc.err
			} else {
				m.OutputMap[appleInspectKey] = tc.out
			}
			mgr := NewManager(m, platform.RuntimeApple, "minecraft", "", "")
			if got := mgr.IsRunning(context.Background()); got != tc.want {
				t.Errorf("IsRunning() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAppleManager_Health(t *testing.T) {
	tests := []struct {
		name string
		out  []byte
		err  error
		want string
	}{
		{"running", appleInspectJSON("running"), nil, "running"},
		{"stopped", appleInspectJSON("stopped"), nil, "stopped"},
		{"inspect error", nil, errors.New("boom"), "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := platform.NewMockRunner()
			if tc.err != nil {
				m.ErrorMap[appleInspectKey] = tc.err
			} else {
				m.OutputMap[appleInspectKey] = tc.out
			}
			mgr := NewManager(m, platform.RuntimeApple, "minecraft", "", "")
			if got := mgr.Health(context.Background()); got != tc.want {
				t.Errorf("Health() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAppleManager_Stats(t *testing.T) {
	m := platform.NewMockRunner()
	m.OutputMap["container [stats --no-stream --format json minecraft]"] = []byte(
		`[{"cpuUtilization":12.5,"memoryUsageBytes":1610612736,"memoryLimitBytes":3221225472,"id":"minecraft"}]`)
	mgr := NewManager(m, platform.RuntimeApple, "minecraft", "", "")

	got, err := mgr.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error: %v", err)
	}
	if want := "CPU: 12.50%  MEM: 1.50GiB / 3.00GiB"; got != want {
		t.Errorf("Stats() = %q, want %q", got, want)
	}
}

func TestAppleManager_StatsErrors(t *testing.T) {
	key := "container [stats --no-stream --format json minecraft]"
	for name, out := range map[string][]byte{"empty": []byte("[]"), "garbage": []byte("nope")} {
		t.Run(name, func(t *testing.T) {
			m := platform.NewMockRunner()
			m.OutputMap[key] = out
			mgr := NewManager(m, platform.RuntimeApple, "minecraft", "", "")
			if _, err := mgr.Stats(context.Background()); err == nil {
				t.Error("Stats() expected error")
			}
		})
	}
}

func TestAppleManager_LaunchPrefersLaunchd(t *testing.T) {
	m := platform.NewMockRunner()
	mgr := NewManager(m, platform.RuntimeApple, "minecraft", "", "")
	if err := mgr.Launch(context.Background()); err != nil {
		t.Fatalf("Launch() error: %v", err)
	}
	if len(m.Commands) != 1 || m.Commands[0].Name != "launchctl" {
		t.Fatalf("expected a single launchctl call, got %+v", m.Commands)
	}
	wantAgent := fmt.Sprintf("gui/%d/%s", os.Getuid(), LaunchdLabel)
	if args := m.Commands[0].Args; len(args) != 2 || args[0] != "kickstart" || args[1] != wantAgent {
		t.Errorf("launchctl args = %v, want [kickstart %s]", args, wantAgent)
	}
}

func TestAppleManager_LaunchFallsBackToContainerStart(t *testing.T) {
	m := platform.NewMockRunner()
	agent := fmt.Sprintf("gui/%d/%s", os.Getuid(), LaunchdLabel)
	m.ErrorMap["launchctl [kickstart "+agent+"]"] = errors.New("no such service")
	mgr := NewManager(m, platform.RuntimeApple, "minecraft", "", "")
	if err := mgr.Launch(context.Background()); err != nil {
		t.Fatalf("Launch() error: %v", err)
	}
	last := m.Commands[len(m.Commands)-1]
	if last.Name != "container" || len(last.Args) != 2 || last.Args[0] != "start" || last.Args[1] != "minecraft" {
		t.Errorf("fallback command = %s %v, want container [start minecraft]", last.Name, last.Args)
	}
}

func TestAppleManager_Stop(t *testing.T) {
	m := platform.NewMockRunner()
	mgr := NewManager(m, platform.RuntimeApple, "minecraft", "", "")
	if err := mgr.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}
	if got := fmt.Sprintf("%s %v", m.Commands[0].Name, m.Commands[0].Args); got != "container [stop -t 60 minecraft]" {
		t.Errorf("Stop() ran %q", got)
	}
}

func TestAppleExists(t *testing.T) {
	m := platform.NewMockRunner()
	if !Exists(context.Background(), m, platform.RuntimeApple, "minecraft") {
		t.Error("Exists() = false, want true")
	}
	if got := fmt.Sprintf("%v", m.Commands[0].Args); got != "[inspect minecraft]" {
		t.Errorf("Exists() args = %s, want [inspect minecraft]", got)
	}

	m = platform.NewMockRunner()
	m.ErrorMap[appleInspectKey] = errors.New("container not found")
	if Exists(context.Background(), m, platform.RuntimeApple, "minecraft") {
		t.Error("Exists() = true for missing container")
	}
}
