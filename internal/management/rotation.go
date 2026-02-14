package management

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// ParkourMaps is the default list of parkour map world folder names.
var ParkourMaps = []string{
	"parkour-spiral",
	"parkour-spiral-3",
	"parkour-volcano",
	"parkour-pyramid",
	"parkour-paradise",
}

// RotateParkour advances the featured parkour map, broadcasts, and teleports.
func RotateParkour(ctx context.Context, serverDir string, screen *ScreenManager, output *ui.UI) error {
	maps := ParkourMaps
	if len(maps) == 0 {
		output.Info("No parkour maps configured")
		return nil
	}

	stateFile := filepath.Join(serverDir, "rotation-state.txt")

	// Read current index
	currentIndex := 0
	if data, err := os.ReadFile(stateFile); err == nil {
		if idx, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			currentIndex = idx
		}
	}

	// Advance
	nextIndex := (currentIndex + 1) % len(maps)
	if err := os.WriteFile(stateFile, []byte(strconv.Itoa(nextIndex)+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing rotation state: %w", err)
	}

	currentMap := maps[currentIndex]
	nextMap := maps[nextIndex]

	output.Info("[%s] Rotating: %s -> %s",
		time.Now().Format("2006-01-02 15:04:05"), currentMap, nextMap)

	// Broadcast
	if err := screen.SendCommand(ctx, fmt.Sprintf(
		"say [PARKOUR] Featured map: %s! Type /mv tp %s to play!", nextMap, nextMap)); err != nil {
		return err
	}
	_ = Sleep(ctx, 1)

	// Teleport players
	if err := screen.SendCommand(ctx, fmt.Sprintf("mv tp * %s", nextMap)); err != nil {
		return err
	}

	output.Success("Rotation complete: %s", nextMap)
	return nil
}
