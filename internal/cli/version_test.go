package cli

import (
	"testing"

	"github.com/alecthomas/kong"

	"github.com/KevinTCoughlin/mc-dad-server/internal/server"
)

func TestMCVersionDefaultsToSupportedPaper(t *testing.T) {
	t.Parallel()

	cases := [][]string{
		{"install", "--dir", t.TempDir()},
		{"generate-compose", "--output", t.TempDir()},
		{"setup-container"},
	}
	for _, args := range cases {
		t.Run(args[0], func(t *testing.T) {
			t.Parallel()

			var cmd CLI
			parser, err := kong.New(&cmd)
			if err != nil {
				t.Fatalf("kong.New: %v", err)
			}
			if _, err := parser.Parse(args); err != nil {
				t.Fatalf("parse %v: %v", args, err)
			}
			got := mcVersion(&cmd, args[0])
			if got != server.SupportedPaperVersion {
				t.Fatalf("%s default = %q, want %s", args[0], got, server.SupportedPaperVersion)
			}
		})
	}
}

func mcVersion(cmd *CLI, name string) string {
	switch name {
	case "install":
		return cmd.Install.MCVersion
	case "generate-compose":
		return cmd.GenerateCompose.MCVersion
	case "setup-container":
		return cmd.SetupContainer.MCVersion
	default:
		return ""
	}
}
