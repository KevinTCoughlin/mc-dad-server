package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KevinTCoughlin/mc-dad-server/internal/config"
	"github.com/KevinTCoughlin/mc-dad-server/internal/configs"
	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
	"github.com/alecthomas/kong"
)

func TestContainerMemoryDefaultsToAuto(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"setup-container"},
		{"generate-compose", "--output", t.TempDir()},
	} {
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
			got := ""
			switch args[0] {
			case "setup-container":
				got = cmd.SetupContainer.Memory
			case "generate-compose":
				got = cmd.GenerateCompose.Memory
			}
			if got != "auto" {
				t.Fatalf("%s --memory default = %q, want auto", args[0], got)
			}
		})
	}
}

func TestResolveContainerMemoryFrom(t *testing.T) {
	t.Parallel()

	const gib = 1024 * 1024 * 1024

	t.Run("explicit heap", func(t *testing.T) {
		t.Parallel()
		plan, err := resolveContainerMemory("4G", ui.NewWriter(&strings.Builder{}, false))
		if err != nil {
			t.Fatalf("resolveContainerMemory() error = %v", err)
		}
		if plan.Heap != "4G" || plan.Limit != "5G" || plan.Percent != 0 {
			t.Fatalf("plan = %+v", plan)
		}
	})

	t.Run("explicit rejected", func(t *testing.T) {
		t.Parallel()
		if _, err := resolveContainerMemory("lots", ui.NewWriter(&strings.Builder{}, false)); err == nil {
			t.Fatal("expected invalid memory to fail")
		}
	})

	t.Run("auto from ram", func(t *testing.T) {
		t.Parallel()
		var buf strings.Builder
		plan, err := resolveContainerMemoryFrom(8*gib, nil, ui.NewWriter(&buf, false))
		if err != nil {
			t.Fatalf("resolveContainerMemoryFrom() error = %v", err)
		}
		if plan.Limit != "6G" || plan.Percent != config.ContainerHeapPercent || plan.Heap != "" {
			t.Fatalf("plan = %+v", plan)
		}
		if !strings.Contains(buf.String(), "6G") || !strings.Contains(buf.String(), "75%") {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("detect failure falls back", func(t *testing.T) {
		t.Parallel()
		var buf strings.Builder
		plan, err := resolveContainerMemoryFrom(0, errors.New("no meminfo"), ui.NewWriter(&buf, false))
		if err != nil {
			t.Fatalf("resolveContainerMemoryFrom() error = %v", err)
		}
		if plan.Heap != config.FallbackHeap || plan.Limit != "3G" || plan.Percent != 0 {
			t.Fatalf("plan = %+v", plan)
		}
		if !strings.Contains(buf.String(), "Could not detect system RAM") {
			t.Fatalf("output = %q", buf.String())
		}
	})

	t.Run("tight machine warns", func(t *testing.T) {
		t.Parallel()
		var buf strings.Builder
		plan, err := resolveContainerMemoryFrom(gib, nil, ui.NewWriter(&buf, false))
		if err != nil {
			t.Fatalf("resolveContainerMemoryFrom() error = %v", err)
		}
		if plan.Limit != "512M" || plan.Percent != config.ContainerHeapPercent {
			t.Fatalf("plan = %+v", plan)
		}
		if !strings.Contains(buf.String(), "under 2 GiB") {
			t.Fatalf("output = %q", buf.String())
		}
	})
}

func TestSetupContainerWritesAutoMemory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	plan := expectedContainerPlan(t)
	cmd := SetupContainerCmd{
		Port:       25565,
		Memory:     "auto",
		Type:       "paper",
		MOTD:       "test",
		Players:    20,
		Difficulty: "normal",
		Gamemode:   "survival",
		GC:         "g1gc",
		Whitelist:  true,
		MCVersion:  "26.3",
	}
	var buf strings.Builder
	err := cmd.Run(&Globals{Runtime: "podman"}, platform.NewMockRunner(), ui.NewWriter(&buf, false), repoDeployer(t))
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	unit, err := os.ReadFile(filepath.Join(home, ".config", "containers", "systemd", "minecraft.container"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(unit)
	if plan.Percent > 0 {
		// systemd treats % as a specifier, so the unit file stores %%.
		wantMem := "MEMORY=75%%"
		wantLimit := "MemoryMax=" + plan.Limit
		if !strings.Contains(text, wantMem) || !strings.Contains(text, wantLimit) {
			t.Fatalf("unit missing %q and %q:\n%s", wantMem, wantLimit, text)
		}
		if strings.Contains(text, "Environment=MEMORY="+plan.Limit) {
			t.Fatalf("unit used the limit as a fixed heap:\n%s", text)
		}
	} else if !strings.Contains(text, "Environment=MEMORY="+plan.Heap) || !strings.Contains(text, "MemoryMax="+plan.Limit) {
		t.Fatalf("fallback unit missing heap %s limit %s:\n%s", plan.Heap, plan.Limit, text)
	}
}

func TestSetupContainerExplicitMemory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := SetupContainerCmd{
		Port:       25565,
		Memory:     "4G",
		Type:       "vanilla",
		MOTD:       "test",
		Players:    10,
		Difficulty: "easy",
		Gamemode:   "creative",
		GC:         "zgc",
		MCVersion:  "26.3",
	}
	if err := cmd.Run(&Globals{Runtime: "podman"}, platform.NewMockRunner(), ui.NewWriter(&strings.Builder{}, false), repoDeployer(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	unit, err := os.ReadFile(filepath.Join(home, ".config", "containers", "systemd", "minecraft.container"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(unit)
	if !strings.Contains(text, "MEMORY=4G") || !strings.Contains(text, "MemoryMax=5G") {
		t.Fatalf("explicit memory was not kept:\n%s", text)
	}
	if strings.Contains(text, `Environment="MEMORY=75`) {
		t.Fatalf("explicit memory used a percentage:\n%s", text)
	}
}

func TestGenerateComposeAutoMemory(t *testing.T) {
	plan := expectedContainerPlan(t)
	dir := t.TempDir()
	cmd := GenerateComposeCmd{
		Port:       25565,
		Memory:     "auto",
		Type:       "vanilla",
		MOTD:       "test",
		Players:    20,
		Difficulty: "normal",
		Gamemode:   "survival",
		GC:         "g1gc",
		Whitelist:  true,
		MCVersion:  "26.3",
		Output:     dir,
	}
	if err := cmd.Run(nil, platform.NewMockRunner(), ui.NewWriter(&strings.Builder{}, false), repoDeployer(t)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if plan.Percent > 0 {
		if !strings.Contains(text, `MEMORY: "75%"`) || !strings.Contains(text, `mem_limit: "`+plan.Limit+`"`) {
			t.Fatalf("compose missing percentage heap or limit %s:\n%s", plan.Limit, text)
		}
	} else if !strings.Contains(text, `MEMORY: "`+plan.Heap+`"`) || !strings.Contains(text, `mem_limit: "`+plan.Limit+`"`) {
		t.Fatalf("compose missing fallback heap %s limit %s:\n%s", plan.Heap, plan.Limit, text)
	}
}

func expectedContainerPlan(t *testing.T) config.ContainerPlan {
	t.Helper()
	total, err := platform.TotalRAM()
	if err != nil {
		plan, perr := config.ExplicitContainerPlan(config.FallbackHeap)
		if perr != nil {
			t.Fatal(perr)
		}
		return plan
	}
	plan, _, err := config.AutoContainerPlan(total)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func repoDeployer(t *testing.T) *configs.Deployer {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return configs.NewDeployer(os.DirFS(dir))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
