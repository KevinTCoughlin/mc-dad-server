package license

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// cacheKey is used to HMAC the cached validation response.
// Not a true secret — anyone can read the source — but it prevents
// casual edits to the .license JSON from being accepted.
var cacheKey = []byte("mc-dad-server-v2-cache-signing-key")

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
	CacheHMAC      string              `json:"cache_hmac,omitempty"`
}

// signCache computes an HMAC-SHA256 over the cached response JSON.
func signCache(resp *ValidationResponse) string {
	if resp == nil {
		return ""
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return ""
	}
	mac := hmac.New(sha256.New, cacheKey)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// verifyCacheHMAC returns true if the stored HMAC matches the cached response.
func verifyCacheHMAC(stored *StoredLicense) bool {
	if stored == nil || stored.CachedResponse == nil || stored.CacheHMAC == "" {
		return false
	}
	return stored.CacheHMAC == signCache(stored.CachedResponse)
}

// Validate validates a license key, using cache if available and recent.
func (m *Manager) Validate(ctx context.Context, licenseKey string) (*ValidationResponse, error) {
	// Try to load stored license
	stored, _ := m.Load()

	var instanceID string
	if stored != nil && stored.LicenseKey == licenseKey {
		instanceID = stored.InstanceID

		// Use cached response if it's recent (within 24 hours) and HMAC is valid
		if stored.CachedResponse != nil && time.Since(stored.LastValidated) < 24*time.Hour {
			if verifyCacheHMAC(stored) && stored.CachedResponse.IsValid() {
				return stored.CachedResponse, nil
			}
		}
	}

	// Validate with LemonSqueezy API
	resp, err := m.client.Validate(ctx, licenseKey, instanceID)
	if err != nil {
		// If offline and we have a valid cached response, use it
		if stored != nil && stored.CachedResponse != nil && stored.LicenseKey == licenseKey && verifyCacheHMAC(stored) {
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
	stored.CacheHMAC = signCache(resp)
	if resp.Instance.ID != "" {
		stored.InstanceID = resp.Instance.ID
		stored.InstanceName = resp.Instance.Name
	}

	// Save the updated license (log error but don't fail validation)
	if err := m.Save(stored); err != nil {
		// Validation succeeded, but we couldn't save the cache.
		// This is a warning, not a failure - the license is still valid.
		fmt.Fprintf(os.Stderr, "Warning: Failed to save license cache: %v\n", err)
	}

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
	if err := m.Save(stored); err != nil {
		// Activation succeeded, but we couldn't save it locally
		fmt.Fprintf(os.Stderr, "Warning: Failed to save license: %v\n", err)
		fmt.Fprintf(os.Stderr, "Your license is activated but not saved locally. Use validate-license to re-sync.\n")
	}

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
	if err := os.Remove(m.licenseFile); err != nil && !os.IsNotExist(err) {
		// Deactivation succeeded but we couldn't remove the local file
		fmt.Fprintf(os.Stderr, "Warning: Failed to remove license file: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please manually remove: %s\n", m.licenseFile)
	}

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
