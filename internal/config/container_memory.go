package config

import (
	"fmt"
	"strings"
)

// ContainerHeapPercent is the share of the container memory limit given to
// the JVM heap. itzg/minecraft-server accepts MEMORY=75% and turns it into
// -XX:InitialRAMPercentage and -XX:MaxRAMPercentage, leaving about a quarter
// of the cgroup limit for metaspace, threads, and native allocations.
const ContainerHeapPercent = 75

// ContainerPlan is how a container is limited and how the JVM inside it
// sizes its heap.
type ContainerPlan struct {
	// Limit is the container memory limit (Quadlet MemoryMax, compose
	// mem_limit, Apple container -m).
	Limit string
	// Heap is a fixed -Xms/-Xmx. Empty when Percent is set.
	Heap string
	// Percent is non-zero when the JVM should use RAM percentages of Limit
	// instead of Heap. ContainerHeapPercent is the value the auto path uses.
	Percent int
}

// ExplicitContainerPlan keeps an operator-supplied heap and adds 1 GiB to
// the container limit for JVM overhead outside the heap.
func ExplicitContainerPlan(heap string) (ContainerPlan, error) {
	heap = strings.TrimSpace(heap)
	if !memoryPattern.MatchString(heap) {
		return ContainerPlan{}, fmt.Errorf("invalid memory %q: must be a number followed by M or G (e.g. 2G, 2048M)", heap)
	}
	return ContainerPlan{
		Heap:  heap,
		Limit: MemoryLimitForHeap(heap),
	}, nil
}

// AutoContainerPlan sizes the container limit with HeapFromRAM, using the
// same OS reservation and clamps as the bare-metal installer. The JVM heap
// is ContainerHeapPercent of that limit.
func AutoContainerPlan(total uint64) (ContainerPlan, bool, error) {
	limit, tight, err := HeapFromRAM(total)
	if err != nil {
		return ContainerPlan{}, false, err
	}
	return ContainerPlan{
		Limit:   limit,
		Percent: ContainerHeapPercent,
	}, tight, nil
}

// MemoryLimitForHeap adds 1 GiB of headroom above a fixed heap so metaspace
// and native allocations fit inside the container limit. Unparseable input
// returns 3G, the historical fallback for a 2G heap.
func MemoryLimitForHeap(memory string) string {
	memory = strings.TrimSpace(memory)
	if memory == "" {
		return "3G"
	}

	suffix := memory[len(memory)-1:]
	numStr := memory[:len(memory)-1]

	var value int
	if _, err := fmt.Sscanf(numStr, "%d", &value); err != nil || value <= 0 {
		return "3G"
	}

	switch strings.ToUpper(suffix) {
	case "G":
		return fmt.Sprintf("%dG", value+1)
	case "M":
		return fmt.Sprintf("%dM", value+1024)
	default:
		return "3G"
	}
}
