package platform

import "testing"

func TestParseMemTotal(t *testing.T) {
	t.Parallel()

	got, err := parseMemTotal("MemFree: 100 kB\nMemTotal:       16384000 kB\nBuffers: 1 kB\n")
	if err != nil {
		t.Fatalf("parseMemTotal() error = %v", err)
	}
	// meminfo's "kB" is kibibytes.
	if want := uint64(16384000) * 1024; got != want {
		t.Fatalf("parseMemTotal() = %d, want %d", got, want)
	}
}

func TestParseMemTotalErrors(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"", "MemFree: 1 kB\n", "MemTotal:\n", "MemTotal: no kB\n"} {
		if _, err := parseMemTotal(text); err == nil {
			t.Fatalf("parseMemTotal(%q) succeeded", text)
		}
	}
}

func TestTotalRAMReadsThisMachine(t *testing.T) {
	total, err := TotalRAM()
	if err != nil {
		t.Fatalf("TotalRAM() error = %v", err)
	}
	if total < 64*1024*1024 {
		t.Fatalf("TotalRAM() = %d, want at least 64 MiB", total)
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	if got := FormatBytes(4 * 1024 * 1024 * 1024); got != "4 GiB" {
		t.Fatalf("FormatBytes(4GiB) = %q", got)
	}
	if got := FormatBytes(512 * 1024 * 1024); got != "512 MiB" {
		t.Fatalf("FormatBytes(512MiB) = %q", got)
	}
}
