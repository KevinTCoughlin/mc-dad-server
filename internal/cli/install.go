package cli

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/configs"
	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/nag"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/plugins"
	"github.com/KevinTCoughlin/mc-dad-server/internal/server"
	"github.com/KevinTCoughlin/mc-dad-server/internal/tunnel"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
	"github.com/KevinTCoughlin/mc-dad-server/internal/vote"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and configure a Minecraft server",
		Long: `Full server setup: installs Java, downloads the server JAR,
deploys configs, installs plugins, sets up services and backups.`,
		RunE: runInstall,
	}

	f := cmd.Flags()
	f.StringVar(&cfg.Edition, "edition", cfg.Edition, "Server edition (java or bedrock)")
	f.IntVar(&cfg.Port, "port", cfg.Port, "Server port")
	f.StringVar(&cfg.Memory, "memory", cfg.Memory, "RAM allocation (e.g., 2G, 4G)")
	f.StringVar(&cfg.ServerType, "type", cfg.ServerType, "Server type (paper, fabric, vanilla)")
	f.StringVar(&cfg.MOTD, "motd", cfg.MOTD, "Message of the day")
	f.IntVar(&cfg.MaxPlayers, "players", cfg.MaxPlayers, "Max players")
	f.StringVar(&cfg.Difficulty, "difficulty", cfg.Difficulty, "Difficulty (peaceful, easy, normal, hard)")
	f.StringVar(&cfg.GameMode, "gamemode", cfg.GameMode, "Game mode (survival, creative, adventure)")
	f.StringVar(&cfg.GCType, "gc", cfg.GCType, "Garbage collector (g1gc or zgc)")
	f.BoolVar(&cfg.Whitelist, "whitelist", cfg.Whitelist, "Enable whitelist")
	f.BoolVar(&cfg.ChatFilter, "chat-filter", cfg.ChatFilter, "Install chat filter plugin")
	f.BoolVar(&cfg.EnablePlayit, "playit", cfg.EnablePlayit, "Set up playit.gg tunnel")
	f.StringVar(&cfg.Version, "version", cfg.Version, "Minecraft version (default: latest)")

	return cmd
}

func runInstall(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	if err := cfg.Validate(); err != nil {
		return err
	}

	printBanner()

	// Detect platform
	output.Step("Detecting Platform")
	plat := platform.Detect(ctx, runner)
	output.Success("OS: %s | Distro: %s | Package Manager: %s | Init: %s",
		plat.OS, plat.Distro, plat.PkgMgr, plat.InitSystem)

	// Install dependencies
	if err := installDependencies(ctx, &plat); err != nil {
		return err
	}

	// Download server
	if err := downloadServer(ctx); err != nil {
		return err
	}

	// Accept EULA
	if err := server.AcceptEULA(cfg.Dir); err != nil {
		return fmt.Errorf("accepting EULA: %w", err)
	}
	output.Success("EULA accepted")

	// Generate RCON password and deploy configs
	cfg.RCONPassword = generateRCONPassword()
	if err := configs.Deploy(cfg); err != nil {
		return fmt.Errorf("deploying configs: %w", err)
	}
	output.Success("Configs deployed with tuned PaperMC defaults")
	output.Info("RCON password saved to server.properties (port 25575)")

	// Chat filter
	if cfg.ChatFilter && cfg.ServerType == "paper" {
		if err := setupChatFilter(ctx); err != nil {
			output.Warn("Chat filter setup failed: %v", err)
		}
	}

	// Plugins
	if cfg.ServerType == "paper" {
		if err := installPlugins(ctx); err != nil {
			output.Warn("Some plugins failed to install: %v", err)
		}
	}

	// Record install timestamp for grace period tracking
	nag.RecordInstall(cfg.Dir)

	// Create start script
	if err := configs.DeployStartScript(cfg); err != nil {
		return fmt.Errorf("creating start script: %w", err)
	}
	output.Success("Start script created")

	// Cron backup
	if err := platform.SetupCronBackup(ctx, runner, cfg.Dir); err != nil {
		output.Warn("Cron backup setup failed: %v", err)
	}

	// Firewall
	platform.ConfigureFirewall(ctx, runner, &plat, cfg.Port, cfg.ServerType)

	// Service
	if err := setupService(ctx, &plat); err != nil {
		output.Warn("Service setup failed: %v", err)
	}

	// Playit
	if cfg.EnablePlayit {
		if err := tunnel.InstallPlayit(ctx, runner, &plat, output); err != nil {
			output.Warn("playit.gg setup failed: %v", err)
		}
		output.Info("To set up your tunnel:")
		fmt.Println("  1. Run: playit")
		fmt.Println("  2. Follow the link to claim your agent")
		fmt.Printf("  3. Create a Minecraft tunnel pointing to localhost:%d\n", cfg.Port)
		fmt.Println("  4. Share the playit.gg address with your kids!")
	}

	// Resolve license state
	nagInfo := nag.Resolve(ctx, cfg.Dir)

	// Summary
	output.PrintInstallSummary(&ui.InstallSummary{
		ServerDir:    cfg.Dir,
		ServerType:   cfg.ServerType,
		Port:         cfg.Port,
		Memory:       cfg.Memory,
		GCType:       cfg.GCType,
		Whitelist:    cfg.Whitelist,
		Difficulty:   cfg.Difficulty,
		GameMode:     cfg.GameMode,
		ChatFilter:   cfg.ChatFilter,
		PlayitSetup:  cfg.EnablePlayit,
		LicenseLabel: nag.StatusLabel(nagInfo),
		InitSystem:   plat.InitSystem,
	})

	nag.MaybeNag(output, nagInfo)

	return nil
}

