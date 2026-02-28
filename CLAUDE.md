# CLAUDE.md

## What is this?

MC Dad Server — a single-binary CLI that installs and manages a Minecraft server (Java/Bedrock cross-play via Geyser, PaperMC tuned configs, parkour, chat filter, playit.gg tunnel). Supports bare-metal (GNU screen) and container (Podman/Docker) modes. Targets Linux (primary), macOS, and Windows. Written in Go.

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
    commands.go             Simple commands (start, stop, status, backup, parkour, vote) + mode resolution
    console.go              Console command (bridges to console package)
    install.go              Install command (flags, validation, orchestration)
    license.go              License commands (validate, activate, deactivate)
  console/                  Interactive TUI console (Bubbletea)
    console.go              Bubbletea model, Run() entry point
    commands.go             Command dispatcher (parses input, calls management logic)
    logtail.go              Polls logs/latest.log, sends lines as TUI messages
  config/                   ServerConfig struct, defaults, validation
  configs/                  Deploy embedded config files + start script
  container/                RCON client (rcon.go) and Podman container manager (container.go)
  license/                  LemonSqueezy license client + manager
  management/               ServerManager interface, screen/container backends, backup, process stats, parkour rotation
  nag/                      Shareware nag/grace-period logic
  parkour/                  Parkour map definitions and setup
  platform/                 OS detection, package install, Java, firewall, cron, services
  plugins/                  Plugin installation (Geyser, chat filter, Hangar, GitHub)
  server/                   Server JAR download (Paper, Fabric, Vanilla)
  tunnel/                   playit.gg tunnel setup
  ui/                       Colored terminal output
  vote/                     In-game map vote system
  bun/                      Bun scripting sidecar (install, deploy, embed)
embedded/bun/               Bun runtime framework (TypeScript) and templates
  runtime/                  Framework files (server, events, rcon, webhooks, security)
  scripts/                  Example user scripts
```

## Architecture

- **CLI framework**: Kong (struct tags + dependency injection, no globals)
- **Dependency injection**: `main.go` creates `runner` (CommandRunner) and `output` (UI), passes them to each command's `Run()` method via Kong bindings
- **Config**: `config.ServerConfig` is framework-agnostic — built from Kong flags in `InstallCmd.toConfig()`, validated via `cfg.Validate()`
- **Embedded assets**: `//go:embed all:embedded` in main.go — configs, templates, blocked-words list
- **Version**: Set via ldflags (`-X main.version=... -X main.commit=...`) by goreleaser
- **Server modes**: `--mode auto|screen|container` — `ServerManager` interface (`management/manager.go`) with `ScreenManager` and `container.Manager` backends. Auto-detection checks for running container first, falls back to screen.
- **Container**: Eclipse Temurin 21 JRE on Ubuntu Noble (builder stays on Debian Trixie slim). FIFO-based stdin (`entrypoint.sh`), RCON for remote commands (`container/rcon.go`), graceful 30s shutdown countdown.

## Key Conventions

- No package-level mutable state — all dependencies passed explicitly
- `platform.CommandRunner` interface for all shell-outs (testable via `MockRunner`)
- `ui.UI` for all user-facing output (color auto-detected); `ui.NewWriter(w, color)` to capture output into a buffer
- Helpers in install.go take explicit `cfg`, `runner`, `output` params
- Enum validation via Kong tags; range validation in `config.Validate()`
- `context.Background()` in each command's `Run()` method

## CI

GitHub Actions workflows:
- **ci.yml**: checks (lint + tidy + vet), test (ubuntu; macOS/Windows on push only), build (goreleaser snapshot), hadolint (Containerfile), shellcheck (entrypoint.sh)
- **container.yml**: build container image, Trivy security scan, push to ghcr.io, SBOM generation
- **dependency-check.yml**: weekly go.sum tidy check, outdated deps, Go toolchain version
- **release.yml**: goreleaser on tags
- **nightly.yml**: nightly pre-release builds
