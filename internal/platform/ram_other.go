//go:build !linux && !darwin && !windows

package platform

import "fmt"

// TotalRAM returns the machine's physical memory in bytes.
func TotalRAM() (uint64, error) {
	return 0, fmt.Errorf("system RAM detection is not implemented on this OS")
}
