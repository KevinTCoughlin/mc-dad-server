//go:build linux

package platform

import (
	"fmt"
	"os"
)

// TotalRAM returns the machine's physical memory in bytes.
func TotalRAM() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("reading /proc/meminfo: %w", err)
	}
	total, err := parseMemTotal(string(data))
	if err != nil {
		return 0, err
	}
	return total, nil
}
