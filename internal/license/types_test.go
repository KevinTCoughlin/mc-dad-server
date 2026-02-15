package license

import (
	"testing"
	"time"
)

func TestValidationResponse_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		response *ValidationResponse
		want     bool
	}{
		{
			name: "valid active license",
			response: &ValidationResponse{
				Valid: true,
				LicenseKey: Key{
					Status: StatusActive,
				},
			},
			want: true,
		},
		{
			name: "inactive license",
			response: &ValidationResponse{
				Valid: true,
				LicenseKey: Key{
					Status: StatusInactive,
				},
			},
			want: false,
		},
		{
			name: "expired license",
			response: &ValidationResponse{
				Valid: true,
				LicenseKey: Key{
					Status: StatusExpired,
				},
			},
			want: false,
		},
		{
			name: "disabled license",
			response: &ValidationResponse{
				Valid: true,
				LicenseKey: Key{
					Status: StatusDisabled,
				},
			},
			want: false,
		},
		{
			name: "invalid response",
			response: &ValidationResponse{
				Valid: false,
				LicenseKey: Key{
					Status: StatusActive,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.response.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationResponse_IsActivationLimitReached(t *testing.T) {
	tests := []struct {
		name     string
		response *ValidationResponse
		want     bool
	}{
		{
			name: "unlimited activations",
			response: &ValidationResponse{
				LicenseKey: Key{
					ActivationLimit: 0,
					ActivationUsage: 10,
				},
			},
			want: false,
		},
		{
			name: "below limit",
			response: &ValidationResponse{
				LicenseKey: Key{
					ActivationLimit: 5,
					ActivationUsage: 3,
				},
			},
			want: false,
		},
		{
			name: "at limit",
			response: &ValidationResponse{
				LicenseKey: Key{
					ActivationLimit: 5,
					ActivationUsage: 5,
				},
			},
			want: true,
		},
		{
			name: "over limit",
			response: &ValidationResponse{
				LicenseKey: Key{
					ActivationLimit: 5,
					ActivationUsage: 7,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.response.IsActivationLimitReached(); got != tt.want {
				t.Errorf("IsActivationLimitReached() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationResponse_IsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	tests := []struct {
		name     string
		response *ValidationResponse
		want     bool
	}{
		{
			name: "no expiration",
			response: &ValidationResponse{
				LicenseKey: Key{
					ExpiresAt: nil,
				},
			},
			want: false,
		},
		{
			name: "expired",
			response: &ValidationResponse{
				LicenseKey: Key{
					ExpiresAt: &past,
				},
			},
			want: true,
		},
		{
			name: "not expired",
			response: &ValidationResponse{
				LicenseKey: Key{
					ExpiresAt: &future,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.response.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
