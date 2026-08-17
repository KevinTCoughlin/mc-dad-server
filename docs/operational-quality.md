# Operational Quality Checklist

Use this checklist before cutting a release or validating a change that affects installation, startup, shutdown, downloads, or container behavior.

## Automated checks

- Run `just check`.
- Run `just coverage` when changing installer, download, server management, or platform logic.
- Confirm CI completed:
  - Go tidy, vet, lint, and tests.
  - Go coverage artifact upload.
  - CodeQL analysis.
  - Container build and Trivy scan when container files changed.
  - Hadolint and ShellCheck when container files changed.

## Installer smoke test

- Install into a temporary directory with default Paper settings.
- Install into a temporary directory with `--type vanilla`.
- Install into a temporary directory with `--type fabric`.
- Re-run install against an existing directory and confirm existing files are not corrupted.
- Confirm a failed server JAR download does not replace an existing `server.jar`.
- Confirm a checksum mismatch removes only the failed temporary download.
- Confirm generated configs include the selected port, memory, game mode, difficulty, MOTD, whitelist, and RCON settings.

## Server management smoke test

- Start, status, backup, and stop in screen mode.
- Start, status, backup, and stop in container mode.
- Confirm `--mode auto` prefers a running container and falls back to screen.
- Confirm graceful shutdown sends the countdown and stop command.
- Confirm RCON failures return useful errors without hanging.

## Platform matrix

- Ubuntu or Debian.
- Fedora or RHEL.
- Arch Linux.
- macOS.
- Windows.
- WSL2.
- Docker or Podman container mode.
- Raspberry Pi or another low-memory ARM host when changing memory or JVM flags.

## Release readiness

- Confirm `goreleaser build --snapshot --clean` succeeds.
- Confirm README quick-start commands still match the released artifact names.
- Confirm docs for install, scripting, licensing, and parkour are still accurate.
- Confirm dependencies are tidy and direct dependency updates have been reviewed.
- Confirm no secrets are present in changed files.