func printBanner() {
	fmt.Println()
	fmt.Println("  ╔═══════════════════════════════════════╗")
	fmt.Println("  ║     MC Dad Server Installer v2.0      ║")
	fmt.Println("  ║   Minecraft for Busy Dads, Made Easy  ║")
	fmt.Println("  ╚═══════════════════════════════════════╝")
	fmt.Println()
}

func installDependencies(ctx context.Context, plat *platform.Platform) error {
	output.Step("Installing Dependencies")

	deps := []string{"curl", "jq", "screen"}
	for _, dep := range deps {
		if err := platform.InstallPackage(ctx, runner, plat, dep, output); err != nil {
			return fmt.Errorf("installing %s: %w", dep, err)
		}
	}

	if cfg.Edition == "java" {
		if err := platform.InstallJava(ctx, runner, plat, output); err != nil {
			return fmt.Errorf("installing Java: %w", err)
		}
	}

	return nil
}

func downloadServer(ctx context.Context) error {
	output.Step("Downloading Minecraft Server")

	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return fmt.Errorf("creating server directory: %w", err)
	}

	return server.Download(ctx, cfg.ServerType, cfg.Version, cfg.Dir, runner, output)
}

func setupChatFilter(ctx context.Context) error {
	output.Step("Setting Up Chat Filter")
	return plugins.SetupChatFilter(cfg.Dir, output)
}

func installPlugins(ctx context.Context) error {
	output.Step("Installing Plugins")
	return plugins.InstallAll(ctx, cfg.Dir, output)
}

func setupService(ctx context.Context, plat *platform.Platform) error {
	svc := platform.NewServiceManager(plat, runner, cfg)
	if svc == nil {
		return nil
	}
	if err := svc.Install(cfg); err != nil {
		return err
	}
	return svc.Enable()
}

func generateRCONPassword() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 24)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			b[i] = 'x'
			continue
		}
		b[i] = chars[n.Int64()]
	}
	return string(b)
}

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the Minecraft server in a screen session",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			screen := management.NewScreenManager(runner, cfg.SessionName)

			if screen.IsRunning(ctx) {
				output.Warn("Server is already running! Use: screen -r %s", cfg.SessionName)
				return nil
			}

			output.Info("Starting Minecraft server in screen session '%s'...", cfg.SessionName)
			if err := screen.Start(ctx, "bash", cfg.Dir+"/start.sh"); err != nil {
				return fmt.Errorf("starting server: %w", err)
			}
			output.Success("Server started!")
			fmt.Println()
			fmt.Printf("  Attach to console:  screen -r %s\n", cfg.SessionName)
			fmt.Println("  Detach from console: Ctrl+A then D")
			fmt.Println("  Stop server:         mc-dad-server stop")
			fmt.Println("  Server status:       mc-dad-server status")
			fmt.Println()
			nagInfo := nag.Resolve(ctx, cfg.Dir)
			nag.MaybeNag(output, nagInfo)
			return nil
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Gracefully stop the Minecraft server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			screen := management.NewScreenManager(runner, cfg.SessionName)

			if !screen.IsRunning(ctx) {
				output.Info("No running Minecraft server found.")
				return nil
			}

			output.Info("Sending shutdown command...")
			if err := screen.SendCommand(ctx, "say Server shutting down in 10 seconds..."); err != nil {
				return err
			}
			if err := management.Sleep(ctx, 10); err != nil {
				return err
			}
			if err := screen.SendCommand(ctx, "stop"); err != nil {
				return err
			}
			output.Success("Stop command sent. Server shutting down...")
			fmt.Println()
			nagInfo := nag.Resolve(ctx, cfg.Dir)
			nag.MaybeNag(output, nagInfo)
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show server status and resource usage",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			screen := management.NewScreenManager(runner, cfg.SessionName)

			fmt.Println("═══ Minecraft Server Status ═══")
			fmt.Println()

			if screen.IsRunning(ctx) {
				fmt.Printf("  Status:  RUNNING\n")
				fmt.Printf("  Session: screen -r %s\n", cfg.SessionName)
			} else {
				fmt.Println("  Status:  STOPPED")
			}
			fmt.Println()

			stats, err := management.GetProcessStats(ctx, runner)
			if err == nil && stats.PID > 0 {
				fmt.Printf("  PID:     %d\n", stats.PID)
				fmt.Printf("  Memory:  %s\n", stats.Memory)
				fmt.Printf("  CPU:     %s\n", stats.CPU)
			}
			fmt.Println()

			nagInfo := nag.Resolve(ctx, cfg.Dir)
			fmt.Printf("  License: %s\n", nag.StatusLabel(nagInfo))
			fmt.Println()
			nag.MaybeNag(output, nagInfo)
			return nil
		},
	}
}

func newBackupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Backup world data with rotation",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			screen := management.NewScreenManager(runner, cfg.SessionName)
			return management.Backup(ctx, cfg.Dir, cfg.MaxBackups, screen, output)
		},
	}
}

func newSetupParkourCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup-parkour",
		Short: "Set up parkour world (first-time setup)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			screen := management.NewScreenManager(runner, cfg.SessionName)

			if !screen.IsRunning(ctx) {
				return fmt.Errorf("server not running — start it first with: mc-dad-server start")
			}

			output.Info("Setting up parkour world...")
			cmds := []string{
				"mv create parkour normal --world-type flat --no-structures",
			}
			for _, c := range cmds {
				if err := screen.SendCommand(ctx, c); err != nil {
					return err
				}
				if err := management.Sleep(ctx, 2); err != nil {
					return err
				}
			}
			if err := management.Sleep(ctx, 3); err != nil {
				return err
			}

			gamerules := []string{
				"mv modify parkour set gamemode adventure",
				"mv modify parkour set difficulty peaceful",
				"mv gamerule set minecraft:spawn_mobs false parkour",
				"mv gamerule set minecraft:advance_weather false parkour",
				"mv gamerule set minecraft:advance_time false parkour",
				"mv gamerule set minecraft:fire_damage false parkour",
				"mv gamerule set minecraft:spawn_monsters false parkour",
				"mv gamerule set minecraft:spawn_phantoms false parkour",
				"mv gamerule set minecraft:mob_griefing false parkour",
			}
			for _, c := range gamerules {
				if err := screen.SendCommand(ctx, c); err != nil {
					return err
				}
				if err := management.Sleep(ctx, 2); err != nil {
					return err
				}
			}

			output.Success("Parkour world created!")
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  1. Join the server and run: /mv tp parkour")
			fmt.Println("  2. Fly to where you want the parkour lobby")
			fmt.Println("  3. Run: /pa setlobby")
			fmt.Println("  4. Start building courses with: /pa create <name>")
			return nil
		},
	}
}

func newVoteMapCmd() *cobra.Command {
	var duration int
	var maxChoices int

	cmd := &cobra.Command{
		Use:   "vote-map",
		Short: "Start a map vote (CS:GO style)",
		Long: `Broadcast a map vote to all online players. Players type a number
in chat to vote. After the timer expires, the winning map is loaded
and all players are teleported.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			screen := management.NewScreenManager(runner, cfg.SessionName)

			if !screen.IsRunning(ctx) {
				return fmt.Errorf("server not running — start it first with: mc-dad-server start")
			}

			result, err := vote.RunVote(ctx, vote.Config{
				Maps:       management.ParkourMaps,
				Duration:   time.Duration(duration) * time.Second,
				MaxChoices: maxChoices,
				ServerDir:  cfg.Dir,
				Screen:     screen,
				Output:     output,
			})
			if err != nil {
				return err
			}

			output.Success("Map vote complete: %s (%d voters)", result.Winner, result.Voters)
			return nil
		},
	}

	cmd.Flags().IntVar(&duration, "duration", cfg.VoteDuration, "Vote duration in seconds")
	cmd.Flags().IntVar(&maxChoices, "choices", cfg.VoteChoices, "Number of maps to vote on")
	return cmd
}

func newRotateParkourCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate-parkour",
		Short: "Rotate the featured parkour map",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			screen := management.NewScreenManager(runner, cfg.SessionName)

			if !screen.IsRunning(ctx) {
				output.Info("Server not running, skipping rotation")
				return nil
			}

			return management.RotateParkour(ctx, cfg.Dir, screen, output)
		},
	}
}
