package management

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

func TestScreenManager_IsRunning(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		output  string
		wantErr bool
		want    bool
	}{
		{
			name:   "session exists",
			output: "There is a screen on:\n\t12345.minecraft\t(Detached)\n1 Socket in /run/screen.",
			want:   true,
		},
		{
			name:   "no session",
			output: "No Sockets found in /run/screen.",
			want:   false,
		},
		{
			name:   "different session name",
			output: "There is a screen on:\n\t12345.other\t(Detached)\n1 Socket in /run/screen.",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := platform.NewMockRunner()
			mock.OutputMap["screen [-list]"] = []byte(tt.output)
			sm := NewScreenManager(mock, "minecraft")

			if got := sm.IsRunning(ctx); got != tt.want {
				t.Errorf("IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScreenManager_IsRunning_Error(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.ErrorMap["screen [-list]"] = context.DeadlineExceeded
	sm := NewScreenManager(mock, "minecraft")

	if sm.IsRunning(context.Background()) {
		t.Error("IsRunning() should return false on error")
	}
}

func TestScreenManager_SendCommand(t *testing.T) {
	mock := platform.NewMockRunner()
	sm := NewScreenManager(mock, "minecraft")

	if err := sm.SendCommand(context.Background(), "say hello"); err != nil {
		t.Fatalf("SendCommand() error = %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if cmd.Name != "screen" {
		t.Errorf("command name = %q, want %q", cmd.Name, "screen")
	}
	// Last arg should contain the command with \r
	lastArg := cmd.Args[len(cmd.Args)-1]
	if lastArg != "say hello\r" {
		t.Errorf("last arg = %q, want %q", lastArg, "say hello\r")
	}
}

func TestScreenManager_Start(t *testing.T) {
	mock := platform.NewMockRunner()
	sm := NewScreenManager(mock, "minecraft")

	if err := sm.Start(context.Background(), "bash", "/srv/start.sh"); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if len(mock.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(mock.Commands))
	}
	cmd := mock.Commands[0]
	if cmd.Name != "screen" {
		t.Errorf("command name = %q, want %q", cmd.Name, "screen")
	}
}

func TestScreenManager_Session(t *testing.T) {
	mock := platform.NewMockRunner()
	sm := NewScreenManager(mock, "myserver")
	if got := sm.Session(); got != "myserver" {
		t.Errorf("Session() = %q, want %q", got, "myserver")
	}
}

func TestSleep_Completes(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	if err := Sleep(ctx, 0); err != nil {
		t.Fatalf("Sleep() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Sleep(0) took %v, expected near-instant", elapsed)
	}
}

func TestSleep_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Sleep(ctx, 60); !errors.Is(err, context.Canceled) {
		t.Errorf("Sleep() error = %v, want context.Canceled", err)
	}
}
