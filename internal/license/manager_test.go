package license

import (
	"encoding/hex"
	"testing"
)

func TestSignCache_NilResponse(t *testing.T) {
	if got := signCache(nil); got != nil {
		t.Errorf("signCache(nil) = %v, want nil", got)
	}
}

func TestSignCache_Deterministic(t *testing.T) {
	resp := &ValidationResponse{
		Valid:      true,
		LicenseKey: Key{Status: StatusActive},
	}
	a := signCache(resp)
	b := signCache(resp)
	if !equal(a, b) {
		t.Errorf("signCache produced different results for same input")
	}
}

func TestSignCache_DifferentInputsDifferentOutput(t *testing.T) {
	a := signCache(&ValidationResponse{Valid: true, LicenseKey: Key{Status: StatusActive}})
	b := signCache(&ValidationResponse{Valid: false, LicenseKey: Key{Status: StatusInactive}})
	if equal(a, b) {
		t.Error("signCache produced same result for different inputs")
	}
}

func TestSignCacheHex_NilResponse(t *testing.T) {
	if got := signCacheHex(nil); got != "" {
		t.Errorf("signCacheHex(nil) = %q, want empty", got)
	}
}

func TestSignCacheHex_ValidHex(t *testing.T) {
	resp := &ValidationResponse{Valid: true, LicenseKey: Key{Status: StatusActive}}
	h := signCacheHex(resp)
	if h == "" {
		t.Fatal("signCacheHex returned empty string")
	}
	if _, err := hex.DecodeString(h); err != nil {
		t.Errorf("signCacheHex returned invalid hex: %v", err)
	}
}

func TestVerifyCacheHMAC(t *testing.T) {
	resp := &ValidationResponse{
		Valid:      true,
		LicenseKey: Key{Status: StatusActive, ActivationLimit: 5, ActivationUsage: 1},
	}

	tests := []struct {
		name   string
		stored *StoredLicense
		want   bool
	}{
		{
			name:   "nil stored",
			stored: nil,
			want:   false,
		},
		{
			name:   "nil cached response",
			stored: &StoredLicense{CacheHMAC: "abc"},
			want:   false,
		},
		{
			name:   "empty HMAC",
			stored: &StoredLicense{CachedResponse: resp, CacheHMAC: ""},
			want:   false,
		},
		{
			name:   "invalid hex in HMAC",
			stored: &StoredLicense{CachedResponse: resp, CacheHMAC: "not-valid-hex!"},
			want:   false,
		},
		{
			name:   "wrong HMAC",
			stored: &StoredLicense{CachedResponse: resp, CacheHMAC: "deadbeef"},
			want:   false,
		},
		{
			name:   "valid HMAC",
			stored: &StoredLicense{CachedResponse: resp, CacheHMAC: signCacheHex(resp)},
			want:   true,
		},
		{
			name: "tampered response",
			stored: &StoredLicense{
				CachedResponse: &ValidationResponse{
					Valid:      true,
					LicenseKey: Key{Status: StatusActive, ActivationLimit: 5, ActivationUsage: 2},
				},
				CacheHMAC: signCacheHex(resp), // HMAC from original, response changed
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verifyCacheHMAC(tt.stored); got != tt.want {
				t.Errorf("verifyCacheHMAC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func equal(a, b []byte) bool {
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
