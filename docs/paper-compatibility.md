# Paper compatibility

New installs use Paper **26.3**, and they refuse a 26.3 build older than **233**. `latest` means the newest stable Paper release that is not newer than 26.3. Snapshots, previews, release candidates, and Minecraft 26.4 are refused by the CLI and by the Containerfile. This project does not migrate to 26.4.

Checked against the PaperMC Fill v3 builds API (`https://fill.papermc.io/v3/projects/paper/versions/26.3/builds`). On 1 November 2026 the newest 26.3 build was 236. On 2 November 2026 `builds/latest` resolved to `paper-26.3-237.jar`. Both are above the floor of 233, so both include the three fixes below. The installer keeps taking the newest build on the 26.3 line.

| Build | Commit | What Paper changed |
|-------|--------|--------------------|
| 228 | `2e549d651` | Call `EntityDamageByBlockEvent` for fire damage ([#12235](https://github.com/PaperMC/Paper/pull/12235)) |
| 229 | `ff31a2c99` | A cancelled `PotionSplashEvent` no longer extinguishes fire ([#12194](https://github.com/PaperMC/Paper/pull/12194)) |
| 233 | `f728a121f` | Respawn-radius selection covers the whole square ([#14390](https://github.com/PaperMC/Paper/pull/14390), MC-302446) |

The downloads page groups several of those commits under build 235. The builds API lists one commit per build and matches the table above.

## Dragon breath

Build 228 does **not** deprecate a dragon-breath damage cause. Commit `2e549d651` only edits `BaseFireBlock` (fire damage now carries the fire block via `eventBlockDamager`) and `CraftEventFactory` (it stops forcing `DamageTypes.IN_FIRE` onto `DamageCause.FIRE`).

`EntityDamageEvent.DamageCause.DRAGON_BREATH` is already `@Deprecated(since = "1.21.5")` in the Paper 26.3 API, including builds before 228, with the note that vanilla uses `ENTITY_ATTACK` instead. `Effect.DRAGON_BREATH` is `@Deprecated(since = "26.2")`. Neither deprecation is in the 228–233 commits. This repository does not reference either constant.

## Plugin code in this repository

There is no Kotlin or Java plugin source here, and no `paper-api` dependency to bump. Parkour, WorldEdit, Multiverse-Core, Geyser, and Floodgate are downloaded JARs. ChatSentry is a deployed config; the installer does not download that JAR.

The pinned Parkour release (`Parkour-7.2.8-RELEASE.136`) was read from its tag. Its `PlayerListener` is the only damage and respawn code those JARs ship:

- `onEntityDamage` listens to `EntityDamageEvent`. `EntityDamageByBlockEvent` is a subclass, so one fire hit is delivered to that handler once. Parkour does not also listen for `EntityDamageByBlockEvent`, so the course-death call is not doubled.
- A fatal hit on a course sets damage to 0, cancels the event, and calls `playerDie` once. The player does not actually die from that hit, so `PlayerRespawnEvent` does not call `playerDie` again for the same fire tick.
- `onEntityDamageByEntity` does not see block fire.
- There is no `DamageCause.DRAGON_BREATH` use and no `PotionSplashEvent` listener. Nothing in this repo cancels splash potions, so build 229 does not require a config change.
- `onPlayerRespawn` calls `playerDie` when the player is on a course. It does not choose a bed, a respawn anchor, or a vanilla respawn-radius square. Checkpoints are Parkour's own teleport. Build 233 changes Paper's `PlayerSpawnFinder` for the vanilla no-bed, no-anchor search. Nothing here implements that search.
- Course immunity is the shipped Parkour config (`DisablePlayerDamage`, `DisableFallDamage`) for players on a course. There is no separate admin-immunity listener. Creative and spectator immunity stay vanilla, and this repo does not un-cancel those events.

No listener was added or edited. Forking Parkour to reimplement behavior it already has would not change the install.

## Bundled plugins

Container image pins, and what each one is. The CLI still resolves Geyser, Floodgate, Parkour, Multiverse-Core, and WorldEdit from each project's latest release. Those latest releases matched the pins below on 1 November 2026. When Geyser's latest build starts emulating Java 26.4, pin the CLI resolver too; a 26.4 Geyser client will not join a 26.3 server without ViaVersion, and ViaVersion is not installed.

| Plugin | Container pin | Role | 26.3 | 26.4 |
|--------|---------------|------|------|------|
| Geyser-Spigot | 2.12.1 build 1280 | Bedrock-to-Java proxy. This is the component that speaks Minecraft Java login encryption and compression. | GeyserMC documents Java 26.3. 2.11.3 fails to enable on Paper 26.3 (`CraftItemStack` reflection). | Do not move the server until a Geyser build documents snapshot 5's acknowledgement packets (`login_finished` / `login_acknowledged` and `apply_options` / `options_applied`). |
| Floodgate | 2.2.5 build 141 | Auth companion for Geyser. Not a Java packet rewriter. | Same pin is the current Floodgate release and is the one Geyser 26.3 setups use. | Update it in the same change as Geyser. |
| WorldEdit | 7.4.6 | Bukkit/Paper world editor. Not a proxy. | 7.4.6 supports Paper 1.21.4–26.3. 7.4.5 stops at 26.2 and fails plugin startup on 26.3. | 7.4.6 does not claim 26.4. |
| Multiverse-Core | 5.8.2 | World management. Not a packet library. | 5.8.2 declares Paper through 26.3. 5.8.1 stops at 26.2. | No 26.4 release. |
| Parkour | 7.2.8 `RELEASE.136` | Course plugin, Spigot API, `api-version: "1.13"`. No Paper API pin. | Newest GitHub release. Damage behavior is covered above. | Not a protocol plugin. |
| ChatSentry | config only | Chat filter config. | No JAR is installed by this repo. | No protocol code. |

Not installed: ProtocolLib, PacketEvents, ViaVersion, ViaBackwards, Velocity, BungeeCord.

RCON in this repo is the Source RCON protocol (`internal/container/rcon.go` and the Bun sidecar). It is not the Minecraft login protocol, so the 26.4 acknowledgement packets do not apply to it.

## Manual checklist

Automated tests cover version selection, the build floor, and the container pins. The items below need a live server.

### Paper 26.3

1. Install with the default version and confirm the server log is Paper 26.3 build 233 or newer.
2. Confirm `--mc-version latest` still resolves to 26.3, and that `--mc-version 26.4` and `--mc-version 26.4-snapshot-5` fail before a JAR is written.
3. Join with a Java 26.3 client, disconnect, and join again. Chat and movement work.
4. On a parkour course, with `DisablePlayerDamage: true`, stand in fire. Damage is cancelled once. A course death, if the course is set to die in fire, is recorded once.
5. Off a course, in survival, stand in fire. Health drops on each fire tick and the player dies once, not twice.
6. In creative, stand in fire. Vanilla immunity still applies.
7. Die with no bed and no respawn anchor, with the respawn radius left at its world default. The respawn position can land on the far side of that square (the build 233 fix). Parkour checkpoints are a separate teleport and are not this search.
8. Throw a water bottle onto fire. Nothing in this repo cancels `PotionSplashEvent`, so the fire goes out. A cancelling plugin would have to be added before the inverse (fire stays lit) can be tested here.
9. Bedrock client through Geyser on UDP 19132: join, disconnect, join again.
10. WorldEdit `//set`, Multiverse `/mv tp`, and Parkour `/pa join` still load.

### Later 26.4 move

Do not point this installer at 26.4 until Geyser, Floodgate, WorldEdit, and Multiverse-Core each document a compatible build, and then run:

1. Java client, `online-mode=true`, default `network-compression-threshold`. Login must finish the acknowledgement (`login_acknowledged` or `options_applied`). A miss looks like a timeout or a malformed packet immediately after login success.
2. Reconnect that same Java client three times.
3. Bedrock client through the updated Geyser: fresh login and a reconnect. Geyser has to speak the new packets; this repo does not implement them.
4. Repeat with `network-compression-threshold=-1` and with it set to `256`.
5. RCON `list` still works. That protocol did not change.
