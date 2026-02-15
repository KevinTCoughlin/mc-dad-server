package nag

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestSignInstallRecord_Deterministic(t *testing.T) {
	ts := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	a := signInstallRecord(ts)
	b := signInstallRecord(ts)
	if !bytesEqual(a, b) {
		t.Error("signInstallRecord produced different results for same input")
	}
}

func TestSignInstallRecord_DifferentTimes(t *testing.T) {
	a := signInstallRecord(time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC))
	b := signInstallRecord(time.Date(2025, 1, 16, 12, 0, 0, 0, time.UTC))
	if bytesEqual(a, b) {
		t.Error("signInstallRecord produced same result for different timestamps")
	}
}

func TestSignInstallRecord_Length(t *testing.T) {
	sig := signInstallRecord(time.Now())
	// SHA-256 produces 32 bytes
	if len(sig) != 32 {
		t.Errorf("signInstallRecord returned %d bytes, want 32", len(sig))
	}
}

func TestVerifyInstallRecord(t *testing.T) {
	now := time.Date(2025, 6, 1, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		rec  *installRecord
		want bool
	}{
		{
			name: "valid record",
			rec: &installRecord{
				InstalledAt: now,
				HMAC:        hex.EncodeToString(signInstallRecord(now)),
			},
			want: true,
		},
		{
			name: "empty HMAC",
			rec: &installRecord{
				InstalledAt: now,
				HMAC:        "",
			},
			want: false,
		},
		{
			name: "invalid hex",
			rec: &installRecord{
				InstalledAt: now,
				HMAC:        "not-valid-hex!",
			},
			want: false,
		},
		{
			name: "wrong HMAC",
			rec: &installRecord{
				InstalledAt: now,
				HMAC:        "deadbeefdeadbeefdeadbeefdeadbeef",
			},
			want: false,
		},
		{
			name: "tampered timestamp",
			rec: &installRecord{
				InstalledAt: now.Add(-48 * time.Hour), // moved back 2 days
				HMAC:        hex.EncodeToString(signInstallRecord(now)),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verifyInstallRecord(tt.rec); got != tt.want {
				t.Errorf("verifyInstallRecord() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		name string
		info Info
		want string
	}{
		{
			name: "licensed",
			info: Info{Status: StatusLicensed, CustomerName: "Alice"},
			want: "Licensed to Alice",
		},
		{
			name: "grace period",
			info: Info{Status: StatusGracePeriod, DaysLeft: 3},
			want: "Grace Period (3 days remaining)",
		},
		{
			name: "unlicensed",
			info: Info{Status: StatusUnlicensed},
			want: "Unlicensed (free)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StatusLabel(tt.info); got != tt.want {
				t.Errorf("StatusLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
