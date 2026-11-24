//go:build !linux && !darwin && !windows

package platform

import "fmt"

func freeBytes(path string) (uint64, error) {
	return 0, fmt.Errorf("disk space checks are not implemented on this OS (%s)", path)
}
