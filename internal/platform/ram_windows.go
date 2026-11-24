//go:build windows

package platform

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// memoryStatusEx matches MEMORYSTATUSEX. The field is not wrapped by
// golang.org/x/sys v0.50, so the installer calls kernel32 directly.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var procGlobalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

// TotalRAM returns the machine's physical memory in bytes.
func TotalRAM() (uint64, error) {
	var status memoryStatusEx
	status.Length = uint32(unsafe.Sizeof(status))
	r, _, callErr := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if r == 0 {
		return 0, fmt.Errorf("reading memory status: %w", callErr)
	}
	if status.TotalPhys == 0 {
		return 0, fmt.Errorf("memory status reported 0 bytes")
	}
	return status.TotalPhys, nil
}
