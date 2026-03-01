# MC Dad Server

[![CI](https://github.com/KevinTCoughlin/mc-dad-server/actions/workflows/ci.yml/badge.svg)](https://github.com/KevinTCoughlin/mc-dad-server/actions/workflows/ci.yml)
[![Release](https://github.com/KevinTCoughlin/mc-dad-server/actions/workflows/release.yml/badge.svg)](https://github.com/KevinTCoughlin/mc-dad-server/actions/workflows/release.yml)
[![Nightly](https://github.com/KevinTCoughlin/mc-dad-server/actions/workflows/nightly.yml/badge.svg)](https://github.com/KevinTCoughlin/mc-dad-server/actions/workflows/nightly.yml)
[![Container](https://github.com/KevinTCoughlin/mc-dad-server/actions/workflows/container.yml/badge.svg)](https://github.com/KevinTCoughlin/mc-dad-server/actions/workflows/container.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/KevinTCoughlin/mc-dad-server)](https://goreportcard.com/report/github.com/KevinTCoughlin/mc-dad-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**Minecraft server in 60 seconds — for busy dads who just want their kids to play.**

One binary, zero fuss. Run `mc-dad-server install` for a bare-metal server, or use the included Containerfile for Docker/Podman — with Bedrock cross-play, Parkour courses, and tuned configs out of the box.

## Quick Start

```bash
# Download the binary for your platform
curl -fsSL https://github.com/KevinTCoughlin/mc-dad-server/releases/latest/download/mc-dad-server-$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') -o mc-dad-server
chmod +x mc-dad-server

# Install everything
./mc-dad-server install
```

Or via `go install`:

```bash
go install github.com/KevinTCoughlin/mc-dad-server/cmd/mc-dad-server@latest
mc-dad-server install
```

That's it. Your server is running.

## Batteries Included

Every Paper server install comes loaded with:

| Plugin | What It Does |
|--------|-------------|
| **Geyser + Floodgate** | Kids on iPad, Switch, and phones connect to your Java server. No second account needed. |
| **Parkour** | Build obstacle courses. Checkpoints, leaderboards, death counters. Kids go nuts for this. |
| **WorldEdit** | Fast course building. `//set`, `//stack`, `//copy` — build a parkour map in minutes. |
| **Multiverse-Core** | Separate worlds for parkour, creative, survival. Keep things organized. |
| **ChatSentry** | Configurable chat filter with blocked words list. Kid-safe by default. |

Plus battle-tested PaperMC configs (bukkit.yml, spigot.yml, paper-global.yml) tuned from a real production server.

## What It Does

- Installs Adoptium Temurin Java 25 (open-source, no Oracle)
- Downloads Paper MC (optimized, fast, plugin-ready)
- Deploys tuned server configs from a battle-tested PaperMC server
- Downloads and installs plugins (Geyser, Parkour, WorldEdit, Multiverse, ChatSentry)
- Sets up start script with GC selection (G1GC or ZGC)
- Sets up auto-start on boot (systemd or launchd)
- Daily automatic backups with rotation
- Opens firewall ports for Java (25565) and Bedrock (19132)
- Optional playit.gg tunnel — **no port forwarding needed**
- Optional [TypeScript/JS scripting sidecar](docs/scripting.md) for custom automation (`--experimental-bun`)

## Works On

| Platform | Status |
|----------|--------|
| Ubuntu/Debian | Fully supported |
| Fedora/RHEL | Fully supported |
| Arch Linux | Fully supported |
| macOS | Fully supported |
| Windows (x64/ARM64) | Fully supported |
| WSL2 (Windows) | Fully supported |
| Docker / Podman | Fully supported (`--mode container`) |
| Raspberry Pi | Works (use `--memory 1G`) |

## Options

```bash
mc-dad-server install \
  --type paper \
  --memory 4G \
  --port 25565 \
  --difficulty normal \
  --gamemode survival \
  --players 20 \
  --gc g1gc \
  --motd "Our Family Server"
```

| Flag | Default | Description |
|------|---------|-------------|
| `--edition` | `java` | `java` or `bedrock` |
| `--type` | `paper` | `paper`, `fabric`, or `vanilla` |
| `--dir` | `~/minecraft-server` | Install location |
| `--port` | `25565` | Server port |
| `--memory` | `2G` | RAM allocation |
| `--difficulty` | `normal` | `peaceful`, `easy`, `normal`, `hard` |
| `--gamemode` | `survival` | `survival`, `creative`, `adventure` |
| `--players` | `20` | Max player count |
| `--gc` | `g1gc` | `g1gc` (Aikar's flags) or `zgc` (low latency) |
| `--motd` | `Dads Minecraft Server` | Message of the day |
| `--playit` | `true` | Set up playit.gg tunnel (`--no-playit` to skip) |
| `--chat-filter` | `true` | Install chat filter plugin (`--no-chat-filter` to skip) |
| `--mc-version` | `latest` | Minecraft version |
| `--experimental-bun` | `false` | Enable [TypeScript/JS scripting sidecar](docs/scripting.md) |

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | `~/minecraft-server` | Server directory |
| `--session` | `minecraft` | Screen session / container name |
| `--mode` | `auto` | `auto`, `screen`, or `container` — how to manage the server process |

## Daily Commands

```bash
# Start the server (runs in background)
mc-dad-server start

# Check if it's running
mc-dad-server status

# View the server console (screen mode)
screen -r minecraft
# (Press Ctrl+A then D to detach)

# Stop the server (graceful 30s countdown)
mc-dad-server stop

# Manual backup
mc-dad-server backup
```

## Container Deployment

Run the server in Docker or Podman instead of a bare-metal screen session. The Containerfile uses Eclipse Temurin 25 JRE on Ubuntu Noble and includes Geyser, Floodgate, Parkour, Multiverse, and WorldEdit.

```bash
# Build and start with Podman Compose
podman compose up -d

# Or with Docker Compose
docker compose up -d

# CLI auto-detects container mode
mc-dad-server status              # auto-detects running container
mc-dad-server --mode container status   # force container mode

# Manage the container
mc-dad-server stop                # graceful shutdown with player countdown
podman logs -f minecraft          # follow server logs
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MEMORY` | `2G` | JVM heap size |
| `GC_TYPE` | `g1gc` | `g1gc` (Aikar's flags) or `zgc` |
| `PORT` | `25565` | Minecraft server port |
| `RCON_PORT` | `25575` | RCON port (localhost only) |
| `RCON_PASSWORD` | `changeme` | RCON password |
| `MC_VERSION` | `latest` | Minecraft version to download |

### Systemd Integration (Quadlet)

For always-on servers, install the Quadlet unit for rootless Podman with systemd:

```bash
mkdir -p ~/.config/containers/systemd
cp quadlet/minecraft.container ~/.config/containers/systemd/
systemctl --user daemon-reload
systemctl --user start minecraft
```

See `compose.yml` for volume mounts, port mappings, and security hardening (cap_drop, no-new-privileges, memory limits).

## Bedrock Cross-Play (iPad, Switch, Phone)

Your kids on Bedrock Edition connect to the same server as Java players. No extra accounts, no extra servers.

- **Java players:** connect to `your-ip:25565` as usual
- **Bedrock players:** connect to `your-ip` on port `19132`

Geyser handles the translation. Floodgate means they don't need a Java Minecraft account. Kids on Switch, iPad, phone, Windows 10 — they all just connect.

## Parkour

After your first server boot, set up the parkour world:

```bash
mc-dad-server setup-parkour
```

Then in-game:

```
/mv tp parkour          # teleport to parkour world
/pa setlobby            # set lobby at your position
/pa create MyCourse     # start building a course
/pa checkpoint          # add checkpoint where you stand
/pa finish              # set the finish line
/pa ready MyCourse      # open it up for players
```

### Pre-built Maps

Install 5 curated parkour maps from [Hielke Maps](https://hielkemaps.com) — no building required. Maps: Parkour Spiral, Spiral 3, Volcano, Pyramid, Paradise.

### Auto Map Rotation

Keep things fresh — rotate the featured parkour map:

```bash
# Manual rotation
mc-dad-server rotate-parkour

# Automate with cron (every 4 hours)
0 */4 * * * mc-dad-server rotate-parkour >> ~/minecraft-server/logs/rotation.log 2>&1
```

When a map rotates, all players get a broadcast and are teleported to the new featured map. See [docs/parkour.md](docs/parkour.md) for full details.

## Support

MC Dad Server is **free and fully functional** — no features are locked behind a paywall.

If you find it useful, you can support development:

[![Sponsor on GitHub](https://img.shields.io/badge/Sponsor-GitHub-ea4aaa?logo=github)](https://github.com/sponsors/KevinTCoughlin)
[![Support on Ko-fi](https://img.shields.io/badge/Support-Ko--fi-FF5E5B?logo=ko-fi)](https://ko-fi.com/kevintcoughlin)

You can also purchase a license — licensed users get clean output without nag messages and a "Licensed to [name]" badge.

```bash
mc-dad-server activate-license --key YOUR-LICENSE-KEY
```

See [docs/license.md](docs/license.md) for license management documentation.


## Multiplayer Without Port Forwarding

The installer optionally sets up [playit.gg](https://playit.gg), which creates a tunnel so your kids' friends can connect without you touching your router.

1. Run `playit` after install
2. Claim your agent at the link shown
3. Create a Minecraft Java tunnel -> `localhost:25565`
4. Share the address with friends' parents

No port forwarding. No exposing your home IP. Easy.

## Building from Source

**Quickest path:** Open in [GitHub Codespaces](https://github.com/codespaces) — the dev container includes Go 1.24, Java 21, Kotlin, and all dev tools pre-configured.

**Local prerequisites:** Go 1.24+, [just](https://github.com/casey/just#installation), and optionally Java 21+ and Kotlin for plugin work. Run `just tools` to install the Go dev tools (golangci-lint, gofumpt, goreleaser).

```bash
git clone https://github.com/KevinTCoughlin/mc-dad-server.git
cd mc-dad-server
just tools       # install dev tools
just build       # build binary
just test        # run tests
just check       # fmt + vet + lint + test
just build-all   # cross-compile all targets (go build matrix)

# Container
just container-build   # build container image
just container-up      # start with Podman Compose
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full development guide.

## FAQ

**Q: How much RAM do I need?**
2GB is fine for 2-5 players. 4GB for 5-10. 8GB if the kids go crazy with mods.

**Q: G1GC or ZGC?**
G1GC (default) is battle-tested with Aikar's flags. ZGC is newer, lower latency, great for Java 21+ servers with 8GB+ RAM. Try `--gc zgc` if you want to experiment.

**Q: Can my kids on iPad/Switch join?**
Yes! Geyser + Floodgate is installed by default. Bedrock players connect on port 19132. No Java account needed.

**Q: Can my kids' friends connect from their house?**
Yes! Use playit.gg (included) or forward port 25565 on your router. For Bedrock friends, also forward 19132/udp.

**Q: What's Paper MC?**
It's Minecraft but faster. Same game, better performance, supports plugins.

**Q: Can I add mods?**
Use `--type fabric` for mods (plugins won't install). Paper supports plugins (different ecosystem).

**Q: Is this safe?**
Whitelist is on by default — only players you approve can join. Online mode verifies real Minecraft accounts.

**Q: Can I run this in Docker/Podman?**
Yes! Use `podman compose up -d` (or `docker compose up -d`). The CLI auto-detects container mode, or force it with `--mode container`. See [Container Deployment](#container-deployment).

**Q: What if my server crashes?**
Systemd will auto-restart it. Backups run daily at 4 AM. In container mode, Podman/Docker restarts the container automatically.

**Q: What Java does it install?**
Adoptium Temurin 25 — open source, production-ready, no Oracle licensing nonsense.

## Uninstall

```bash
# Stop the server
mc-dad-server stop

# Remove systemd service (Linux)
sudo systemctl disable minecraft
sudo rm /etc/systemd/system/minecraft.service
sudo systemctl daemon-reload

# Remove files
rm -rf ~/minecraft-server

# Remove cron backup
crontab -e  # delete the mc-dad-server line
```

## License

MIT — do whatever you want with it.

## Contributing

PRs welcome! This is for dads, by dads. Keep it simple.
