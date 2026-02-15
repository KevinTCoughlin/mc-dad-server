package dadpack

import (
	"context"
	"fmt"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/license"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// Features represents the available Dad Pack features.
type Features struct {
	GriefPrevention bool
	Dynmap          bool
	WebDashboard    bool
	DadGuide        bool
}

// Manager handles Dad Pack feature installation and management.
type Manager struct {
	licenseManager *license.Manager
	output         *ui.UI
}

// NewManager creates a new Dad Pack manager.
func NewManager(serverDir string, output *ui.UI) *Manager {
	return &Manager{
		licenseManager: license.NewManager(serverDir),
		output:         output,
	}
}

// CheckLicense validates the license and returns whether Dad Pack features are available.
func (m *Manager) CheckLicense(ctx context.Context, cfg *config.ServerConfig) (bool, error) {
	// If no license key provided, Dad Pack features are not available
	if cfg.LicenseKey == "" {
		return false, nil
	}

	m.output.Info("Validating Dad Pack license...")

	// Validate the license
	resp, err := m.licenseManager.Validate(ctx, cfg.LicenseKey)
	if err != nil {
		m.output.Warn("License validation failed: %v", err)
		return false, err
	}

	if !resp.IsValid() {
		m.output.Warn("License is not valid (status: %s)", resp.LicenseKey.Status)
		return false, fmt.Errorf("invalid license")
	}

	m.output.Success("Dad Pack license validated!")
	return true, nil
}

// InstallFeatures installs the Dad Pack features for a Paper server.
func (m *Manager) InstallFeatures(ctx context.Context, serverDir string) error {
	m.output.Step("Installing Dad Pack Features")

	// Note: These features are placeholders for when Dad Pack is actually implemented
	// For now, we just acknowledge that they would be installed

	m.output.Info("Dad Pack features will be installed in a future update:")
	fmt.Println("  • GriefPrevention - Auto-configured build protection")
	fmt.Println("  • Dynmap - Web-based live map")
	fmt.Println("  • Web Dashboard - Simple status page")
	fmt.Println("  • Dad's Guide PDF - Non-technical admin guide")

	m.output.Success("Dad Pack features prepared (placeholder)")

	// TODO: Implement actual feature installation when features are ready
	// - Download and install GriefPrevention plugin
	// - Download and install Dynmap plugin
	// - Set up web dashboard
	// - Download Dad's Guide PDF

	return nil
}

// GetAvailableFeatures returns which Dad Pack features are available.
func (m *Manager) GetAvailableFeatures() Features {
	// For now, return placeholder status
	// In the future, check which features are actually installed
	return Features{
		GriefPrevention: false, // Not yet implemented
		Dynmap:          false, // Not yet implemented
		WebDashboard:    false, // Not yet implemented
		DadGuide:        false, // Not yet implemented
	}
}
