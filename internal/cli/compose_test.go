package cli

import (
	"strings"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

func TestGenerateComposeRejectsUnsupportedPaper(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"26.4", "26.4-snapshot-5"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()

			cmd := paperComposeCmd(t, version)
			err := cmd.Run(nil, platform.NewMockRunner(), ui.NewWriter(&strings.Builder{}, false), nil)
			if err == nil {
				t.Fatal("expected Paper version to be refused before compose.yml is written")
			}
			if !strings.Contains(err.Error(), "resolving Paper version") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func paperComposeCmd(t *testing.T, version string) GenerateComposeCmd {
	t.Helper()
	return GenerateComposeCmd{
		Port:       25565,
		Memory:     "2G",
		Type:       "paper",
		MOTD:       "test",
		Players:    20,
		Difficulty: "normal",
		Gamemode:   "survival",
		GC:         "g1gc",
		MCVersion:  version,
		Output:     t.TempDir(),
	}
}
