# MC Dad Server

**Minecraft server in 60 seconds — for busy dads who just want their kids to play.**

No Docker. No Kubernetes. No nonsense. Just one command and you're hosting Minecraft.

## Quick Start

> **Security note:** Always [review a script](https://raw.githubusercontent.com/KevinTCoughlin/mc-dad-server/main/install.sh) before piping it to bash. You can also download it first with `curl -fsSL ... -o install.sh` and inspect it.

```bash
curl -fsSL https://raw.githubusercontent.com/KevinTCoughlin/mc-dad-server/main/install.sh | bash
```

That's it. Your server is running.

## What It Does

- ✅ Installs Java automatically (detects your OS)
- ✅ Downloads Paper MC (optimized, fast, plugin-ready)
- ✅ Pre-configures kid-safe defaults (whitelist on, PVP off, easy difficulty)
- ✅ Creates start/stop/restart/backup scripts
- ✅ Sets up auto-start on boot (systemd or launchd)
- ✅ Daily automatic backups with rotation
- ✅ Optional playit.gg tunnel — **no port forwarding needed**

## Works On

| Platform | Status |
|----------|--------|
| Ubuntu/Debian | ✅ Fully supported |
| Fedora/RHEL | ✅ Fully supported |
| Arch Linux | ✅ Fully supported |
| macOS | ✅ Fully supported |
| WSL2 (Windows) | ✅ Fully supported |
| Raspberry Pi | ✅ Works (use `--memory 1G`) |

## Options

```bash
bash install.sh \
  --edition java \
  --type paper \
  --memory 4G \
  --port 25565 \
  --difficulty normal \
  --gamemode survival \
  --players 10 \
  --motd "Our Family Server"
```

| Flag | Default | Description |
|------|---------|-------------|
| `--edition` | `java` | `java` or `bedrock` |
| `--type` | `paper` | `paper`, `fabric`, or `vanilla` |
| `--dir` | `~/minecraft-server` | Install location |
| `--port` | `25565` | Server port |
| `--memory` | `2G` | RAM allocation |
| `--difficulty` | `easy` | `peaceful`, `easy`, `normal`, `hard` |
| `--gamemode` | `survival` | `survival`, `creative`, `adventure` |
| `--players` | `10` | Max player count |
| `--motd` | `Dads Minecraft Server` | Message of the day |
| `--no-playit` | *(enabled)* | Skip playit.gg tunnel |
| `--license` | *(none)* | Dad Pack license key |
| `--version` | `latest` | Minecraft version |

## Daily Commands

```bash
# Start the server (runs in background)
bash ~/minecraft-server/run.sh

# Check if it's running
bash ~/minecraft-server/status.sh

# Add your kid to the whitelist
bash ~/minecraft-server/whitelist-add.sh KidUsername

# View the server console
screen -r minecraft
# (Press Ctrl+A then D to detach)

# Stop the server
bash ~/minecraft-server/stop.sh

# Manual backup
bash ~/minecraft-server/backup.sh
```

## Dad Pack (Coming Soon)

The installer works great for free. The **Dad Pack** will add:

- 🛡️ **GriefPrevention** — auto-configured so kids' builds are protected
- 🗺️ **Dynmap** — web-based live map (show kids their world on a tablet)
- 🔒 **Optimized configs** — server.properties and JVM flags tuned by experts
- 📊 **Web dashboard** — simple status page you can bookmark
- 🎨 **Texture pack** — optional kid-friendly texture pack
- 📖 **Dad's Guide PDF** — non-technical guide to being a Minecraft server admin

Star this repo to get notified when the Dad Pack launches.

## Multiplayer Without Port Forwarding

The installer optionally sets up [playit.gg](https://playit.gg), which creates a tunnel so your kids' friends can connect without you touching your router.

1. Run `playit` after install
2. Claim your agent at the link shown
3. Create a Minecraft Java tunnel → `localhost:25565`
4. Share the address with friends' parents

No port forwarding. No exposing your home IP. Easy.

## FAQ

**Q: How much RAM do I need?**
2GB is fine for 2-5 players. 4GB for 5-10. 8GB if the kids go crazy with mods.

**Q: Can my kids' friends connect from their house?**
Yes! Use playit.gg (included) or forward port 25565 on your router.

**Q: What's Paper MC?**
It's Minecraft but faster. Same game, better performance, supports plugins.

**Q: Can I add mods?**
Use `--type fabric` for mods. Paper supports plugins (different ecosystem).

**Q: Is this safe?**
Whitelist is on by default — only players you approve can join. Online mode verifies real Minecraft accounts.

**Q: What if my server crashes?**
Systemd will auto-restart it. Backups run daily at 4 AM.

## Uninstall

```bash
# Stop the server
bash ~/minecraft-server/stop.sh

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
