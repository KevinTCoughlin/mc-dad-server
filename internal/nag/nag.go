package nag

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/license"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// Status represents the license/nag state.
type Status int

const (
	StatusLicensed    Status = iota
	StatusGracePeriod Status = iota
	StatusUnlicensed  Status = iota
)

const graceDays = 7

// Info holds resolved license state.
type Info struct {
	Status       Status
	CustomerName string
	DaysLeft     int
}

type installRecord struct {
	InstalledAt time.Time `json:"installed_at"`
}

// RecordInstall writes a .mc-dad-installed file on first install. Idempotent.
func RecordInstall(serverDir string) {
	path := filepath.Join(serverDir, ".mc-dad-installed")
	if _, err := os.Stat(path); err == nil {
		return // already exists
	}
	rec := installRecord{InstalledAt: time.Now()}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

// Resolve determines the license state for a server directory.
func Resolve(ctx context.Context, serverDir string) Info {
	mgr := license.NewManager(serverDir)
	stored, err := mgr.Load()
	if err == nil && stored != nil {
		resp, err := mgr.Validate(ctx, stored.LicenseKey)
		if err == nil && resp.IsValid() {
			return Info{
				Status:       StatusLicensed,
				CustomerName: resp.Meta.CustomerName,
			}
		}
	}

	// No valid license — check grace period
	path := filepath.Join(serverDir, ".mc-dad-installed")
	data, err := os.ReadFile(path)
	if err == nil {
		var rec installRecord
		if json.Unmarshal(data, &rec) == nil {
			elapsed := time.Since(rec.InstalledAt)
			daysLeft := graceDays - int(math.Ceil(elapsed.Hours()/24))
			if daysLeft > 0 {
				return Info{
					Status:   StatusGracePeriod,
					DaysLeft: daysLeft,
				}
			}
		}
	}

	return Info{Status: StatusUnlicensed}
}

// StatusLabel returns a human-readable label for the license state.
func StatusLabel(info Info) string {
	switch info.Status {
	case StatusLicensed:
		return fmt.Sprintf("Licensed to %s", info.CustomerName)
	case StatusGracePeriod:
		return fmt.Sprintf("Grace Period (%d days remaining)", info.DaysLeft)
	default:
		return "Unlicensed (free)"
	}
}

// MaybeNag prints a nag message for unlicensed users.
func MaybeNag(output *ui.UI, info Info) {
	if info.Status != StatusUnlicensed {
		return
	}
	output.Info("MC Dad Server is free and fully functional!")
	output.Info("Support development at: https://cascadiacollections.lemonsqueezy.com/checkout/buy/6badc98c-1292-4e97-af7a-0dd17546418f")
	output.Info("Use 'mc-dad-server activate-license --key YOUR-KEY' to remove this message.")
}
