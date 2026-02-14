package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cfg    *config.ServerConfig
	runner platform.CommandRunner
	output *ui.UI
)

// NewRootCmd creates the root cobra command with all subcommands.
func NewRootCmd(version, commit string) *cobra.Command {
	cfg = config.DefaultConfig()
	runner = platform.NewOSCommandRunner()
	output = ui.Default()

	root := &cobra.Command{
		Use:   "mc-dad-server",
		Short: "Minecraft server for busy dads",
		Long: `MC Dad Server — Minecraft server in 60 seconds.

No Docker. No Kubernetes. No nonsense. Just one command
and you're hosting Minecraft — with Bedrock cross-play,
Parkour courses, and tuned configs out of the box.`,
		Version: fmt.Sprintf("%s (%s)", version, commit),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cfg.Dir == "" {
				home, _ := os.UserHomeDir()
				cfg.Dir = filepath.Join(home, "minecraft-server")
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	pf := root.PersistentFlags()
	pf.StringVar(&cfg.Dir, "dir", "", "Server directory (default: ~/minecraft-server)")
	pf.StringVar(&cfg.SessionName, "session", "minecraft", "Screen session name")

	// Register subcommands
	root.AddCommand(
		newInstallCmd(),
		newStartCmd(),
		newStopCmd(),
		newStatusCmd(),
		newBackupCmd(),
		newSetupParkourCmd(),
		newRotateParkourCmd(),
	)

	return root
}
