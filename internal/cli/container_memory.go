package cli

import (
	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// resolveContainerMemory turns --memory into a container limit and a JVM
// heap mode. auto sizes the limit from host RAM. Any other value is a fixed
// heap, and the container limit is 1 GiB higher.
func resolveContainerMemory(memory string, output *ui.UI) (config.ContainerPlan, error) {
	if !config.IsAutoMemory(memory) {
		return config.ExplicitContainerPlan(memory)
	}
	total, err := platform.TotalRAM()
	return resolveContainerMemoryFrom(total, err, output)
}

func resolveContainerMemoryFrom(total uint64, detectErr error, output *ui.UI) (config.ContainerPlan, error) {
	if detectErr != nil {
		plan, err := config.ExplicitContainerPlan(config.FallbackHeap)
		if err != nil {
			return config.ContainerPlan{}, err
		}
		output.Warn("Could not detect system RAM (%v). Using a %s heap and a %s container limit. Pass --memory to choose a size.", detectErr, plan.Heap, plan.Limit)
		return plan, nil
	}
	plan, tight, err := config.AutoContainerPlan(total)
	if err != nil {
		fallback, ferr := config.ExplicitContainerPlan(config.FallbackHeap)
		if ferr != nil {
			return config.ContainerPlan{}, ferr
		}
		output.Warn("Could not size the container memory limit from %s of RAM (%v). Using a %s heap and a %s container limit. Pass --memory to choose a size.", platform.FormatBytes(total), err, fallback.Heap, fallback.Limit)
		return fallback, nil
	}
	output.Info("Container memory limit set to %s from %s of RAM. The JVM heap is %d%% of that limit. Pass --memory to set a fixed heap.", plan.Limit, platform.FormatBytes(total), plan.Percent)
	if tight {
		output.Warn("This machine has under 2 GiB of RAM. The container limit is the smallest Paper 26.3 budget, and the JVM will use %d%% of it, so the OS and the heap will both be tight.", plan.Percent)
	}
	return plan, nil
}

// applyContainerPlan copies plan onto cfg. When the heap is a percentage,
// Memory is set to the container limit so Validate still sees a sized value.
// Deploy renders the percentage into the container's MEMORY variable.
func applyContainerPlan(cfg *config.ServerConfig, plan config.ContainerPlan) {
	cfg.ContainerLimit = plan.Limit
	cfg.HeapPercent = plan.Percent
	if plan.Heap != "" {
		cfg.Memory = plan.Heap
		return
	}
	cfg.Memory = plan.Limit
}
