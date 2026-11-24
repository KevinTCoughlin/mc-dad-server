//go:build windows

package platform

import "golang.org/x/sys/windows"

func freeBytes(path string) (uint64, error) {
	var free uint64
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	if err := windows.GetDiskFreeSpaceEx(p, &free, nil, nil); err != nil {
		return 0, err
	}
	return free, nil
}
