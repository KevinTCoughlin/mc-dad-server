package management

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// ProcessStats holds resource usage info for the server process.
type ProcessStats struct {
	PID    int
	Memory string
	CPU    string
}

// serverJarPatterns are the jar names used by supported Minecraft server types.
var serverJarPatterns = []string{"server.jar", "paper.jar", "fabric-server-launch.jar"}

// GetProcessStats finds the Minecraft server process and returns its stats.
func GetProcessStats(ctx context.Context, runner platform.CommandRunner) (ProcessStats, error) {
	var out []byte
	var err error
	for _, pattern := range serverJarPatterns {
		out, err = runner.RunWithOutput(ctx, "pgrep", "-f", pattern)
		if err == nil {
			break
		}
	}
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

// IsServerRunning checks whether a Minecraft server is running using the
// manager's own detection, process detection, and port probing.
func IsServerRunning(ctx context.Context, mgr ServerManager, runner platform.CommandRunner, port int) bool {
	if mgr.IsRunning(ctx) {
		return true
	}
	if stats, err := GetProcessStats(ctx, runner); err == nil && stats.PID > 0 {
		return true
	}
	return IsPortListening(port)
}

// IsPortListening checks if something is listening on the given TCP port.
func IsPortListening(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 1*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
