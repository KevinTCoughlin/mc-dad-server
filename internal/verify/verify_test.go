package verify

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// digests of "hello\n"
const (
	helloSHA256 = "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"
	helloSHA1   = "f572d396fae9206628714fb2ce00f72e94f2258f"
	helloSize   = int64(6)
)

func writeHello(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artifact.bin")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	return path
}

func TestFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		exp     Expected
		wantErr string
	}{
		{name: "sha256 match", exp: Expected{SHA256: helloSHA256}},
		{name: "sha256 uppercase", exp: Expected{SHA256: strings.ToUpper(helloSHA256)}},
		{name: "sha1 match", exp: Expected{SHA1: helloSHA1}},
		{name: "size match", exp: Expected{Size: helloSize}},
		{name: "size and digest match", exp: Expected{SHA256: helloSHA256, Size: helloSize}},
		{name: "nothing published", exp: Expected{}},
		{
			name:    "sha256 mismatch",
			exp:     Expected{SHA256: strings.Repeat("0", 64)},
			wantErr: "sha256 checksum mismatch",
		},
		{
			name:    "sha1 mismatch",
			exp:     Expected{SHA1: strings.Repeat("0", 40)},
			wantErr: "sha1 checksum mismatch",
		},
		{
			name:    "size mismatch",
			exp:     Expected{Size: 999},
			wantErr: "size mismatch",
		},
		{
			name:    "sha256 preferred over sha1",
			exp:     Expected{SHA256: strings.Repeat("0", 64), SHA1: helloSHA1},
			wantErr: "sha256 checksum mismatch",
		},
		{
			name:    "size checked even when digest is good",
			exp:     Expected{SHA256: helloSHA256, Size: 999},
			wantErr: "size mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := File(writeHello(t), tt.exp)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestFileMissing(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "absent.bin")

	if err := File(missing, Expected{SHA256: helloSHA256}); err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if err := File(missing, Expected{Size: helloSize}); err == nil {
		t.Fatal("expected an error for a missing file")
	}
	// Nothing to check means nothing to open.
	if err := File(missing, Expected{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExpectedEmptyAndDescribe(t *testing.T) {
	t.Parallel()

	if !(Expected{}).Empty() {
		t.Fatal("zero Expected should be empty")
	}
	for _, e := range []Expected{{SHA256: "x"}, {SHA1: "x"}, {Size: 1}} {
		if e.Empty() {
			t.Fatalf("%+v should not be empty", e)
		}
	}

	tests := map[string]Expected{
		"sha256":  {SHA256: "x", SHA1: "y", Size: 1},
		"sha1":    {SHA1: "y", Size: 1},
		"size":    {Size: 1},
		"nothing": {},
	}
	for want, e := range tests {
		if got := e.Describe(); got != want {
			t.Fatalf("Describe() = %q, want %q", got, want)
		}
	}
}
