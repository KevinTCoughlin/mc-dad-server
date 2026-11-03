#!/bin/bash
# MC Dad Server — Container entrypoint
# Manages the Minecraft Java process with graceful multi-step shutdown.
set -euo pipefail

# --- Environment defaults ---
MEMORY="${MEMORY:-2G}"
GC_TYPE="${GC_TYPE:-g1gc}"
PORT="${PORT:-25565}"
RCON_PORT="${RCON_PORT:-25575}"
RCON_PASSWORD="${RCON_PASSWORD:?RCON_PASSWORD must be set}"

escape_sed_replacement() {
    local value="$1"
    value="${value//\\/\\\\}"
    value="${value//&/\\&}"
    value="${value//|/\\|}"
    printf '%s' "${value}"
}

escape_property_replacement() {
    local value="${1//\\/\\\\}"
    escape_sed_replacement "${value}"
}

configure_server_properties() {
    local value difficulty gamemode max_players motd port rcon_port rcon_password whitelist

    for value in "${DIFFICULTY:-normal}" "${GAMEMODE:-survival}" "${MAX_PLAYERS:-20}" \
        "${MOTD:-Dads Minecraft Server}" "${PORT}" "${RCON_PASSWORD}" "${WHITELIST:-true}" "${RCON_PORT}"; do
        if [[ "${value}" == *$'\n'* || "${value}" == *$'\r'* ]]; then
            echo "[entrypoint] Configuration values must not contain newlines" >&2
            return 1
        fi
    done

    difficulty="$(escape_sed_replacement "${DIFFICULTY:-normal}")"
    gamemode="$(escape_sed_replacement "${GAMEMODE:-survival}")"
    max_players="$(escape_sed_replacement "${MAX_PLAYERS:-20}")"
    motd="$(escape_property_replacement "${MOTD:-Dads Minecraft Server}")"
    port="$(escape_sed_replacement "${PORT}")"
    rcon_port="$(escape_sed_replacement "${RCON_PORT}")"
    rcon_password="$(escape_property_replacement "${RCON_PASSWORD}")"
    whitelist="$(escape_sed_replacement "${WHITELIST:-true}")"

    if [[ -f server.properties ]] && [[ -w server.properties ]]; then
        sed -i \
            -e "s|%%MC_DIFFICULTY%%|${difficulty}|" \
            -e "s|%%MC_GAMEMODE%%|${gamemode}|" \
            -e "s|%%MC_MAX_PLAYERS%%|${max_players}|" \
            -e "s|%%MC_MOTD%%|${motd}|" \
            -e "s|%%MC_PORT%%|${port}|g" \
            -e "s|%%MC_RCON_PASSWORD%%|${rcon_password}|" \
            -e "s|%%MC_WHITELIST%%|${whitelist}|" \
            -e "s/^enable-rcon=.*/enable-rcon=true/" \
            -e "s|^rcon\\.port=.*|rcon.port=${rcon_port}|" \
            -e "s|^rcon\\.password=.*|rcon.password=${rcon_password}|" \
            server.properties
    fi
}

# Paper renames whole world directories during format upgrades, so the worlds
# volume is mounted at a parent directory and passed as --world-container.
# Earlier images mounted the volume at /minecraft/world, which left the world at
# the volume root; move it one level down, inside the same volume, on first start.
prepare_world_container() {
    local dir="${WORLDS_DIR:-/minecraft/worlds}" level staging entry
    WORLD_ARGS=()
    [[ -d "${dir}" ]] || return 0

    level="$(sed -n 's/^level-name=//p' server.properties 2>/dev/null | tail -n 1)"
    level="${level:-world}"
    staging="${dir}/.mc-dad-world-migration"

    if [[ ! -e "${dir}/${level}" ]] && { [[ -f "${dir}/level.dat" ]] || [[ -d "${staging}" ]]; }; then
        echo "[entrypoint] Moving the world at the root of ${dir} into ${dir}/${level}..."
        mkdir -p "${staging}"
        shopt -s dotglob nullglob
        for entry in "${dir}"/*; do
            case "${entry##*/}" in
                .mc-dad-world-migration | lost+found) continue ;;
            esac
            mv -- "${entry}" "${staging}/"
        done
        shopt -u dotglob nullglob
        mv -- "${staging}" "${dir}/${level}"
    fi

    WORLD_ARGS=(--world-container "${dir}")
}

if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
    return 0
fi

# --- Substitute template variables and configure RCON in server.properties ---
configure_server_properties
prepare_world_container

# --- Build JVM flags ---
JVM_FLAGS=(
    -Xms"${MEMORY}"
    -Xmx"${MEMORY}"
)

# Use AppCDS archive if available (baked during container build)
if [[ -f app-cds.jsa ]]; then
    JVM_FLAGS+=(-XX:SharedArchiveFile=app-cds.jsa)
    echo "[entrypoint] Using AppCDS archive for faster startup"
fi

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
    exec 3>&-
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
rm -f "${FIFO}"
mkfifo "${FIFO}"
exec 3<>"${FIFO}"

echo "[entrypoint] Starting Minecraft server with ${MEMORY} RAM (${gc_lower^^} GC)..."

# Keep the FIFO open in the parent without an orphaned tail process.
java "${JVM_FLAGS[@]}" -jar server.jar "${WORLD_ARGS[@]}" nogui < "${FIFO}" 3>&- &
JAVA_PID=$!

echo "[entrypoint] Minecraft started (PID: ${JAVA_PID})"

wait "${JAVA_PID}"
