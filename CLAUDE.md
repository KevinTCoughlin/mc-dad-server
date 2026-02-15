# CLAUDE.md

## What is this?

MC Dad Server — a single-binary CLI that installs and manages a Minecraft server (Java/Bedrock cross-play via Geyser, PaperMC tuned configs, parkour, chat filter, playit.gg tunnel). Targets Linux (primary), macOS, and Windows. Written in Go.

## Build & Test

```bash
go build ./cmd/mc-dad-server/          # build
go test -race ./...                    # test (all packages)
go vet ./...                           # vet
golangci-lint run                      # lint (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
gofumpt -w .                           # format (install: go install mvdan.cc/gofumpt@latest)
```

Or via justfile: `just build`, `just test`, `just lint`, `just check` (all).

## Project Layout

```
cmd/mc-dad-server/          Entry point, embedded assets (configs, templates)
internal/
  cli/                      Kong CLI structs and command handlers
    cli.go                  Globals + CLI struct (top-level command tree)
    commands.go             Simple commands (start, stop, status, backup, parkour, vote)
    install.go              Install command (flags, validation, orchestration)
    license.go              License commands (validate, activate, deactivate)
  config/                   ServerConfig struct, defaults, validation
  configs/                  Deploy embedded config files + start script
  license/                  LemonSqueezy license client + manager
  management/               Screen session, backup, process stats, parkour rotation
  nag/                      Shareware nag/grace-period logic
  parkour/                  Parkour map definitions and setup
  platform/                 OS detection, package install, Java, firewall, cron, services
  plugins/                  Plugin installation (Geyser, chat filter, Hangar, GitHub)
  server/                   Server JAR download (Paper, Fabric, Vanilla)
  tunnel/                   playit.gg tunnel setup
  ui/                       Colored terminal output
  vote/                     In-game map vote system
```

## Architecture

- **CLI framework**: Kong (struct tags + dependency injection, no globals)
- **Dependency injection**: `main.go` creates `runner` (CommandRunner) and `output` (UI), passes them to each command's `Run()` method via Kong bindings
- **Config**: `config.ServerConfig` is framework-agnostic — built from Kong flags in `InstallCmd.toConfig()`, validated via `cfg.Validate()`
- **Embedded assets**: `//go:embed all:embedded` in main.go — configs, templates, blocked-words list
- **Version**: Set via ldflags (`-X main.version=... -X main.commit=...`) by goreleaser

## Key Conventions

- No package-level mutable state — all dependencies passed explicitly
- `platform.CommandRunner` interface for all shell-outs (testable via `MockRunner`)
- `ui.UI` for all user-facing output (color auto-detected)
- Helpers in install.go take explicit `cfg`, `runner`, `output` params
- Enum validation via Kong tags; range validation in `config.Validate()`
- `context.Background()` in each command's `Run()` method

## CI

GitHub Actions runs lint, test (ubuntu/macos/windows matrix), and cross-compile build verification on push to main and PRs. Release via goreleaser on tags.
