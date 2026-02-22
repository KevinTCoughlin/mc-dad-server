package cli

import (
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
)

// Globals holds flags shared by all subcommands.
type Globals struct {
	Dir     string           `help:"Server directory (default: ~/minecraft-server)" default:""`
	Session string           `help:"Screen session name" default:"minecraft"`
	Version kong.VersionFlag `help:"Print version" short:"v" hidden:""`
}

// AfterApply sets Dir to ~/minecraft-server when the user hasn't provided one.
func (g *Globals) AfterApply() error {
	if g.Dir == "" {
		home, _ := os.UserHomeDir()
		g.Dir = filepath.Join(home, "minecraft-server")
	}
	return nil
}

// CLI is the top-level command tree parsed by Kong.
type CLI struct {
	Globals

	Install           InstallCmd           `cmd:"" help:"Install and configure a Minecraft server"`
	Start             StartCmd             `cmd:"" help:"Start the Minecraft server in a screen session"`
	Stop              StopCmd              `cmd:"" help:"Gracefully stop the Minecraft server"`
	Status            StatusCmd            `cmd:"" help:"Show server status and resource usage"`
	Backup            BackupCmd            `cmd:"" help:"Backup world data with rotation"`
	Console           ConsoleCmd           `cmd:"" help:"Interactive console with live server log"`
	SetupParkour      SetupParkourCmd      `cmd:"setup-parkour" help:"Set up parkour world (first-time setup)"`
	RotateParkour     RotateParkourCmd     `cmd:"rotate-parkour" help:"Rotate the featured parkour map"`
	VoteMap           VoteMapCmd           `cmd:"vote-map" help:"Start a map vote (CS:GO style)"`
	ValidateLicense   ValidateLicenseCmd   `cmd:"validate-license" help:"Validate your license key"`
	ActivateLicense   ActivateLicenseCmd   `cmd:"activate-license" help:"Activate a license key for this server"`
	DeactivateLicense DeactivateLicenseCmd `cmd:"deactivate-license" help:"Deactivate the license for this server"`
}
