# Contributing to MC Dad Server

PRs welcome! This is for dads, by dads. Keep it simple.

## Quick Start (GitHub Codespaces)

Click **Code → Codespaces → New codespace** on the repo page. The dev container comes pre-configured with Go 1.24, Java 21, Kotlin, and all required tooling — zero local setup.

## Local Development

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| [Go](https://go.dev/dl/) | 1.24+ | Build the CLI binary |
| [just](https://github.com/casey/just#installation) | latest | Task runner (replaces Make) |
| [Java (Temurin)](https://adoptium.net/) | 21+ | Minecraft server runtime |
| [Kotlin](https://kotlinlang.org/docs/command-line.html) | latest | Plugin development |
| [golangci-lint](https://golangci-lint.run/welcome/install/) | latest | Go linter |
| [gofumpt](https://github.com/mvdan/gofumpt) | latest | Go formatter |

### Setup

```bash
git clone https://github.com/KevinTCoughlin/mc-dad-server.git
cd mc-dad-server
just tools   # install golangci-lint, gofumpt, goreleaser
go mod download
```

### Common Commands

```bash
just build   # build binary
just test    # run tests with race detector
just lint    # run golangci-lint
just fmt     # format code with gofumpt
just vet     # run go vet
just check   # fmt + vet + lint + test (run before submitting)
just clean   # remove build artifacts

# Container development
just container-build   # build container image
just container-up      # start with Podman Compose
just container-down    # stop containers
just container-logs    # follow server logs
just container-shell   # shell into the container
```

## Project Layout

```
cmd/mc-dad-server/     Entry point — CLI binary
internal/
  cli/                 Kong CLI structs and command handlers
  config/              Server configuration and validation
  configs/             Embedded Minecraft config files
  container/           RCON client and Podman container manager
  license/             LemonSqueezy license client and manager
  management/          ServerManager interface, backup, screen, process mgmt
  nag/                 Shareware nag/grace-period logic
  parkour/             Parkour world and map features
  platform/            OS-specific helpers (Java install, cron, firewall)
  plugins/             Plugin managers (Geyser, Hangar, ChatSentry)
  server/              Server types (Paper, Fabric, Vanilla)
  tunnel/              Networking (playit.gg)
  ui/                  Terminal output and summaries
  vote/                Voting system
  bun/                 Bun scripting sidecar (install, deploy, embed)
embedded/bun/          Bun runtime framework (TypeScript) and templates
configs/               Minecraft server config templates
docs/                  Documentation and GitHub Pages
scripts/               Utility scripts
Containerfile          Multi-stage container build (Debian Trixie + Temurin 21)
entrypoint.sh          Container entrypoint with graceful shutdown
compose.yml            Podman/Docker Compose configuration
quadlet/               Systemd Quadlet unit for rootless Podman
```

## Submitting Changes

1. Fork the repo and create a branch from `main`.
2. Make your changes — keep diffs small and focused.
3. Run `just check` to verify formatting, linting, and tests pass.
4. Open a pull request against `main`.

CI runs lint (golangci-lint), tests (Ubuntu, macOS, Windows), cross-platform build verification, hadolint (Containerfile), and shellcheck (entrypoint.sh) automatically on every PR.

## Style Guide

- Follow existing code conventions — the linter enforces most rules.
- Avoid stuttering in exported names (e.g., use `Config` not `VoteConfig` in package `vote`).
- Always handle or explicitly ignore `Close()` error returns (`_ = f.Close()`).
- Run `just fmt` before committing to keep formatting consistent.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
