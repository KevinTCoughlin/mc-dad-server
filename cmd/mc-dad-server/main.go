package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/KevinTCoughlin/mc-dad-server/internal/cli"
	"github.com/KevinTCoughlin/mc-dad-server/internal/configs"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
	"github.com/alecthomas/kong"
)

//go:embed all:embedded
var embeddedFS embed.FS

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	configs.SetEmbeddedFS(embeddedFS)

	var app cli.CLI
	var runner platform.CommandRunner = platform.NewOSCommandRunner()
	output := ui.Default()

	ctx := kong.Parse(&app,
		kong.Name("mc-dad-server"),
		kong.Description("MC Dad Server — Minecraft server in 60 seconds.\n\nNo Docker. No Kubernetes. No nonsense. Just one command\nand you're hosting Minecraft — with Bedrock cross-play,\nParkour courses, and tuned configs out of the box."),
		kong.Vars{"version": fmt.Sprintf("%s (%s)", version, commit)},
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)

	ctx.BindTo(runner, (*platform.CommandRunner)(nil))
	err := ctx.Run(&app.Globals, output)
	ctx.FatalIfErrorf(err)
	os.Exit(0)
}
