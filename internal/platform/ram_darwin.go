//go:build darwin

package platform

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// TotalRAM returns the machine's physical memory in bytes.
func TotalRAM() (uint64, error) {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0, fmt.Errorf("reading hw.memsize: %w", err)
	}
	if total == 0 {
		return 0, fmt.Errorf("hw.memsize reported 0")
	}
	return total, nil
}
