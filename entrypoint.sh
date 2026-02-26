#!/bin/bash
# MC Dad Server — Container entrypoint
# Manages the Minecraft Java process with graceful multi-step shutdown.
set -euo pipefail

# --- Environment defaults ---
MEMORY="${MEMORY:-2G}"
GC_TYPE="${GC_TYPE:-g1gc}"
PORT="${PORT:-25565}"
RCON_PORT="${RCON_PORT:-25575}"
RCON_PASSWORD="${RCON_PASSWORD:-changeme}"

# --- Configure RCON in server.properties ---
if [[ -f server.properties ]]; then
    sed -i \
        -e "s/^enable-rcon=.*/enable-rcon=true/" \
        -e "s/^rcon\\.port=.*/rcon.port=${RCON_PORT}/" \
        -e "s/^rcon\\.password=.*/rcon.password=${RCON_PASSWORD}/" \
        server.properties
    # Add RCON settings if missing
    grep -q '^enable-rcon=' server.properties || echo "enable-rcon=true" >> server.properties
    grep -q '^rcon\.port=' server.properties || echo "rcon.port=${RCON_PORT}" >> server.properties
    grep -q '^rcon\.password=' server.properties || echo "rcon.password=${RCON_PASSWORD}" >> server.properties
fi

# --- Build JVM flags ---
JVM_FLAGS=(
    -Xms"${MEMORY}"
    -Xmx"${MEMORY}"
)

gc_lower="${GC_TYPE,,}"
if [[ "${gc_lower}" == "zgc" ]]; then
    JVM_FLAGS+=(
        -XX:+UseZGC
        -XX:+ZGenerational
        -XX:+AlwaysPreTouch
        -XX:+DisableExplicitGC
        -XX:+PerfDisableSharedMem
    )
else
    # G1GC — Aikar's flags (https://mcflags.emc.gs)
    JVM_FLAGS+=(
        -XX:+UseG1GC
        -XX:+ParallelRefProcEnabled
        -XX:MaxGCPauseMillis=200
        -XX:+UnlockExperimentalVMOptions
        -XX:+DisableExplicitGC
        -XX:+AlwaysPreTouch
        -XX:G1NewSizePercent=30
        -XX:G1MaxNewSizePercent=40
        -XX:G1HeapRegionSize=8M
        -XX:G1ReservePercent=20
        -XX:G1HeapWastePercent=5
        -XX:G1MixedGCCountTarget=4
        -XX:InitiatingHeapOccupancyPercent=15
        -XX:G1MixedGCLiveThresholdPercent=90
        -XX:G1RSetUpdatingPauseTimePercent=5
        -XX:SurvivorRatio=32
        -XX:+PerfDisableSharedMem
        -XX:MaxTenuringThreshold=1
        -Dusing.aikars.flags=https://mcflags.emc.gs
        -Daikars.new.flags=true
    )
fi

# --- FIFO for external command injection ---
FIFO=/tmp/mc-input
JAVA_PID=""

cleanup() {
    rm -f "${FIFO}"
}
trap cleanup EXIT

mc_command() {
    if [[ -p "${FIFO}" ]]; then
        echo "$1" > "${FIFO}"
    fi
}

graceful_shutdown() {
    echo "[entrypoint] Caught shutdown signal, starting graceful shutdown..."

    if [[ -n "${JAVA_PID}" ]] && kill -0 "${JAVA_PID}" 2>/dev/null; then
        mc_command "say [SERVER] Shutting down in 30 seconds..."
        sleep 20

        mc_command "say [SERVER] Shutting down in 10 seconds..."
        sleep 5

        mc_command "say [SERVER] Shutting down in 5 seconds..."
        sleep 3

        mc_command "say [SERVER] Shutting down in 2 seconds..."
        sleep 1

        mc_command "say [SERVER] Shutting down in 1 second..."
        sleep 1

        mc_command "say [SERVER] Goodbye!"

        echo "[entrypoint] Sending stop command..."
        mc_command "stop"

        # Wait up to 15s for clean shutdown
        for _ in $(seq 1 15); do
            kill -0 "${JAVA_PID}" 2>/dev/null || break
            sleep 1
        done

        # Force kill if still running
        if kill -0 "${JAVA_PID}" 2>/dev/null; then
            echo "[entrypoint] Force-killing Java process..."
            kill -9 "${JAVA_PID}" 2>/dev/null || true
        fi
    fi

    wait "${JAVA_PID}" 2>/dev/null || true
    echo "[entrypoint] Shutdown complete."
}
trap graceful_shutdown SIGTERM SIGINT

# --- Create FIFO for external command input ---
mkfifo "${FIFO}"

echo "[entrypoint] Starting Minecraft server with ${MEMORY} RAM (${gc_lower^^} GC)..."

# Pipe FIFO input to Java stdin so external processes can send commands
tail -f "${FIFO}" | java "${JVM_FLAGS[@]}" -jar server.jar nogui &
JAVA_PID=$!

echo "[entrypoint] Minecraft started (PID: ${JAVA_PID})"

wait "${JAVA_PID}" || true
