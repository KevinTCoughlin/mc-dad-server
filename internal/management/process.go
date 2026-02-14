package management

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// ProcessStats holds resource usage info for the server process.
type ProcessStats struct {
	PID    int
	Memory string
	CPU    string
}

// GetProcessStats finds the server.jar process and returns its stats.
func GetProcessStats(ctx context.Context, runner platform.CommandRunner) (ProcessStats, error) {
	out, err := runner.RunWithOutput(ctx, "pgrep", "-f", "server.jar")
	if err != nil {
		return ProcessStats{}, fmt.Errorf("server not running")
	}

	pidStr := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return ProcessStats{}, fmt.Errorf("invalid PID: %s", pidStr)
	}

	stats := ProcessStats{PID: pid}

	// Get memory (RSS in KB)
	memOut, err := runner.RunWithOutput(ctx, "ps", "-o", "rss=", "-p", pidStr)
	if err == nil {
		rssKB, err := strconv.Atoi(strings.TrimSpace(string(memOut)))
		if err == nil {
			stats.Memory = fmt.Sprintf("%d MB", rssKB/1024)
		}
	}

	// Get CPU %
	cpuOut, err := runner.RunWithOutput(ctx, "ps", "-o", "%cpu=", "-p", pidStr)
	if err == nil {
		stats.CPU = strings.TrimSpace(string(cpuOut)) + "%"
	}

	return stats, nil
}
