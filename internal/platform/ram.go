package platform

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatBytes renders n as a whole GiB or MiB value for operator messages.
func FormatBytes(n uint64) string {
	const (
		gib = 1024 * 1024 * 1024
		mib = 1024 * 1024
	)
	switch {
	case n >= gib && n%gib == 0:
		return fmt.Sprintf("%d GiB", n/gib)
	case n >= gib:
		return fmt.Sprintf("%.1f GiB", float64(n)/float64(gib))
	case n >= mib:
		return fmt.Sprintf("%d MiB", n/mib)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// parseMemTotal reads the MemTotal field from a /proc/meminfo document.
// The kernel reports that field in KiB.
func parseMemTotal(text string) (uint64, error) {
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("malformed MemTotal line %q", line)
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parsing MemTotal %q: %w", fields[1], err)
		}
		return kb * 1024, nil
	}
	return 0, fmt.Errorf("MemTotal not found in meminfo")
}
