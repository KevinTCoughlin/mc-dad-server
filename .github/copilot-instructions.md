# Copilot Instructions

## Project

MC Dad Server — a single-binary Go CLI that installs and manages a Minecraft server (Java/Bedrock cross-play via Geyser, PaperMC tuned configs, parkour, chat filter, playit.gg tunnel). Targets Linux (primary), macOS, and Windows.

## Build & Test

```bash
go build ./cmd/mc-dad-server/          # build
go test -race ./...                    # test
go vet ./...                           # vet
golangci-lint run                      # lint
gofumpt -w .                           # format
```

## Project Layout

- `cmd/mc-dad-server/` — entry point, embedded assets (configs, templates)
- `internal/cli/` — Kong CLI structs and command handlers
- `internal/config/` — `ServerConfig` struct, defaults, validation
- `internal/configs/` — deploy embedded config files + start script
- `internal/license/` — LemonSqueezy license client + manager
- `internal/management/` — screen session, backup, process stats, parkour rotation
- `internal/nag/` — shareware nag/grace-period logic
- `internal/parkour/` — parkour map definitions and setup
- `internal/platform/` — OS detection, package install, Java, firewall, cron, services
- `internal/plugins/` — plugin installation (Geyser, chat filter, Hangar, GitHub)
- `internal/server/` — server JAR download (Paper, Fabric, Vanilla)
- `internal/tunnel/` — playit.gg tunnel setup
- `internal/ui/` — colored terminal output
- `internal/vote/` — in-game map vote system
- `internal/bun/` — Bun scripting sidecar (install, deploy, embed)
- `embedded/bun/` — Bun runtime framework (TypeScript) and templates

## Architecture & Conventions

- **CLI framework**: Kong (declarative struct tags + dependency injection, no globals)
- **Dependency injection**: `main.go` creates `runner` (`CommandRunner`) and `output` (`UI`), passes them to each command's `Run()` method via Kong bindings. Use `ctx.BindTo()` for interface types.
- **No package-level mutable state** — all dependencies passed explicitly to functions
- **`platform.CommandRunner`** interface for all shell-outs — testable via `MockRunner`
- **`ui.UI`** for all user-facing output — color auto-detected
- **Config**: `config.ServerConfig` is framework-agnostic — built from Kong flags in `InstallCmd.toConfig()`, validated via `cfg.Validate()`
- **Embedded assets**: `//go:embed all:embedded` in `main.go`
- **Version**: set via ldflags (`-X main.version=... -X main.commit=...`) by goreleaser
- **Enum validation**: Kong struct tags; range validation in `config.Validate()`
- **Context**: `context.Background()` in each command's `Run()` method

## Code Style

- Go 1.24, formatted with gofumpt
- No unnecessary abstractions — keep helpers minimal and explicit
- Prefer passing explicit params over closures capturing shared state
- Only add comments where logic isn't self-evident
- CI runs checks (lint + tidy + vet) + test (ubuntu; macOS/Windows on push only) + build (goreleaser snapshot)
