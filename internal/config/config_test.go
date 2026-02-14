package config

import (
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
