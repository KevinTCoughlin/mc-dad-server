package nag

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

// Nag statuses.
const (
	StatusLicensed    Status = iota
	StatusGracePeriod Status = iota
	StatusUnlicensed  Status = iota
)

const graceDays = 7

var installKey = []byte("mc-dad-server-v2-install-record-key")

// Info holds resolved license state.
type Info struct {
	Status       Status
	CustomerName string
	DaysLeft     int
}

type installRecord struct {
	InstalledAt time.Time `json:"installed_at"`
	HMAC        string    `json:"hmac,omitempty"`
}

// signInstallRecord computes an HMAC-SHA256 over the install timestamp.
func signInstallRecord(t time.Time) []byte {
	mac := hmac.New(sha256.New, installKey)
	mac.Write([]byte(t.Format(time.RFC3339Nano)))
	return mac.Sum(nil)
}

// RecordInstall writes a .mc-dad-installed file on first install. Idempotent.
func RecordInstall(serverDir string) {
	path := filepath.Join(serverDir, ".mc-dad-installed")
	if _, err := os.Stat(path); err == nil {
		return // already exists
	}
	now := time.Now()
	rec := installRecord{
		InstalledAt: now,
		HMAC:        hex.EncodeToString(signInstallRecord(now)),
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

// verifyInstallRecord returns true if the record's HMAC matches its timestamp.
func verifyInstallRecord(rec *installRecord) bool {
	if rec.HMAC == "" {
		return false
	}
	storedMAC, err := hex.DecodeString(rec.HMAC)
	if err != nil {
		return false
	}
	return hmac.Equal(storedMAC, signInstallRecord(rec.InstalledAt))
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
		if json.Unmarshal(data, &rec) == nil && verifyInstallRecord(&rec) {
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
