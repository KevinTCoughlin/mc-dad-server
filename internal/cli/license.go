package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/KevinTCoughlin/mc-dad-server/internal/license"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// ValidateLicenseCmd validates a license key.
type ValidateLicenseCmd struct {
	Key string `help:"License key to validate" default:""`
}

// Run validates a license key.
func (cmd *ValidateLicenseCmd) Run(globals *Globals, _ platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	mgr := license.NewManager(globals.Dir)

	licenseKey := cmd.Key

	// If no key provided, try to load stored license
	if licenseKey == "" {
		stored, err := mgr.Load()
		if err != nil {
			return fmt.Errorf("loading license: %w", err)
		}
		if stored == nil {
			return fmt.Errorf("no license key found. Use --key to provide one")
		}
		licenseKey = stored.LicenseKey
		output.Info("Validating stored license...")
	} else {
		output.Info("Validating license key...")
	}

	resp, err := mgr.Validate(ctx, licenseKey)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if resp.IsValid() {
		output.Success("License is valid!")
		output.Info("")
		output.Info("  Product:         %s", resp.Meta.ProductName)
		output.Info("  Variant:         %s", resp.Meta.VariantName)
		output.Info("  Status:          %s", resp.LicenseKey.Status)
		output.Info("  Activations:     %d / %d", resp.LicenseKey.ActivationUsage, resp.LicenseKey.ActivationLimit)
		if resp.LicenseKey.ExpiresAt != nil {
			output.Info("  Expires:         %s", resp.LicenseKey.ExpiresAt.Format("2006-01-02"))
		} else {
			output.Info("  Expires:         Never")
		}
		if resp.Instance.ID != "" {
			output.Info("  Instance:        %s", resp.Instance.Name)
		}
	} else {
		output.Warn("License is not valid")
		if resp.Error != "" {
			output.Info("  Error: %s", resp.Error)
		}
		output.Info("  Status: %s", resp.LicenseKey.Status)
		if resp.IsExpired() {
			output.Info("  Reason: License has expired")
		} else if resp.IsActivationLimitReached() {
			output.Info("  Reason: Activation limit reached")
		}
		return fmt.Errorf("license validation failed")
	}

	return nil
}

// ActivateLicenseCmd activates a license key for this server.
type ActivateLicenseCmd struct {
	Key  string `help:"License key to activate" required:""`
	Name string `help:"Instance name (default: hostname)" default:""`
}

// Run activates a license key.
func (cmd *ActivateLicenseCmd) Run(globals *Globals, _ platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()

	instanceName := cmd.Name
	if instanceName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			instanceName = "mc-dad-server"
		} else {
			instanceName = hostname
		}
	}

	mgr := license.NewManager(globals.Dir)
	output.Info("Activating license for instance: %s", instanceName)

	resp, err := mgr.Activate(ctx, cmd.Key, instanceName)
	if err != nil {
		return fmt.Errorf("activation failed: %w", err)
	}

	output.Success("License activated successfully!")
	output.Info("")
	output.Info("  Product:      %s", resp.Meta.ProductName)
	output.Info("  Instance:     %s", resp.Instance.Name)
	output.Info("  Instance ID:  %s", resp.Instance.ID)
	output.Info("")
	output.Info("Nag messages will no longer appear.")

	return nil
}

// DeactivateLicenseCmd deactivates the license for this server.
type DeactivateLicenseCmd struct{}

// Run deactivates the license.
func (cmd *DeactivateLicenseCmd) Run(globals *Globals, _ platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()

	mgr := license.NewManager(globals.Dir)
	stored, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("loading license: %w", err)
	}
	if stored == nil {
		return fmt.Errorf("no active license found")
	}

	output.Info("Deactivating license for instance: %s", stored.InstanceName)

	if err := mgr.Deactivate(ctx); err != nil {
		return fmt.Errorf("deactivation failed: %w", err)
	}

	output.Success("License deactivated successfully!")
	output.Info("")
	output.Info("The license has been freed and can be used on another server.")

	return nil
}
