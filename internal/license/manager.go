package license

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Manager handles license validation and storage.
type Manager struct {
	client      *Client
	licenseFile string
}

// NewManager creates a new license manager.
func NewManager(serverDir string) *Manager {
	return &Manager{
		client:      NewClient(),
		licenseFile: filepath.Join(serverDir, ".license"),
	}
}

// StoredLicense represents a license stored on disk.
type StoredLicense struct {
	LicenseKey     string              `json:"license_key"`
	InstanceID     string              `json:"instance_id"`
	InstanceName   string              `json:"instance_name"`
	LastValidated  time.Time           `json:"last_validated"`
	CachedResponse *ValidationResponse `json:"cached_response,omitempty"`
}

// Validate validates a license key, using cache if available and recent.
func (m *Manager) Validate(ctx context.Context, licenseKey string) (*ValidationResponse, error) {
	// Try to load stored license
	stored, _ := m.Load()

	var instanceID string
	if stored != nil && stored.LicenseKey == licenseKey {
		instanceID = stored.InstanceID

		// Use cached response if it's recent (within 24 hours)
		if stored.CachedResponse != nil && time.Since(stored.LastValidated) < 24*time.Hour {
			if stored.CachedResponse.IsValid() {
				return stored.CachedResponse, nil
			}
		}
	}

	// Validate with LemonSqueezy API
	resp, err := m.client.Validate(ctx, licenseKey, instanceID)
	if err != nil {
		// If offline and we have a cached response, use it
		if stored != nil && stored.CachedResponse != nil && stored.LicenseKey == licenseKey {
			return stored.CachedResponse, nil
		}
		return nil, fmt.Errorf("validating license: %w", err)
	}

	// Update stored license
	if stored == nil {
		stored = &StoredLicense{}
	}
	stored.LicenseKey = licenseKey
	stored.LastValidated = time.Now()
	stored.CachedResponse = resp
	if resp.Instance.ID != "" {
		stored.InstanceID = resp.Instance.ID
		stored.InstanceName = resp.Instance.Name
	}

	_ = m.Save(stored)

	return resp, nil
}

// Activate activates a license for this instance.
func (m *Manager) Activate(ctx context.Context, licenseKey, instanceName string) (*ActivationResponse, error) {
	resp, err := m.client.Activate(ctx, licenseKey, instanceName)
	if err != nil {
		return nil, fmt.Errorf("activating license: %w", err)
	}

	if !resp.Activated {
		if resp.Error != "" {
			return nil, fmt.Errorf("activation failed: %s", resp.Error)
		}
		return nil, fmt.Errorf("activation failed")
	}

	// Store the activated license
	stored := &StoredLicense{
		LicenseKey:    licenseKey,
		InstanceID:    resp.Instance.ID,
		InstanceName:  resp.Instance.Name,
		LastValidated: time.Now(),
	}
	_ = m.Save(stored)

	return resp, nil
}

// Deactivate deactivates the current license instance.
func (m *Manager) Deactivate(ctx context.Context) error {
	stored, err := m.Load()
	if err != nil {
		return fmt.Errorf("loading license: %w", err)
	}
	if stored == nil {
		return fmt.Errorf("no license found")
	}

	resp, err := m.client.Deactivate(ctx, stored.LicenseKey, stored.InstanceID)
	if err != nil {
		return fmt.Errorf("deactivating license: %w", err)
	}

	if !resp.Deactivated {
		if resp.Error != "" {
			return fmt.Errorf("deactivation failed: %s", resp.Error)
		}
		return fmt.Errorf("deactivation failed")
	}

	// Remove the stored license
	_ = os.Remove(m.licenseFile)

	return nil
}

// Load loads the stored license from disk.
func (m *Manager) Load() (*StoredLicense, error) {
	data, err := os.ReadFile(m.licenseFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading license file: %w", err)
	}

	var stored StoredLicense
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, fmt.Errorf("parsing license file: %w", err)
	}

	return &stored, nil
}

// Save saves the license to disk.
func (m *Manager) Save(stored *StoredLicense) error {
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling license: %w", err)
	}

	if err := os.WriteFile(m.licenseFile, data, 0o600); err != nil {
		return fmt.Errorf("writing license file: %w", err)
	}

	return nil
}

// HasValidLicense checks if there's a valid license stored.
func (m *Manager) HasValidLicense(ctx context.Context) bool {
	stored, err := m.Load()
	if err != nil || stored == nil {
		return false
	}

	resp, err := m.Validate(ctx, stored.LicenseKey)
	if err != nil {
		return false
	}

	return resp.IsValid()
}
