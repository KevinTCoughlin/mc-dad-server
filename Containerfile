# MC Dad Server — Multi-stage Minecraft Paper Server Build
# Builder: Debian Trixie slim (curl/jq only)
# Runtime: Eclipse Temurin 25 JRE on Alpine Linux
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
# GeyserMC/Floodgate hashes come from their builds API; Hangar provides SHA-256
# via fileInfo; GitHub (Parkour) has no hash so we verify file size instead.
# hadolint ignore=SC2016,SC2317
RUN validate_sha256() { \
        echo "$1" | grep -qE '^[a-f0-9]{64}$' || \
            { echo "Invalid SHA-256 for $2: $1"; exit 1; }; \
    } && \
    set -e && mkdir -p plugins && \
    # Geyser — SHA-256 from GeyserMC build API
    GEYSER_META=$(curl -fsSL "https://download.geysermc.org/v2/projects/geyser/versions/latest/builds/latest") && \
    GEYSER_SHA256=$(echo "$GEYSER_META" | jq -r '.downloads.spigot.sha256') && \
    validate_sha256 "$GEYSER_SHA256" "Geyser" && \
    curl -fsSL -o plugins/Geyser-Spigot.jar \
      "https://download.geysermc.org/v2/projects/geyser/versions/latest/builds/latest/downloads/spigot" && \
    echo "${GEYSER_SHA256}  plugins/Geyser-Spigot.jar" | sha256sum -c - && \
    # Floodgate — SHA-256 from GeyserMC build API
    FLOODGATE_META=$(curl -fsSL "https://download.geysermc.org/v2/projects/floodgate/versions/latest/builds/latest") && \
    FLOODGATE_SHA256=$(echo "$FLOODGATE_META" | jq -r '.downloads.spigot.sha256') && \
    validate_sha256 "$FLOODGATE_SHA256" "Floodgate" && \
    curl -fsSL -o plugins/Floodgate-Spigot.jar \
      "https://download.geysermc.org/v2/projects/floodgate/versions/latest/builds/latest/downloads/spigot" && \
    echo "${FLOODGATE_SHA256}  plugins/Floodgate-Spigot.jar" | sha256sum -c - && \
    # Parkour — GitHub releases API does not provide SHA-256; verify file size
    PARKOUR_RELEASE=$(curl -fsSL "https://api.github.com/repos/A5H73Y/Parkour/releases/latest") && \
    PARKOUR_URL=$(echo "$PARKOUR_RELEASE" | jq -r '[.assets[] | select(.name | endswith(".jar"))][0].browser_download_url') && \
    PARKOUR_EXPECTED_SIZE=$(echo "$PARKOUR_RELEASE" | jq -r '[.assets[] | select(.name | endswith(".jar"))][0].size') && \
    if [ -z "$PARKOUR_URL" ] || [ "$PARKOUR_URL" = "null" ] || \
       [ -z "$PARKOUR_EXPECTED_SIZE" ] || [ "$PARKOUR_EXPECTED_SIZE" = "null" ]; then \
        echo "Failed to resolve Parkour JAR asset from GitHub releases API"; exit 1; \
    fi && \
    curl -fsSL -o plugins/Parkour.jar "$PARKOUR_URL" && \
    PARKOUR_ACTUAL_SIZE=$(stat -c%s plugins/Parkour.jar) && \
    [ "$PARKOUR_ACTUAL_SIZE" = "$PARKOUR_EXPECTED_SIZE" ] || \
        { echo "Parkour size mismatch: expected ${PARKOUR_EXPECTED_SIZE}, got ${PARKOUR_ACTUAL_SIZE}"; exit 1; } && \
    echo "Parkour SHA-256: $(sha256sum plugins/Parkour.jar)" && \
    # Multiverse-Core — SHA-256 from Hangar API
    MV_VERSION=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/latestrelease" \
      | tr -d '"') && \
    MV_SHA256=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/versions/${MV_VERSION}" \
      | jq -r '.downloads.PAPER.fileInfo.sha256Hash') && \
    validate_sha256 "$MV_SHA256" "Multiverse-Core" && \
    curl -fsSL -o plugins/Multiverse-Core.jar \
      "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/versions/${MV_VERSION}/PAPER/download" && \
    echo "${MV_SHA256}  plugins/Multiverse-Core.jar" | sha256sum -c - && \
    # WorldEdit — SHA-256 from Hangar API
    WE_VERSION=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/WorldEdit/latestrelease" \
      | tr -d '"') && \
    WE_SHA256=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/WorldEdit/versions/${WE_VERSION}" \
      | jq -r '.downloads.PAPER.fileInfo.sha256Hash') && \
    validate_sha256 "$WE_SHA256" "WorldEdit" && \
    curl -fsSL -o plugins/WorldEdit.jar \
      "https://hangar.papermc.io/api/v1/projects/WorldEdit/versions/${WE_VERSION}/PAPER/download" && \
    echo "${WE_SHA256}  plugins/WorldEdit.jar" | sha256sum -c - && \
    ls -la plugins/*.jar && \
    echo "All plugins downloaded and verified"

