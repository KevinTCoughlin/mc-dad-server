package license

import "time"

// Status represents the state of a license.
type Status string

// License statuses returned by LemonSqueezy.
const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusExpired  Status = "expired"
	StatusDisabled Status = "disabled"
)

// ValidationResponse represents the response from LemonSqueezy license validation.
type ValidationResponse struct {
	Valid      bool       `json:"valid"`
	Error      string     `json:"error,omitempty"`
	LicenseKey Key `json:"license_key"`
	Instance   Instance   `json:"instance,omitempty"`
	Meta       Meta       `json:"meta"`
}

// Key contains details about the license.
type Key struct {
	ID              int        `json:"id"`
	Status          Status     `json:"status"`
	Key             string     `json:"key"`
	ActivationLimit int        `json:"activation_limit"`
	ActivationUsage int        `json:"activation_usage"`
	CreatedAt       time.Time  `json:"created_at"`
	ExpiresAt       *time.Time `json:"expires_at"`
}

// Instance represents an activated instance of a license.
type Instance struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Meta contains metadata about the validation request.
type Meta struct {
	StoreID       int    `json:"store_id"`
	OrderID       int    `json:"order_id"`
	ProductID     int    `json:"product_id"`
	ProductName   string `json:"product_name"`
	VariantID     int    `json:"variant_id"`
	VariantName   string `json:"variant_name"`
	CustomerID    int    `json:"customer_id"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
}

// ActivationResponse represents the response from license activation.
type ActivationResponse struct {
	Activated bool     `json:"activated"`
	Error     string   `json:"error,omitempty"`
	Instance  Instance `json:"instance,omitempty"`
	Meta      Meta     `json:"meta"`
}

// DeactivationResponse represents the response from license deactivation.
type DeactivationResponse struct {
	Deactivated bool   `json:"deactivated"`
	Error       string `json:"error,omitempty"`
}

// IsValid checks if the license validation response indicates a valid, active license.
func (v *ValidationResponse) IsValid() bool {
	return v.Valid && v.LicenseKey.Status == StatusActive
}

// IsActivationLimitReached checks if the license has reached its activation limit.
func (v *ValidationResponse) IsActivationLimitReached() bool {
	if v.LicenseKey.ActivationLimit == 0 {
		return false // Unlimited activations
	}
	return v.LicenseKey.ActivationUsage >= v.LicenseKey.ActivationLimit
}

// IsExpired checks if the license has expired.
func (v *ValidationResponse) IsExpired() bool {
	if v.LicenseKey.ExpiresAt == nil {
		return false // No expiration
	}
	return time.Now().After(*v.LicenseKey.ExpiresAt)
}
