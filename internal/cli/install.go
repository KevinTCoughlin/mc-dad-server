package cli

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strings"

	bunpkg "github.com/KevinTCoughlin/mc-dad-server/internal/bun"
	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/configs"
	"github.com/KevinTCoughlin/mc-dad-server/internal/nag"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/plugins"
	"github.com/KevinTCoughlin/mc-dad-server/internal/server"
	"github.com/KevinTCoughlin/mc-dad-server/internal/tunnel"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// InstallCmd installs and configures a Minecraft server.
type InstallCmd struct {
	Edition    string `help:"Server edition" default:"java" enum:"java,bedrock"`
	Port       int    `help:"Server port" default:"25565"`
	Memory     string `help:"RAM allocation (e.g., 2G, 4G)" default:"2G"`
	Type       string `help:"Server type" default:"paper" enum:"paper,fabric,vanilla" name:"type"`
	MOTD       string `help:"Message of the day" default:"Dads Minecraft Server" name:"motd"`
	Players    int    `help:"Max players" default:"20" name:"players"`
	Difficulty string `help:"Difficulty" default:"normal" enum:"peaceful,easy,normal,hard"`
	Gamemode   string `help:"Game mode" default:"survival" enum:"survival,creative,adventure" name:"gamemode"`
	GC         string `help:"Garbage collector" default:"g1gc" enum:"g1gc,zgc,G1GC,ZGC" name:"gc"`
	Whitelist  bool   `help:"Enable whitelist" default:"true" negatable:""`
	ChatFilter bool   `help:"Install chat filter plugin" default:"true" name:"chat-filter" negatable:""`
	Playit     bool   `help:"Set up playit.gg tunnel" default:"true" negatable:""`
	Bun        bool   `help:"[Experimental] Enable Bun scripting sidecar" default:"false" name:"experimental-bun"`
	MCVersion  string `help:"Minecraft version" default:"latest" name:"mc-version"`
}

func (cmd *InstallCmd) toConfig(globals *Globals) *config.ServerConfig {
	return &config.ServerConfig{
		Edition:      cmd.Edition,
		Dir:          globals.Dir,
		Port:         cmd.Port,
		Memory:       cmd.Memory,
		ServerType:   cmd.Type,
		MOTD:         cmd.MOTD,
		MaxPlayers:   cmd.Players,
		Difficulty:   cmd.Difficulty,
		GameMode:     cmd.Gamemode,
		GCType:       strings.ToLower(cmd.GC),
		Whitelist:    cmd.Whitelist,
		ChatFilter:   cmd.ChatFilter,
		EnablePlayit: cmd.Playit,
		EnableBun:    cmd.Bun,
		Version:      cmd.MCVersion,
		SessionName:  globals.Session,
		MaxBackups:   5,
		VoteDuration: 30,
		VoteChoices:  5,
	}
}

// Run installs and configures a Minecraft server.
func (cmd *InstallCmd) Run(globals *Globals, runner platform.CommandRunner, output *ui.UI) error {
	ctx := context.Background()
	cfg := cmd.toConfig(globals)

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
	if err := installDependencies(ctx, &plat, cfg, runner, output); err != nil {
		return err
	}

	// Download server
	if err := downloadServer(ctx, cfg, runner, output); err != nil {
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
		if err := plugins.SetupChatFilter(cfg.Dir, output); err != nil {
			output.Warn("Chat filter setup failed: %v", err)
		}
	}

	// Plugins
	if cfg.ServerType == "paper" {
		if err := plugins.InstallAll(ctx, cfg.Dir, output); err != nil {
			output.Warn("Some plugins failed to install: %v", err)
		}
	}

	// Bun scripting sidecar
	if cfg.EnableBun {
		if err := bunpkg.DeployScripts(cfg); err != nil {
			return fmt.Errorf("deploying bun scripts: %w", err)
		}
		if err := bunpkg.InstallDependencies(ctx, runner, cfg.Dir); err != nil {
			output.Warn("Bun dependency install failed: %v", err)
		}
		output.Success("Bun scripting sidecar deployed to bun-scripts/")
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
	if err := setupService(ctx, &plat, runner, cfg); err != nil {
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
		BedrockPort:  config.BedrockPort,
		Memory:       cfg.Memory,
		GCType:       cfg.GCType,
		Whitelist:    cfg.Whitelist,
		Difficulty:   cfg.Difficulty,
		GameMode:     cfg.GameMode,
		ChatFilter:   cfg.ChatFilter,
		PlayitSetup:  cfg.EnablePlayit,
		BunEnabled:   cfg.EnableBun,
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

func installDependencies(ctx context.Context, plat *platform.Platform, cfg *config.ServerConfig, runner platform.CommandRunner, output *ui.UI) error {
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

	if cfg.EnableBun {
		if err := bunpkg.InstallBun(ctx, runner, plat, output); err != nil {
			return fmt.Errorf("installing Bun: %w", err)
		}
	}

	return nil
}

func downloadServer(ctx context.Context, cfg *config.ServerConfig, runner platform.CommandRunner, output *ui.UI) error {
	output.Step("Downloading Minecraft Server")

	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return fmt.Errorf("creating server directory: %w", err)
	}

	return server.Download(ctx, cfg.ServerType, cfg.Version, cfg.Dir, runner, output)
}

func setupService(ctx context.Context, plat *platform.Platform, runner platform.CommandRunner, cfg *config.ServerConfig) error {
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
