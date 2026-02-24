package container

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

func TestManager_SendCommand_PersistentConnection(t *testing.T) {
	srv := newFakeRCON(t, "pass")
	defer srv.close()

	mock := platform.NewMockRunner()
	mgr := NewManager(mock, "minecraft", srv.addr(), "pass")
	defer func() { _ = mgr.Close() }()

	ctx := context.Background()

	// Send multiple commands — should reuse the same connection.
	for i := 0; i < 5; i++ {
		if err := mgr.SendCommand(ctx, "list"); err != nil {
			t.Fatalf("SendCommand() #%d error = %v", i, err)
		}
	}

	cmds := srv.getCommands()
	if len(cmds) != 5 {
		t.Errorf("server received %d commands, want 5", len(cmds))
	}
}

func TestManager_SendCommand_LazyConnect(t *testing.T) {
	srv := newFakeRCON(t, "pass")
	defer srv.close()

	mock := platform.NewMockRunner()
	mgr := NewManager(mock, "minecraft", srv.addr(), "pass")
	defer func() { _ = mgr.Close() }()

	// rcon should be nil before first SendCommand.
	mgr.mu.Lock()
	if mgr.rcon != nil {
		t.Error("rcon should be nil before first SendCommand")
	}
	mgr.mu.Unlock()

	if err := mgr.SendCommand(context.Background(), "say hello"); err != nil {
		t.Fatalf("SendCommand() error = %v", err)
	}

	// rcon should be non-nil after first SendCommand.
	mgr.mu.Lock()
	if mgr.rcon == nil {
		t.Error("rcon should be non-nil after SendCommand")
	}
	mgr.mu.Unlock()
}

func TestManager_SendCommand_ReconnectsOnBrokenConnection(t *testing.T) {
	srv := newFakeRCON(t, "pass")
	defer srv.close()

	mock := platform.NewMockRunner()
	mgr := NewManager(mock, "minecraft", srv.addr(), "pass")
	defer func() { _ = mgr.Close() }()

	ctx := context.Background()

	// Establish the connection.
	if err := mgr.SendCommand(ctx, "say first"); err != nil {
		t.Fatalf("first SendCommand() error = %v", err)
	}

	// Simulate a broken connection by closing the underlying RCON client.
	mgr.mu.Lock()
	_ = mgr.rcon.Close()
	mgr.mu.Unlock()

	// The next command should trigger a reconnect.
	if err := mgr.SendCommand(ctx, "say second"); err != nil {
		t.Fatalf("SendCommand() after broken connection error = %v", err)
	}

	cmds := srv.getCommands()
	if len(cmds) != 2 {
		t.Errorf("server received %d commands, want 2", len(cmds))
	}
}

func TestManager_Close_Idempotent(t *testing.T) {
	srv := newFakeRCON(t, "pass")
	defer srv.close()

	mock := platform.NewMockRunner()
	mgr := NewManager(mock, "minecraft", srv.addr(), "pass")

	if err := mgr.SendCommand(context.Background(), "list"); err != nil {
		t.Fatalf("SendCommand() error = %v", err)
	}

	if err := mgr.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := mgr.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestManager_Close_WithoutConnect(t *testing.T) {
	mock := platform.NewMockRunner()
	mgr := NewManager(mock, "minecraft", "127.0.0.1:0", "pass")

	if err := mgr.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestManager_SendCommand_ConnectFailure(t *testing.T) {
	mock := platform.NewMockRunner()
	mgr := NewManager(mock, "minecraft", "127.0.0.1:0", "pass")

	err := mgr.SendCommand(context.Background(), "list")
	if err == nil {
		t.Fatal("SendCommand() should fail when RCON server is unreachable")
	}
}

func TestManager_IsRunning(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		output string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := platform.NewMockRunner()
			mock.OutputMap["podman [inspect --format {{.State.Running}} minecraft]"] = []byte(tt.output)
			mgr := NewManager(mock, "minecraft", "127.0.0.1:25575", "pass")

			if got := mgr.IsRunning(ctx); got != tt.want {
				t.Errorf("IsRunning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_Session(t *testing.T) {
	mock := platform.NewMockRunner()
	mgr := NewManager(mock, "mycontainer", "127.0.0.1:25575", "pass")
	if got := mgr.Session(); got != "mycontainer" {
		t.Errorf("Session() = %q, want %q", got, "mycontainer")
	}
}

func TestManager_Exists(t *testing.T) {
	ctx := context.Background()

	t.Run("exists", func(t *testing.T) {
		mock := platform.NewMockRunner()
		if !Exists(ctx, mock, "minecraft") {
			t.Error("Exists() = false, want true")
		}
	})

	t.Run("does not exist", func(t *testing.T) {
		mock := platform.NewMockRunner()
		mock.ErrorMap["podman [inspect --type container minecraft]"] = errors.New("no such container")
		if Exists(ctx, mock, "minecraft") {
			t.Error("Exists() = true, want false")
		}
	})
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "EOF", err: io.EOF, want: true},
		{name: "UnexpectedEOF", err: io.ErrUnexpectedEOF, want: true},
		{name: "net.OpError", err: &net.OpError{Op: "write", Err: errors.New("broken pipe")}, want: true},
		{name: "broken pipe text", err: errors.New("write: broken pipe"), want: true},
		{name: "connection reset text", err: errors.New("read: connection reset by peer"), want: true},
		{name: "closed conn", err: errors.New("use of closed network connection"), want: true},
		{name: "not connected", err: errors.New("rcon not connected"), want: true},
		{name: "other error", err: errors.New("some other error"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isConnectionError(tt.err); got != tt.want {
				t.Errorf("isConnectionError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
