package platform

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestCheckPortAvailable(t *testing.T) {
	ctx := context.Background()
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if err := CheckPortAvailable(ctx, port); err != nil {
		t.Fatalf("free port %d: %v", port, err)
	}

	busy, err := lc.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("relisten: %v", err)
	}
	defer func() { _ = busy.Close() }()

	err = CheckPortAvailable(ctx, port)
	if err == nil {
		t.Fatal("expected port-in-use error")
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("port %d", port)) || !strings.Contains(err.Error(), "--port") {
		t.Fatalf("error = %q, want the port and the --port hint", err)
	}
}

func TestCheckDiskSpace(t *testing.T) {
	dir := t.TempDir()
	if err := CheckDiskSpace(dir, 1); err != nil {
		t.Fatalf("1 byte should fit in %s: %v", dir, err)
	}
	err := CheckDiskSpace(dir, ^uint64(0))
	if err == nil {
		t.Fatal("expected a disk space error")
	}
	msg := err.Error()
	if !strings.Contains(msg, dir) || !strings.Contains(msg, "not enough free disk space") || !strings.Contains(msg, "--dir") {
		t.Fatalf("error = %q", err)
	}
}