# ARG placed here so MC_VERSION changes only bust the Paper download layer
ARG MC_VERSION

# Download Paper server JAR via PaperMC Fill v3 API with SHA-256 verification
# Fill v3 requires a descriptive User-Agent header on every request.
RUN set -e && \
    UA="mc-dad-server (https://github.com/KevinTCoughlin/mc-dad-server)" && \
    MC_VER="${MC_VERSION}" && \
    if [ "$MC_VER" = "latest" ] || [ -z "$MC_VER" ]; then \
        MC_VER=$(curl -fsSL -H "User-Agent: ${UA}" \
            "https://fill.papermc.io/v3/projects/paper/versions" \
            | jq -r '.versions[0].version.id'); \
    fi && \
    BUILD_JSON=$(curl -fsSL -H "User-Agent: ${UA}" \
        "https://fill.papermc.io/v3/projects/paper/versions/${MC_VER}/builds/latest") && \
    JAR_NAME=$(echo "$BUILD_JSON" | jq -r '.downloads["server:default"].name') && \
    EXPECTED_SHA256=$(echo "$BUILD_JSON" | jq -r '.downloads["server:default"].checksums.sha256') && \
    DOWNLOAD_URL=$(echo "$BUILD_JSON" | jq -r '.downloads["server:default"].url') && \
    echo "$EXPECTED_SHA256" | grep -qE '^[a-f0-9]{64}$' || \
        { echo "Invalid SHA-256 for Paper: ${EXPECTED_SHA256}"; exit 1; } && \
    curl -fsSL -H "User-Agent: ${UA}" -o server.jar "$DOWNLOAD_URL" && \
    echo "${EXPECTED_SHA256}  server.jar" | sha256sum -c - && \
    echo "Downloaded and verified Paper ${MC_VER}: ${JAR_NAME}"

# Accept EULA
RUN echo "eula=true" > eula.txt

# ---------------------------------------------------------------------------
# Stage 2: Runtime — Eclipse Temurin 25 JRE on Alpine Linux
# ---------------------------------------------------------------------------
FROM eclipse-temurin:25-jre-alpine@sha256:f10d6259d0798c1e12179b6bf3b63cea0d6843f7b09c9f9c9c422c50e44379ec AS runtime

# Install bash (required by entrypoint.sh for arrays and parameter expansion)
# hadolint ignore=DL3018
RUN apk add --no-cache bash && \
    rm -rf /tmp/* /var/tmp/* && \
    (find / -xdev -perm /6000 -type f -exec chmod a-s {} + 2>/dev/null || true)

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Non-root user
RUN adduser -D -S -s /sbin/nologin minecraft

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
# hadolint ignore=SC2016
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
