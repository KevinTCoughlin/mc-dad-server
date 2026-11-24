package config

import "testing"

func TestIsAutoMemory(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"", "auto", "AUTO", " Auto "} {
		if !IsAutoMemory(s) {
			t.Errorf("IsAutoMemory(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"2G", "4G", "512M", "automatic"} {
		if IsAutoMemory(s) {
			t.Errorf("IsAutoMemory(%q) = true, want false", s)
		}
	}
}

func TestHeapFromRAM(t *testing.T) {
	t.Parallel()

	const (
		mib = 1024 * 1024
		gib = 1024 * mib
	)

	tests := []struct {
		name  string
		total uint64
		want  string
		tight bool
	}{
		{name: "512 MiB floors at 512M", total: 512 * mib, want: "512M", tight: true},
		{name: "640 MiB still floors at 512M", total: 640 * mib, want: "512M", tight: true},
		{name: "1 GiB keeps 512M for the OS", total: 1 * gib, want: "512M", tight: true},
		{name: "1.5 GiB uses half, aligned", total: 1536 * mib, want: "768M", tight: true},
		{name: "1792 MiB is already aligned", total: 1792 * mib, want: "896M", tight: true},
		{name: "1700 MiB rounds down to 128 MiB", total: 1700 * mib, want: "768M", tight: true},
		{name: "2 GiB reserves 1 GiB", total: 2 * gib, want: "1G"},
		{name: "2.5 GiB reserves 1 GiB and rounds down", total: 2560 * mib, want: "1G"},
		{name: "3 GiB", total: 3 * gib, want: "2G"},
		{name: "4 GiB", total: 4 * gib, want: "3G"},
		{name: "6 GiB rounds the remainder down", total: 6 * gib, want: "4G"},
		{name: "8 GiB", total: 8 * gib, want: "6G"},
		{name: "16 GiB hits the 12 GiB cap via headroom", total: 16 * gib, want: "12G"},
		{name: "32 GiB is clamped to 12 GiB", total: 32 * gib, want: "12G"},
		{name: "64 GiB is clamped to 12 GiB", total: 64 * gib, want: "12G"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, tight, err := HeapFromRAM(tt.total)
			if err != nil {
				t.Fatalf("HeapFromRAM() error = %v", err)
			}
			if got != tt.want || tight != tt.tight {
				t.Fatalf("HeapFromRAM() = (%q, tight=%v), want (%q, tight=%v)", got, tight, tt.want, tt.tight)
			}
		})
	}
}

func TestHeapFromRAMRejectsZero(t *testing.T) {
	t.Parallel()

	_, _, err := HeapFromRAM(0)
	if err == nil {
		t.Fatal("expected an error for 0 RAM")
	}
}

func TestHeapFromRAMRespectsOverrideRange(t *testing.T) {
	t.Parallel()

	// Every automatic result must still pass config validation, so an
	// operator is never handed a heap the rest of the installer rejects.
	for _, total := range []uint64{
		512 * byteMiB,
		byteGiB,
		2 * byteGiB,
		4 * byteGiB,
		8 * byteGiB,
		32 * byteGiB,
	} {
		heap, _, err := HeapFromRAM(total)
		if err != nil {
			t.Fatalf("total %d: %v", total, err)
		}
		cfg := DefaultConfig()
		cfg.Dir = "/srv/minecraft"
		cfg.Memory = heap
		if err := cfg.Validate(); err != nil {
			t.Fatalf("heap %q from %d bytes failed validation: %v", heap, total, err)
		}
	}
}
