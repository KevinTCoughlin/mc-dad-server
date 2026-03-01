# MC Dad Server — Multi-stage Minecraft Paper Server Build
# Builder: Debian Trixie slim (curl/jq only)
# Runtime: Eclipse Temurin 25 JRE on Ubuntu Noble
# https://github.com/KevinTCoughlin/mc-dad-server

# Pinned versions — update these to bump components
ARG MC_VERSION=latest

# ---------------------------------------------------------------------------
# Stage 1: Builder — Downloads Paper JAR + plugins
# ---------------------------------------------------------------------------
FROM debian:trixie-slim@sha256:1d3c811171a08a5adaa4a163fbafd96b61b87aa871bbc7aa15431ac275d3d430 AS builder

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# hadolint ignore=DL3008
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        jq \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /minecraft

# Download plugins sequentially with SHA-256 verification where available.
# GeyserMC/Floodgate hashes come from their builds API; Hangar/GitHub verify via HTTPS.
# hadolint ignore=SC2016
RUN set -e && mkdir -p plugins && \
    GEYSER_DATA=$(curl -fsSL "https://download.geysermc.org/v2/projects/geyser/versions/latest/builds/latest") && \
    GEYSER_SHA=$(echo "$GEYSER_DATA" | jq -r '.downloads.spigot.sha256') && \
    curl -fsSL -o plugins/Geyser-Spigot.jar \
      "https://download.geysermc.org/v2/projects/geyser/versions/latest/builds/latest/downloads/spigot" && \
    echo "${GEYSER_SHA}  plugins/Geyser-Spigot.jar" | sha256sum -c && \
    FLOOD_DATA=$(curl -fsSL "https://download.geysermc.org/v2/projects/floodgate/versions/latest/builds/latest") && \
    FLOOD_SHA=$(echo "$FLOOD_DATA" | jq -r '.downloads.spigot.sha256') && \
    curl -fsSL -o plugins/Floodgate-Spigot.jar \
      "https://download.geysermc.org/v2/projects/floodgate/versions/latest/builds/latest/downloads/spigot" && \
    echo "${FLOOD_SHA}  plugins/Floodgate-Spigot.jar" | sha256sum -c && \
    PARKOUR_URL=$(curl -fsSL "https://api.github.com/repos/A5H73Y/Parkour/releases/latest" \
      | jq -r '.assets[0].browser_download_url') && \
    curl -fsSL -o plugins/Parkour.jar "$PARKOUR_URL" && \
    MV_VERSION=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/latestrelease" \
      | tr -d '"') && \
    curl -fsSL -o plugins/Multiverse-Core.jar \
      "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/versions/${MV_VERSION}/PAPER/download" && \
    WE_VERSION=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/WorldEdit/latestrelease" \
      | tr -d '"') && \
    curl -fsSL -o plugins/WorldEdit.jar \
      "https://hangar.papermc.io/api/v1/projects/WorldEdit/versions/${WE_VERSION}/PAPER/download" && \
    ls -la plugins/*.jar && \
    echo "All plugins downloaded"

# ARG placed here so MC_VERSION changes only bust the Paper download layer
ARG MC_VERSION

# Download Paper server JAR via PaperMC API with SHA-256 verification
RUN set -e && \
    MC_VER="${MC_VERSION}" && \
    if [ "$MC_VER" = "latest" ] || [ -z "$MC_VER" ]; then \
        MC_VER=$(curl -fsSL https://api.papermc.io/v2/projects/paper | jq -r '.versions[-1]'); \
    fi && \
    BUILD_DATA=$(curl -fsSL "https://api.papermc.io/v2/projects/paper/versions/${MC_VER}/builds") && \
    LATEST_BUILD=$(echo "$BUILD_DATA" | jq -r '.builds[-1].build') && \
    JAR_NAME=$(echo "$BUILD_DATA" | jq -r '.builds[-1].downloads.application.name') && \
    JAR_SHA=$(echo "$BUILD_DATA" | jq -r '.builds[-1].downloads.application.sha256') && \
    curl -fsSL -o server.jar \
        "https://api.papermc.io/v2/projects/paper/versions/${MC_VER}/builds/${LATEST_BUILD}/downloads/${JAR_NAME}" && \
    echo "${JAR_SHA}  server.jar" | sha256sum -c && \
    echo "Downloaded and verified Paper ${MC_VER} build ${LATEST_BUILD}"

# Accept EULA
RUN echo "eula=true" > eula.txt

# ---------------------------------------------------------------------------
# Stage 2: Runtime — Eclipse Temurin 25 JRE on Ubuntu Noble
# ---------------------------------------------------------------------------
FROM eclipse-temurin:25-jre-noble@sha256:e7e348559e36c85a3fe868d7c298517b7cc75f01b34ce3813798a5cd781795f1 AS runtime

# Non-root user
RUN useradd --no-log-init -r -m -s /usr/sbin/nologin minecraft

# Copy server files from builder
COPY --from=builder --chown=minecraft:minecraft /minecraft /minecraft
COPY --chmod=755 entrypoint.sh /entrypoint.sh

# Copy config files into image (avoids rootless podman bind-mount permission issues)
COPY --chown=minecraft:minecraft configs/server.properties /minecraft/server.properties
COPY --chown=minecraft:minecraft configs/bukkit.yml /minecraft/bukkit.yml
COPY --chown=minecraft:minecraft configs/spigot.yml /minecraft/spigot.yml
COPY --chown=minecraft:minecraft configs/paper-global.yml /minecraft/config/paper-global.yml
COPY --chown=minecraft:minecraft configs/paper-world-defaults.yml /minecraft/config/paper-world-defaults.yml

LABEL org.opencontainers.image.title="MC Dad Server" \
      org.opencontainers.image.description="Containerized Minecraft Paper server with Geyser cross-play, Parkour, and tuned configs" \
      org.opencontainers.image.source="https://github.com/KevinTCoughlin/mc-dad-server" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="KevinTCoughlin"

USER minecraft
WORKDIR /minecraft

# ---------------------------------------------------------------------------
# AppCDS training — boot server once to pre-cache class metadata.
# Uses -XX:ArchiveClassesAtExit to dump a shared archive on JVM exit.
# Speeds up subsequent container starts by ~30-40% (skips class loading).
# Plugins are backed up before training and restored after because some
# plugins (Geyser) auto-update on first boot, and killing the server
# mid-download truncates JARs.
# ---------------------------------------------------------------------------
RUN cp -a plugins plugins.pristine && \
    sed -i \
        -e 's/%%MC_DIFFICULTY%%/normal/' \
        -e 's/%%MC_GAMEMODE%%/survival/' \
        -e 's/%%MC_MAX_PLAYERS%%/20/' \
        -e 's|%%MC_MOTD%%|AppCDS Training|' \
        -e 's/%%MC_PORT%%/25565/g' \
        -e 's/%%MC_RCON_PASSWORD%%/training/' \
        -e 's/%%MC_WHITELIST%%/false/' \
        server.properties && \
    timeout 180 bash -c ' \
        java -XX:ArchiveClassesAtExit=app-cds.jsa \
            -Xms512M -Xmx512M -XX:+UseG1GC \
            -jar server.jar nogui & \
        PID=$! && \
        until grep -q "Done" logs/latest.log 2>/dev/null; do sleep 2; done && \
        kill $PID && wait $PID' || true && \
    test -f app-cds.jsa && echo "AppCDS archive created successfully" && \
    rm -rf world world_nether world_the_end logs cache version_history.json && \
    rm -rf plugins && mv plugins.pristine plugins

# Java (25565/tcp), Bedrock/Geyser (19132/udp), RCON (25575/tcp)
EXPOSE 25565/tcp 19132/udp 25575/tcp

HEALTHCHECK --interval=30s --timeout=5s --start-period=90s --retries=3 \
    CMD cat /proc/*/cmdline 2>/dev/null | tr '\0' '\n' | grep -q server.jar || exit 1

ENTRYPOINT ["/entrypoint.sh"]
