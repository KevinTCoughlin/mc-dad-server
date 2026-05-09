# Copilot Instructions

## Priority Order

Follow instructions in this order:

1. System/developer instructions
2. This file
3. User request

If instructions conflict, higher priority wins.

## Working Style

- Inspect the relevant code before editing.
- Make the smallest change that solves the problem.
- Prefer explicit, local changes over broad refactors.
- Avoid introducing abstractions unless they clearly improve the code.
- Ask a clarifying question only when the request is ambiguous or risky.
- If you can verify a change, do so before finishing.
- After 1–2 failed fix attempts, stop and explain what is blocking you.

## Project

MC Dad Server is a single-binary Go CLI for installing and managing Minecraft servers.

It supports:

- Java/Bedrock cross-play via Geyser
- PaperMC-tuned configs
- Parkour maps
- Chat filtering
- playit.gg tunnels

Targets Linux first, with macOS and Windows support.

## Build & Test

```bash
go build ./cmd/mc-dad-server/
go test -race ./...
go vet ./...
golangci-lint run
gofumpt -w .
```

## Project Layout

- `cmd/mc-dad-server/` — entry point, embedded assets, templates
- `internal/cli/` — Kong CLI structs and command handlers
- `internal/config/` — `ServerConfig`, defaults, validation
- `internal/configs/` — embedded config deployment and start scripts
- `internal/license/` — LemonSqueezy license client and manager
- `internal/management/` — screen session, backup, process stats, parkour rotation
- `internal/nag/` — shareware nag and grace-period logic
- `internal/parkour/` — parkour map definitions and setup
- `internal/platform/` — OS detection, package install, Java, firewall, cron, services
- `internal/plugins/` — plugin installation for Geyser, chat filter, Hangar, and GitHub
- `internal/server/` — server JAR download for Paper, Fabric, and Vanilla
- `internal/tunnel/` — playit.gg tunnel setup
- `internal/ui/` — colored terminal output
- `internal/vote/` — in-game map vote system
- `internal/bun/` — Bun scripting sidecar for install, deploy, and embed
- `embedded/bun/` — Bun runtime framework and templates

## Architecture & Conventions

- **CLI framework**: Kong with declarative struct tags and dependency injection.
- **Dependency injection**: `main.go` creates `runner` (`CommandRunner`) and `output` (`UI`), then binds them into command handlers via Kong. Use `ctx.BindTo()` for interfaces.
- **No package-level mutable state**: pass dependencies explicitly.
- **Shell-outs**: use `platform.CommandRunner` so commands are testable with `MockRunner`.
- **User output**: use `ui.UI` so color is auto-detected consistently.
- **Config**: build `config.ServerConfig` from Kong flags in `InstallCmd.toConfig()`, then validate with `cfg.Validate()`.
- **Embedded assets**: use `//go:embed all:embedded` in `main.go`.
- **Versioning**: set `main.version` and `main.commit` via ldflags in goreleaser.
- **Enum validation**: prefer Kong struct tags; use range validation in `config.Validate()`.
- **Context**: use `context.Background()` in each command's `Run()` method.

## Code Style

- Go 1.26, formatted with gofumpt.
- Keep helpers minimal and explicit.
- Prefer explicit parameters over closures that capture shared state.
- Only add comments when the logic is not self-evident.
- CI runs lint, tidy, vet, tests, and a goreleaser snapshot build.

## Quality Bar

- Preserve existing behavior unless the user asks for a change.
- Keep edits scoped to the requested task.
- Prefer readable code over clever code.
- If validation fails, report the exact failure and likely cause.
- Do not claim success without actually verifying the change when verification is available.
