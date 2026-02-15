package cli

import (
	"fmt"
	"os"

	"github.com/KevinTCoughlin/mc-dad-server/internal/license"
	"github.com/spf13/cobra"
)

func newValidateLicenseCmd() *cobra.Command {
	var licenseKey string

	cmd := &cobra.Command{
		Use:   "validate-license",
		Short: "Validate your license key",
		Long: `Validate a license key with LemonSqueezy to check if it's active.
If a license key is already stored, it will be validated. Otherwise,
provide a license key with --key flag.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			mgr := license.NewManager(cfg.Dir)

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
		},
	}

	cmd.Flags().StringVar(&licenseKey, "key", "", "License key to validate")
	return cmd
}

func newActivateLicenseCmd() *cobra.Command {
	var licenseKey string
	var instanceName string

	cmd := &cobra.Command{
		Use:   "activate-license",
		Short: "Activate a license key for this server",
		Long: `Activate a license key with LemonSqueezy for this server instance.
This will consume one activation from your license.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			if licenseKey == "" {
				return fmt.Errorf("license key is required (use --key)")
			}

			if instanceName == "" {
				hostname, err := os.Hostname()
				if err != nil {
					instanceName = "mc-dad-server"
				} else {
					instanceName = hostname
				}
			}

			mgr := license.NewManager(cfg.Dir)
			output.Info("Activating license for instance: %s", instanceName)

			resp, err := mgr.Activate(ctx, licenseKey, instanceName)
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
		},
	}

	cmd.Flags().StringVar(&licenseKey, "key", "", "License key to activate (required)")
	cmd.Flags().StringVar(&instanceName, "name", "", "Instance name (default: hostname)")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}

func newDeactivateLicenseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "deactivate-license",
		Short: "Deactivate the license for this server",
		Long: `Deactivate the license for this server instance.
This will free up one activation so you can use it on another server.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			mgr := license.NewManager(cfg.Dir)
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
		},
	}
}
