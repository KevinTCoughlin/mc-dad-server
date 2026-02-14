package main

import (
	"embed"
	"os"

	"github.com/KevinTCoughlin/mc-dad-server/internal/cli"
	"github.com/KevinTCoughlin/mc-dad-server/internal/configs"
)

//go:embed all:embedded
var embeddedFS embed.FS

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	configs.SetEmbeddedFS(embeddedFS)

	cmd := cli.NewRootCmd(version, commit)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
