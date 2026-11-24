package config

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// byteKiB is the unit /proc/meminfo uses for MemTotal.
	byteKiB = 1024
	byteMiB = 1024 * byteKiB
	byteGiB = 1024 * byteMiB

	// smallRAM is the point below which a normal headroom reservation would
	// leave no usable heap. Those machines take a separate, tighter policy.
	smallRAM = 2 * byteGiB
	// minSmallHeap is the smallest heap Paper 26.3 can boot. Below 2 GiB of
	// RAM this leaves the OS very little room, and callers should say so.
	minSmallHeap = 512 * byteMiB
	smallStep    = 128 * byteMiB
	// smallReserve is memory kept for the OS on machines under smallRAM,
	// when half of RAM would eat into it.
	smallReserve = 256 * byteMiB

	// minHeap is the smallest heap once the machine has room for the OS.
	minHeap = 1 * byteGiB
	// maxHeap caps a family server. Larger heaps need their own G1 region
	// tuning; 12 GiB keeps the shipped flags predictable.
	maxHeap = 12 * byteGiB
	// minHeadroom and maxHeadroom bound the 25% OS reservation.
	minHeadroom = 1 * byteGiB
	maxHeadroom = 4 * byteGiB

	// FallbackHeap is used when RAM cannot be detected. It matches the
	// historical --memory default.
	FallbackHeap = "2G"
)

// IsAutoMemory reports whether the operator asked the installer to choose
// the heap. An empty value means the same thing as "auto".
func IsAutoMemory(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return true
	default:
		return false
	}
}

// HeapFromRAM chooses a JVM -Xms/-Xmx value from physical RAM.
//
// On machines with at least 2 GiB, headroom is 25% of RAM clamped to
// [1 GiB, 4 GiB]. The heap is the remainder, rounded down to a whole
// gibibyte, then clamped to [1 GiB, 12 GiB]. Rounding down leaves the spare
// mebibytes with the OS.
//
// Below 2 GiB the heap is half of RAM, rounded down to 128 MiB, and never
// below 512 MiB. tight is true on that path: the OS is competing with the
// server for the same small machine.
func HeapFromRAM(total uint64) (heap string, tight bool, err error) {
	if total == 0 {
		return "", false, errors.New("system RAM size is 0")
	}
	if total < smallRAM {
		return formatHeap(smallMachineHeap(total)), true, nil
	}

	headroom := total / 4
	if headroom < minHeadroom {
		headroom = minHeadroom
	}
	if headroom > maxHeadroom {
		headroom = maxHeadroom
	}

	var sized uint64 = minHeap
	if total > headroom {
		sized = total - headroom
	}
	if sized < minHeap {
		sized = minHeap
	}
	if sized > maxHeap {
		sized = maxHeap
	}
	sized = sized / byteGiB * byteGiB
	if sized < minHeap {
		sized = minHeap
	}
	return formatHeap(sized), false, nil
}

func smallMachineHeap(total uint64) uint64 {
	heap := total / 2
	heap = heap / smallStep * smallStep
	if total > smallReserve && heap > total-smallReserve {
		reduced := (total - smallReserve) / smallStep * smallStep
		if reduced > 0 {
			heap = reduced
		}
	}
	if heap < minSmallHeap {
		heap = minSmallHeap
	}
	return heap
}

func formatHeap(n uint64) string {
	if n >= byteGiB && n%byteGiB == 0 {
		return fmt.Sprintf("%dG", n/byteGiB)
	}
	mib := n / byteMiB
	if mib == 0 {
		mib = 1
	}
	return fmt.Sprintf("%dM", mib)
}
