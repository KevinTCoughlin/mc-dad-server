#!/usr/bin/env bash
#
# setup-parkour-maps.sh - Download and install curated parkour maps
#
# Idempotent: skips maps that already exist in the server directory.
# Downloads from hielkemaps.com and configures for Paper/Multiverse.
#
# Usage: bash setup-parkour-maps.sh [--dir ~/minecraft-server] [--dry-run]
#

set -euo pipefail

# Defaults
SERVER_DIR="${MC_DIR:-$HOME/minecraft-server}"
SESSION_NAME="minecraft"
DRY_RUN=""

# Parse args
while [[ $# -gt 0 ]]; do
    case $1 in
        --dir)     SERVER_DIR="$2"; shift 2 ;;
        --dry-run) DRY_RUN="true"; shift ;;
        --help|-h)
            echo "Usage: bash setup-parkour-maps.sh [--dir <server-dir>] [--dry-run]"
            echo ""
            echo "Downloads and installs curated Hielke Maps parkour maps."
            echo "Skips any maps that already exist."
            echo ""
            echo "Options:"
            echo "  --dir <path>   Server directory (default: ~/minecraft-server)"
            echo "  --dry-run      Preview without downloading"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# =============================================================================
# Map Catalog
# =============================================================================

MAP_NAMES=(
    "parkour-spiral"
    "parkour-spiral-3"
    "parkour-volcano"
    "parkour-pyramid"
    "parkour-paradise"
)

MAP_URLS=(
    "https://hielkemaps.com/downloads/Parkour Spiral.zip"
    "https://hielkemaps.com/downloads/Parkour Spiral 3.zip"
    "https://hielkemaps.com/downloads/Parkour Volcano.zip"
    "https://hielkemaps.com/downloads/Parkour Pyramid.zip"
    "https://hielkemaps.com/downloads/Parkour Paradise.zip"
)

# =============================================================================
# Helpers
# =============================================================================
log() { echo "[$(date '+%H:%M:%S')] $*"; }

send_cmd() {
    if screen -list 2>/dev/null | grep -q "$SESSION_NAME"; then
        screen -S "$SESSION_NAME" -p 0 -X stuff "$1$(printf '\r')"
        sleep 2
    fi
}

is_server_running() {
    screen -list 2>/dev/null | grep -q "$SESSION_NAME"
}

# =============================================================================
# Paper world config for parkour (disable mobs, weather, explosions)
# =============================================================================
PARKOUR_WORLD_YML='# Paper world config for parkour world
# Optimized for parkour: no mobs, no weather, no explosions

_version: 31

entities:
  spawning:
    spawn-limits:
      ambient: 0
      axolotls: 0
      creature: 0
      monster: 0
      underground_water_creature: 0
      water_ambient: 0
      water_creature: 0

environment:
  disable-explosion-knockback: true
  disable-ice-and-snow: true
  disable-thunder: true
  optimize-explosions: true
'

# =============================================================================
# Main
# =============================================================================
main() {
    log "Parkour map setup starting..."
    log "Server dir: $SERVER_DIR"

    if [[ -n "$DRY_RUN" ]]; then
        log "DRY RUN - no files will be modified"
    fi

    [[ -d "$SERVER_DIR" ]] || { log "ERROR: Server directory not found: $SERVER_DIR"; exit 1; }

    local installed=0
    local skipped=0

    for i in "${!MAP_NAMES[@]}"; do
        local name="${MAP_NAMES[$i]}"
        local url="${MAP_URLS[$i]}"
        local dest="$SERVER_DIR/$name"

        if [[ -d "$dest" ]]; then
            log "SKIP: $name (already exists)"
            ((skipped++))
            continue
        fi

        if [[ -n "$DRY_RUN" ]]; then
            log "WOULD INSTALL: $name from $url"
            continue
        fi

        log "INSTALLING: $name"

        # Download to temp directory
        local tmp_dir
        tmp_dir="$(mktemp -d)"
        trap "rm -rf '$tmp_dir'" EXIT

        log "  Downloading from $url..."
        if ! curl -fSL -o "$tmp_dir/map.zip" "$url"; then
            log "  ERROR: Download failed for $name, skipping"
            rm -rf "$tmp_dir"
            continue
        fi

        # Extract
        log "  Extracting..."
        unzip -q "$tmp_dir/map.zip" -d "$tmp_dir/extracted"

        # Find the world folder (contains level.dat)
        local world_dir=""
        world_dir="$(find "$tmp_dir/extracted" -name "level.dat" -print -quit 2>/dev/null)"
        if [[ -z "$world_dir" ]]; then
            log "  ERROR: No level.dat found in zip for $name, skipping"
            rm -rf "$tmp_dir"
            continue
        fi
        world_dir="$(dirname "$world_dir")"

        # Move to server directory
        log "  Installing to $dest..."
        mv "$world_dir" "$dest"

        # Write paper-world.yml (disable mobs/weather for parkour)
        echo "$PARKOUR_WORLD_YML" > "$dest/paper-world.yml"
        log "  Created paper-world.yml"

        # Import into Multiverse if server is running
        if is_server_running; then
            log "  Importing into Multiverse..."
            send_cmd "mv import $name normal"
        else
            log "  Server not running; import with: mv import $name normal"
        fi

        rm -rf "$tmp_dir"
        trap - EXIT
        ((installed++))
        log "  Done: $name"
    done

    echo ""
    log "Setup complete: $installed installed, $skipped skipped"
    log "Available parkour maps: ${MAP_NAMES[*]}"

    if ! is_server_running && ((installed > 0)); then
        echo ""
        log "Start your server, then maps will be auto-imported by Multiverse."
        log "Or import manually in console: mv import <name> normal"
    fi
}

main
