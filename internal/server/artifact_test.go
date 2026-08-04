package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sha256 / sha1 of "hello\n"
const (
	helloSHA256 = "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"
	helloSHA1   = "f572d396fae9206628714fb2ce00f72e94f2258f"
)

func writeHello(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "server.jar")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return path
}

func TestArtifactVerify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		art     Artifact
		wantErr string
	}{
		{name: "sha256 match", art: Artifact{SHA256: helloSHA256}},
		{name: "sha256 uppercase match", art: Artifact{SHA256: strings.ToUpper(helloSHA256)}},
		{name: "sha1 match", art: Artifact{SHA1: helloSHA1}},
		{name: "no checksum published", art: Artifact{}},
		{
			name:    "sha256 mismatch",
			art:     Artifact{SHA256: strings.Repeat("0", 64)},
			wantErr: "sha256 checksum mismatch",
		},
		{
			name:    "sha1 mismatch",
			art:     Artifact{SHA1: strings.Repeat("0", 40)},
			wantErr: "sha1 checksum mismatch",
		},
		{
			name:    "sha256 preferred over sha1",
			art:     Artifact{SHA256: strings.Repeat("0", 64), SHA1: helloSHA1},
			wantErr: "sha256 checksum mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.art.Verify(writeHello(t))
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected checksum error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestArtifactVerifyMissingFile(t *testing.T) {
	t.Parallel()

	art := Artifact{SHA256: helloSHA256}
	if err := art.Verify(filepath.Join(t.TempDir(), "absent.jar")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
