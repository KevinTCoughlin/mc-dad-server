# MC Dad Server — Multi-stage Minecraft Paper Server Build
# Builder: Debian Trixie slim (curl/jq only)
# Runtime: Eclipse Temurin 21 JRE on Ubuntu Noble
# https://github.com/KevinTCoughlin/mc-dad-server

# Pinned versions — update these to bump components
ARG MC_VERSION=latest

# ---------------------------------------------------------------------------
# Stage 1: Builder — Downloads Paper JAR + plugins
# ---------------------------------------------------------------------------
FROM debian:trixie-slim@sha256:b29a157cc8540addda9836c23750e389693bf3b6d9a932a55504899e5601a66b AS builder

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# hadolint ignore=DL3008
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        jq \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /minecraft

# Download plugins (changes infrequently — cached layer)
# hadolint ignore=SC2015
RUN mkdir -p plugins && \
    ( curl -fsSL -o plugins/Geyser-Spigot.jar \
        "https://download.geysermc.org/v2/projects/geyser/versions/latest/builds/latest/downloads/spigot" ) & \
    ( curl -fsSL -o plugins/Floodgate-Spigot.jar \
        "https://download.geysermc.org/v2/projects/floodgate/versions/latest/builds/latest/downloads/spigot" ) & \
    ( PARKOUR_URL=$(curl -fsSL "https://api.github.com/repos/A5H73Y/Parkour/releases/latest" \
        | jq -r '.assets[0].browser_download_url') && \
      curl -fsSL -o plugins/Parkour.jar "$PARKOUR_URL" ) & \
    ( MV_VERSION=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/latestrelease" \
        | tr -d '"') && \
      curl -fsSL -o plugins/Multiverse-Core.jar \
        "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/versions/${MV_VERSION}/PAPER/download" ) & \
    ( WE_VERSION=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/WorldEdit/latestrelease" \
        | tr -d '"') && \
      curl -fsSL -o plugins/WorldEdit.jar \
        "https://hangar.papermc.io/api/v1/projects/WorldEdit/versions/${WE_VERSION}/PAPER/download" ) & \
    wait && echo "All plugins downloaded"

# ARG placed here so MC_VERSION changes only bust the Paper download layer
ARG MC_VERSION

# Download Paper server JAR via PaperMC API
RUN set -e && \
    MC_VER="${MC_VERSION}" && \
    if [ "$MC_VER" = "latest" ] || [ -z "$MC_VER" ]; then \
        MC_VER=$(curl -fsSL https://api.papermc.io/v2/projects/paper | jq -r '.versions[-1]'); \
    fi && \
    LATEST_BUILD=$(curl -fsSL "https://api.papermc.io/v2/projects/paper/versions/${MC_VER}/builds" \
        | jq -r '.builds[-1].build') && \
    JAR_NAME=$(curl -fsSL "https://api.papermc.io/v2/projects/paper/versions/${MC_VER}/builds" \
        | jq -r '.builds[-1].downloads.application.name') && \
    curl -fsSL -o server.jar \
        "https://api.papermc.io/v2/projects/paper/versions/${MC_VER}/builds/${LATEST_BUILD}/downloads/${JAR_NAME}" && \
    echo "Downloaded Paper ${MC_VER} build ${LATEST_BUILD}"

# Accept EULA
RUN echo "eula=true" > eula.txt

# ---------------------------------------------------------------------------
# Stage 2: Runtime — Eclipse Temurin 21 JRE on Ubuntu Noble
# ---------------------------------------------------------------------------
FROM eclipse-temurin:21-jre-noble@sha256:bb4d41d883e59e82cad021feb8e06401c15bff1d40bdaca23cabc48a80c3114b AS runtime

# Non-root user
RUN useradd --no-log-init -r -m -s /usr/sbin/nologin minecraft

# Copy server files from builder
COPY --from=builder --chown=minecraft:minecraft /minecraft /minecraft
COPY --chmod=755 entrypoint.sh /entrypoint.sh

LABEL org.opencontainers.image.title="MC Dad Server" \
      org.opencontainers.image.description="Containerized Minecraft Paper server with Geyser cross-play, Parkour, and tuned configs" \
      org.opencontainers.image.source="https://github.com/KevinTCoughlin/mc-dad-server" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="KevinTCoughlin"

USER minecraft
WORKDIR /minecraft

# Java (25565/tcp), Bedrock/Geyser (19132/udp), RCON (25575/tcp)
EXPOSE 25565/tcp 19132/udp 25575/tcp

HEALTHCHECK --interval=30s --timeout=5s --start-period=90s --retries=3 \
    CMD cat /proc/*/cmdline 2>/dev/null | tr '\0' '\n' | grep -q server.jar || exit 1

ENTRYPOINT ["/entrypoint.sh"]
