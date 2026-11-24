package config

import "strings"

// G1Flags is the G1 collector configuration emitted for Paper 26.3 on Java 25.
// The values are Aikar's flags, which JDK 25 still accepts, plus
// ExitOnOutOfMemoryError so a heap exhaustion stops the process instead of
// leaving a wedged server that systemd will not restart.
func G1Flags() []string {
	return []string{
		"-XX:+UseG1GC",
		"-XX:+ParallelRefProcEnabled",
		"-XX:MaxGCPauseMillis=200",
		"-XX:+UnlockExperimentalVMOptions",
		"-XX:+DisableExplicitGC",
		"-XX:+AlwaysPreTouch",
		"-XX:G1NewSizePercent=30",
		"-XX:G1MaxNewSizePercent=40",
		"-XX:G1HeapRegionSize=8M",
		"-XX:G1ReservePercent=20",
		"-XX:G1HeapWastePercent=5",
		"-XX:G1MixedGCCountTarget=4",
		"-XX:InitiatingHeapOccupancyPercent=15",
		"-XX:G1MixedGCLiveThresholdPercent=90",
		"-XX:G1RSetUpdatingPauseTimePercent=5",
		"-XX:SurvivorRatio=32",
		"-XX:+PerfDisableSharedMem",
		"-XX:MaxTenuringThreshold=1",
		"-XX:+ExitOnOutOfMemoryError",
	}
}

// ZGCFlags is the ZGC configuration for Java 25. Generational mode is the
// only ZGC mode from JDK 24 on, and the ZGenerational flag was removed, so
// it is not included.
func ZGCFlags() []string {
	return []string{
		"-XX:+UseZGC",
		"-XX:+AlwaysPreTouch",
		"-XX:+DisableExplicitGC",
		"-XX:+PerfDisableSharedMem",
		"-XX:+ExitOnOutOfMemoryError",
	}
}

// GCFlags returns the collector flags for gc, which is "g1gc" or "zgc".
// Anything else is treated as G1, matching the start script's else branch.
func GCFlags(gc string) []string {
	if strings.EqualFold(gc, "zgc") {
		return ZGCFlags()
	}
	return G1Flags()
}
