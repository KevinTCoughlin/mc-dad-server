package management

import (
	"context"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

func TestGetProcessStats_Success(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.OutputMap["pgrep [-f server.jar]"] = []byte("12345\n")
	mock.OutputMap["ps [-o rss= -p 12345]"] = []byte("524288\n")
	mock.OutputMap["ps [-o %cpu= -p 12345]"] = []byte("15.3\n")

	stats, err := GetProcessStats(context.Background(), mock)
	if err != nil {
		t.Fatalf("GetProcessStats() error = %v", err)
	}
	if stats.PID != 12345 {
		t.Errorf("PID = %d, want 12345", stats.PID)
	}
	if stats.Memory != "512 MB" {
		t.Errorf("Memory = %q, want %q", stats.Memory, "512 MB")
	}
	if stats.CPU != "15.3%" {
		t.Errorf("CPU = %q, want %q", stats.CPU, "15.3%")
	}
}

func TestGetProcessStats_NotRunning(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.ErrorMap["pgrep [-f server.jar]"] = context.DeadlineExceeded

	_, err := GetProcessStats(context.Background(), mock)
	if err == nil {
		t.Fatal("expected error when server not running")
	}
}

func TestGetProcessStats_InvalidPID(t *testing.T) {
	mock := platform.NewMockRunner()
	mock.OutputMap["pgrep [-f server.jar]"] = []byte("not-a-number\n")

	_, err := GetProcessStats(context.Background(), mock)
	if err == nil {
		t.Fatal("expected error for invalid PID")
	}
}
