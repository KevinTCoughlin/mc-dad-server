package platform

import (
	"context"
	"testing"
)

func TestMockRunner_Run(t *testing.T) {
	m := NewMockRunner()
	ctx := context.Background()

	if err := m.Run(ctx, "echo", "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(m.Commands))
	}
	if m.Commands[0].Name != "echo" {
		t.Fatalf("expected echo, got %s", m.Commands[0].Name)
	}
}

func TestMockRunner_RunWithOutput(t *testing.T) {
	m := NewMockRunner()
	m.OutputMap[m.Key("cat", "/etc/hostname")] = []byte("testhost\n")

	out, err := m.RunWithOutput(context.Background(), "cat", "/etc/hostname")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != "testhost\n" {
		t.Fatalf("expected testhost, got %s", out)
	}
}

func TestMockRunner_CommandExists(t *testing.T) {
	m := NewMockRunner()
	m.ExistsMap["screen"] = true

	if !m.CommandExists("screen") {
		t.Fatal("expected screen to exist")
	}
	if m.CommandExists("nonexistent") {
		t.Fatal("expected nonexistent to not exist")
	}
}

func TestMockRunner_RunSudo(t *testing.T) {
	m := NewMockRunner()
	if err := m.RunSudo(context.Background(), "apt-get", "install", "-y", "screen"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(m.Commands))
	}
	if !m.Commands[0].Sudo {
		t.Fatal("expected sudo flag to be set")
	}
}
