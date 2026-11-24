# Changelog

## v0.5.0

- `install --memory` defaults to `auto`. Headroom is 25% of RAM, clamped to 1–4 GiB, and the heap is clamped to 512 MiB–12 GiB. An explicit `--memory 4G` still wins.
- Paper builds with no SHA-256 in the PaperMC API are refused. Install preflight checks Java 25, a free TCP port, at least 2 GiB free, and the Minecraft EULA. The systemd unit runs as a dedicated user with `NoNewPrivileges`, `ProtectSystem=strict`, `PrivateTmp`, and `Restart=on-failure`.
- `setup-container` and `generate-compose` default `--memory` to `auto`. The container memory limit uses the same reservation and clamps. The JVM heap is 75% of that limit (`MEMORY=75%`, which becomes `-XX:InitialRAMPercentage` and `-XX:MaxRAMPercentage`, the same form itzg/minecraft-server accepts). `--memory 4G` keeps a fixed heap and a container limit 1 GiB higher.
- Fabric installer jars are verified with the strongest checksum Maven publishes beside the jar: SHA-512, then SHA-256, then SHA-1. `meta.fabricmc.net` does not include a hash. MD5 is ignored. If none of those sidecars is published, the download warns and continues instead of using a hash pinned in this repo, because the installer version comes from the live meta API.

## v0.4.1

- Pin Geyser 2.12.1 build 1307 and Paper 26.3 build 258.
- Dependency updates for `golang.org/x/term` and the container images.

Earlier releases are on the [GitHub releases page](https://github.com/KevinTCoughlin/mc-dad-server/releases).
