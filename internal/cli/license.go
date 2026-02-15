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
		fmt.Println()
		fmt.Printf("  Product:         %s\n", resp.Meta.ProductName)
		fmt.Printf("  Variant:         %s\n", resp.Meta.VariantName)
		fmt.Printf("  Status:          %s\n", resp.LicenseKey.Status)
		fmt.Printf("  Activations:     %d / %d\n", resp.LicenseKey.ActivationUsage, resp.LicenseKey.ActivationLimit)
		if resp.LicenseKey.ExpiresAt != nil {
			fmt.Printf("  Expires:         %s\n", resp.LicenseKey.ExpiresAt.Format("2006-01-02"))
		} else {
			fmt.Println("  Expires:         Never")
		}
		if resp.Instance.ID != "" {
			fmt.Printf("  Instance:        %s\n", resp.Instance.Name)
		}
	} else {
		output.Warn("License is not valid")
		if resp.Error != "" {
			fmt.Printf("  Error: %s\n", resp.Error)
		}
		fmt.Printf("  Status: %s\n", resp.LicenseKey.Status)
		if resp.IsExpired() {
			fmt.Println("  Reason: License has expired")
		} else if resp.IsActivationLimitReached() {
			fmt.Println("  Reason: Activation limit reached")
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
	fmt.Println()
	fmt.Printf("  Product:      %s\n", resp.Meta.ProductName)
	fmt.Printf("  Instance:     %s\n", resp.Instance.Name)
	fmt.Printf("  Instance ID:  %s\n", resp.Instance.ID)
	fmt.Println()
	fmt.Println("Nag messages will no longer appear.")

	return nil
}

// DeactivateLicenseCmd deactivates the license for this server.
type DeactivateLicenseCmd struct{}

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
	fmt.Println()
	fmt.Println("The license has been freed and can be used on another server.")

	return nil
}
