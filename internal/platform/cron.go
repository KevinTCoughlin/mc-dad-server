package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// SetupCronBackup adds a daily 4 AM backup cron job.
func SetupCronBackup(ctx context.Context, runner CommandRunner, serverDir string) error {
	output := ui.Default()
	output.Step("Setting Up Automated Backups")

	// Ensure logs directory exists
	logsDir := filepath.Join(serverDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return fmt.Errorf("creating logs dir: %w", err)
	}

	cronLine := fmt.Sprintf("0 4 * * * /usr/local/bin/mc-dad-server backup --dir %s >> %s/backup.log 2>&1",
		serverDir, logsDir)

	// Check if already exists
	existing, err := runner.RunWithOutput(ctx, "crontab", "-l")
	if err == nil && strings.Contains(string(existing), "mc-dad-server backup") {
		output.Warn("Backup cron job already exists")
		return nil
	}

	// Add cron job
	var crontab string
	if err == nil {
		crontab = string(existing)
	}
	crontab += "\n# mc-dad-server daily backup\n" + cronLine + "\n"

	// Write via stdin
	tmpFile := filepath.Join(os.TempDir(), "mc-dad-server-crontab")
	if err := os.WriteFile(tmpFile, []byte(crontab), 0o600); err != nil {
		return fmt.Errorf("writing temp crontab: %w", err)
	}
	defer os.Remove(tmpFile)

	if err := runner.Run(ctx, "crontab", tmpFile); err != nil {
		return fmt.Errorf("installing crontab: %w", err)
	}

	output.Success("Daily backup scheduled at 4:00 AM")
	return nil
}
