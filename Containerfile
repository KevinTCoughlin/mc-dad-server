# MC Dad Server — Multi-stage Minecraft Paper Server Build
# Builder: Debian Trixie slim (curl/jq only)
# Runtime: Eclipse Temurin 25 JRE on Alpine Linux
# https://github.com/KevinTCoughlin/mc-dad-server

# Pinned versions — update these to bump components.
# Paper stays on 26.3. Minecraft 26.4 snapshot 5 changed the login protocol
# and is not selected, even when MC_VERSION=latest.
# WorldEdit 7.4.6, Multiverse-Core 5.8.2, and Geyser 2.12.1 support Paper 26.3.
# WorldEdit 7.4.5 and Geyser 2.11.3 do not start on that line.
ARG MC_VERSION=26.3

# ---------------------------------------------------------------------------
# Stage 1: Builder — Downloads Paper JAR + plugins
# ---------------------------------------------------------------------------
FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd AS builder

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# hadolint ignore=DL3008
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        jq \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /minecraft

# Plugin pins also invalidate the download layer when upstream releases change.
ARG GEYSER_VERSION=2.12.1
ARG GEYSER_BUILD=1280
ARG FLOODGATE_VERSION=2.2.5
ARG FLOODGATE_BUILD=141
ARG PARKOUR_RELEASE_TAG=Parkour-7.2.8-RELEASE.136
ARG MULTIVERSE_VERSION=5.8.2
ARG WORLDEDIT_VERSION=7.4.6

# Download plugins sequentially with SHA-256 verification where available.
# GeyserMC/Floodgate hashes come from their builds API; Hangar provides SHA-256
# via fileInfo; GitHub (Parkour) has no hash so we verify file size instead.
#
# NOTE: Geyser/Floodgate snapshot builds may have compatibility issues with
# certain Paper versions. If AppCDS training fails (plugin crash),
# entrypoint.sh will start the server without the JVM warmup cache.
# hadolint ignore=SC2016,SC2317
RUN validate_sha256() { \
        echo "$1" | grep -qE '^[a-f0-9]{64}$' || \
            { echo "Invalid SHA-256 for $2: $1"; exit 1; }; \
    } && \
    set -e && mkdir -p plugins && \
    # Geyser — metadata and JAR use the same pinned build.
    GEYSER_META=$(curl -fsSL "https://download.geysermc.org/v2/projects/geyser/versions/${GEYSER_VERSION}/builds/${GEYSER_BUILD}") && \
    GEYSER_SHA256=$(echo "$GEYSER_META" | jq -r '.downloads.spigot.sha256') && \
    echo "Downloading Geyser build ${GEYSER_BUILD}" && \
    validate_sha256 "$GEYSER_SHA256" "Geyser" && \
    curl -fsSL -o plugins/Geyser-Spigot.jar \
      "https://download.geysermc.org/v2/projects/geyser/versions/${GEYSER_VERSION}/builds/${GEYSER_BUILD}/downloads/spigot" && \
    echo "${GEYSER_SHA256}  plugins/Geyser-Spigot.jar" | sha256sum -c - && \
    # Floodgate — metadata and JAR use the same pinned build.
    FLOODGATE_META=$(curl -fsSL "https://download.geysermc.org/v2/projects/floodgate/versions/${FLOODGATE_VERSION}/builds/${FLOODGATE_BUILD}") && \
    FLOODGATE_SHA256=$(echo "$FLOODGATE_META" | jq -r '.downloads.spigot.sha256') && \
    echo "Downloading Floodgate build ${FLOODGATE_BUILD}" && \
    validate_sha256 "$FLOODGATE_SHA256" "Floodgate" && \
    curl -fsSL -o plugins/Floodgate-Spigot.jar \
      "https://download.geysermc.org/v2/projects/floodgate/versions/${FLOODGATE_VERSION}/builds/${FLOODGATE_BUILD}/downloads/spigot" && \
    echo "${FLOODGATE_SHA256}  plugins/Floodgate-Spigot.jar" | sha256sum -c - && \
    # Parkour — GitHub releases API does not provide SHA-256; verify file size
    PARKOUR_RELEASE=$(curl -fsSL "https://api.github.com/repos/A5H73Y/Parkour/releases/tags/${PARKOUR_RELEASE_TAG}") && \
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
    MV_SHA256=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/versions/${MULTIVERSE_VERSION}" \
      | jq -r '.downloads.PAPER.fileInfo.sha256Hash') && \
    validate_sha256 "$MV_SHA256" "Multiverse-Core" && \
    curl -fsSL -o plugins/Multiverse-Core.jar \
      "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/versions/${MULTIVERSE_VERSION}/PAPER/download" && \
    echo "${MV_SHA256}  plugins/Multiverse-Core.jar" | sha256sum -c - && \
    # WorldEdit — SHA-256 from Hangar API
    WE_SHA256=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/WorldEdit/versions/${WORLDEDIT_VERSION}" \
      | jq -r '.downloads.PAPER.fileInfo.sha256Hash') && \
    validate_sha256 "$WE_SHA256" "WorldEdit" && \
    curl -fsSL -o plugins/WorldEdit.jar \
      "https://hangar.papermc.io/api/v1/projects/WorldEdit/versions/${WORLDEDIT_VERSION}/PAPER/download" && \
    echo "${WE_SHA256}  plugins/WorldEdit.jar" | sha256sum -c - && \
    ls -la plugins/*.jar && \
    echo "All plugins downloaded and verified"

# ARG placed here so MC_VERSION changes only bust the Paper download layer.
# PAPER_MIN_BUILD is the first 26.3 build that contains the fire, splash, and
# respawn fixes (builds 228, 229, and 233). Keep this in sync with
# server.MinPaperBuild.
ARG MC_VERSION
ARG PAPER_MIN_BUILD=233

# Download Paper server JAR via PaperMC Fill v3 API with SHA-256 verification.
# "latest" means the newest stable release at or below 26.3. Numeric 26.4 ids,
# snapshots, previews, and release candidates are refused. 26.3 builds older
# than PAPER_MIN_BUILD are refused. Fill v3 requires a User-Agent header.
# hadolint ignore=SC2016
RUN set -euo pipefail && \
    UA="mc-dad-server (https://github.com/KevinTCoughlin/mc-dad-server)" && \
    SUPPORTED="26.3" && \
    MIN_BUILD="${PAPER_MIN_BUILD}" && \
    version_newer() { \
        local a="$1" b="$2" i n av bv; \
        local -a aa bb; \
        IFS='.' read -r -a aa <<< "$a"; \
        IFS='.' read -r -a bb <<< "$b"; \
        n=${#aa[@]}; \
        if [ "${#bb[@]}" -gt "$n" ]; then n=${#bb[@]}; fi; \
        for ((i = 0; i < n; i++)); do \
            av=${aa[i]:-0}; \
            bv=${bb[i]:-0}; \
            if [ "$av" -gt "$bv" ]; then return 0; fi; \
            if [ "$av" -lt "$bv" ]; then return 1; fi; \
        done; \
        return 1; \
    } && \
    MC_VER="${MC_VERSION}" && \
    if [ "$MC_VER" = "latest" ] || [ -z "$MC_VER" ]; then \
        VERSIONS=$(curl -fsSL -H "User-Agent: ${UA}" \
            "https://fill.papermc.io/v3/projects/paper/versions") && \
        MC_VER="" && \
        while IFS= read -r candidate; do \
            [ -n "$candidate" ] || continue; \
            if version_newer "$candidate" "$SUPPORTED"; then continue; fi; \
            if [ -z "$MC_VER" ] || version_newer "$candidate" "$MC_VER"; then \
                MC_VER="$candidate"; \
            fi; \
        done < <(printf '%s\n' "$VERSIONS" | jq -r '.versions[].version.id | select(test("^[0-9]+(\\.[0-9]+){1,2}$"))'); \
        [ -n "$MC_VER" ] || { echo "No stable Paper version at or below ${SUPPORTED}"; exit 1; }; \
    fi && \
    printf '%s\n' "$MC_VER" | grep -qE '^[0-9]+(\.[0-9]+){1,2}$' || { \
        echo "Refusing Paper version ${MC_VER}: snapshots, previews, and release candidates are not installed"; \
        exit 1; \
    } && \
    if version_newer "$MC_VER" "$SUPPORTED"; then \
        echo "Refusing Paper ${MC_VER}: newest supported release is ${SUPPORTED}"; \
        exit 1; \
    fi && \
    BUILD_JSON=$(curl -fsSL -H "User-Agent: ${UA}" \
        "https://fill.papermc.io/v3/projects/paper/versions/${MC_VER}/builds/latest") && \
    BUILD_ID=$(echo "$BUILD_JSON" | jq -r '.id') && \
    case "$MC_VER" in \
        26.3|26.3.0) \
            printf '%s\n' "$BUILD_ID" | grep -qE '^[0-9]+$' || { \
                echo "Paper ${MC_VER} response has no build id"; exit 1; \
            }; \
            if [ "$BUILD_ID" -lt "$MIN_BUILD" ]; then \
                echo "Refusing Paper ${MC_VER} build ${BUILD_ID}: need build ${MIN_BUILD} or newer"; \
                exit 1; \
            fi ;; \
    esac && \
    JAR_NAME=$(echo "$BUILD_JSON" | jq -r '.downloads["server:default"].name') && \
    EXPECTED_SHA256=$(echo "$BUILD_JSON" | jq -r '.downloads["server:default"].checksums.sha256') && \
    DOWNLOAD_URL=$(echo "$BUILD_JSON" | jq -r '.downloads["server:default"].url') && \
    echo "$EXPECTED_SHA256" | grep -qE '^[a-f0-9]{64}$' || \
        { echo "Invalid SHA-256 for Paper: ${EXPECTED_SHA256}"; exit 1; } && \
    curl -fsSL -H "User-Agent: ${UA}" -o server.jar "$DOWNLOAD_URL" && \
    echo "${EXPECTED_SHA256}  server.jar" | sha256sum -c - && \
    echo "Downloaded and verified Paper ${MC_VER} build ${BUILD_ID}: ${JAR_NAME}"

# Accept EULA
RUN echo "eula=true" > eula.txt

# ---------------------------------------------------------------------------
# Stage 2: Runtime — Eclipse Temurin 25 JRE on Alpine Linux.
# Paper 26.3 requires Java 25. Do not switch this base to Java 21.
# ---------------------------------------------------------------------------
FROM eclipse-temurin:25-jre-alpine@sha256:28db6fdf60e38945e43d840c0333aeaec66c15943070104f7586fd3c9d1665b0 AS runtime

# Install bash (required by entrypoint.sh for arrays and parameter expansion)
# hadolint ignore=DL3018
RUN apk add --no-cache bash && \
    rm -rf /tmp/* /var/tmp/*

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# Non-root user
RUN addgroup -g 1000 -S minecraft && \
    adduser -u 1000 -D -S -G minecraft -s /sbin/nologin minecraft

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
      org.opencontainers.image.description="Containerized Minecraft Paper 26.3 server on Eclipse Temurin 25, with Geyser cross-play, Parkour, and tuned configs" \
      org.opencontainers.image.base.name="eclipse-temurin:25-jre-alpine" \
      org.opencontainers.image.source="https://github.com/KevinTCoughlin/mc-dad-server" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="KevinTCoughlin"

# Remove setuid/setgid bits after all COPY steps to cover files from builder stage
RUN find / -xdev -perm /6000 -type f -exec chmod a-s {} + 2>/dev/null || true

USER 1000:1000
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
        until grep -qE "INFO\]: Done \([0-9.]+s\)!" logs/latest.log 2>/dev/null; do \
            kill -0 $PID 2>/dev/null || { echo "Server process died during AppCDS training"; break; }; \
            sleep 2; \
        done && \
        kill $PID 2>/dev/null; wait $PID 2>/dev/null; true' || true && \
    if test -f app-cds.jsa; then \
        echo "AppCDS archive created successfully"; \
    else \
        echo "WARNING: AppCDS archive not created. This may be due to plugin compatibility issues with the selected Paper/MC version."; \
        echo "The container will start without JVM warmup — functionality is unaffected."; \
        echo "Check logs above for plugin startup errors, or see: https://github.com/GeyserMC/Geyser/issues/6297"; \
    fi && \
    rm -rf world world_nether world_the_end logs cache version_history.json && \
    rm -rf plugins && mv plugins.pristine plugins

# Java (25565/tcp), Bedrock/Geyser (19132/udp), RCON (25575/tcp)
EXPOSE 25565/tcp 19132/udp 25575/tcp

HEALTHCHECK --interval=30s --timeout=5s --start-period=90s --retries=3 \
    CMD ["bash", "-c", "cat /proc/*/cmdline 2>/dev/null | tr '\\0' '\\n' | grep -q server.jar || exit 1"]

ENTRYPOINT ["/entrypoint.sh"]
