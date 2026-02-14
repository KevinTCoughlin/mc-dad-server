#!/usr/bin/env bash
#
# MC Dad Server Installer
# A dead-simple Minecraft server installer for busy dads.
# MIT License - https://github.com/KevinTCoughlin/mc-dad-server
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/KevinTCoughlin/mc-dad-server/main/install.sh | bash
#   OR
#   bash install.sh [--edition bedrock|java] [--dir /path] [--port 25565] [--no-playit]
#
set -euo pipefail

# ─── Configuration Defaults ───────────────────────────────────────────────────
MC_EDITION="${MC_EDITION:-java}"
MC_DIR="${MC_DIR:-$HOME/minecraft-server}"
MC_PORT="${MC_PORT:-25565}"
MC_MEMORY="${MC_MEMORY:-2G}"
MC_SERVER_TYPE="${MC_SERVER_TYPE:-paper}"       # paper, fabric, vanilla
MC_MOTD="${MC_MOTD:-Dads Minecraft Server}"
MC_MAX_PLAYERS="${MC_MAX_PLAYERS:-20}"
MC_DIFFICULTY="${MC_DIFFICULTY:-normal}"
MC_GAMEMODE="${MC_GAMEMODE:-survival}"
MC_ENABLE_PLAYIT="${MC_ENABLE_PLAYIT:-true}"
MC_WHITELIST="${MC_WHITELIST:-true}"
MC_LICENSE_KEY="${MC_LICENSE_KEY:-}"
MC_VERSION="${MC_VERSION:-latest}"
MC_GC_TYPE="${MC_GC_TYPE:-g1gc}"               # g1gc or zgc
MC_CHAT_FILTER="${MC_CHAT_FILTER:-true}"

# ─── Colors & Output ─────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
step()    { echo -e "\n${CYAN}${BOLD}━━━ $* ━━━${NC}\n"; }

# ─── Parse Arguments ─────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case $1 in
        --edition)    MC_EDITION="$2"; shift 2 ;;
        --dir)        MC_DIR="$2"; shift 2 ;;
        --port)       MC_PORT="$2"; shift 2 ;;
        --memory)     MC_MEMORY="$2"; shift 2 ;;
        --type)       MC_SERVER_TYPE="$2"; shift 2 ;;
        --motd)       MC_MOTD="$2"; shift 2 ;;
        --players)    MC_MAX_PLAYERS="$2"; shift 2 ;;
        --difficulty) MC_DIFFICULTY="$2"; shift 2 ;;
        --gamemode)   MC_GAMEMODE="$2"; shift 2 ;;
        --gc)         MC_GC_TYPE="$2"; shift 2 ;;
        --no-playit)  MC_ENABLE_PLAYIT="false"; shift ;;
        --no-chat-filter) MC_CHAT_FILTER="false"; shift ;;
        --license)    MC_LICENSE_KEY="$2"; shift 2 ;;
        --version)    MC_VERSION="$2"; shift 2 ;;
        --help|-h)
            echo "MC Dad Server Installer"
            echo ""
            echo "Usage: bash install.sh [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --edition <java|bedrock>    Server edition (default: java)"
            echo "  --dir <path>                Install directory (default: ~/minecraft-server)"
            echo "  --port <port>               Server port (default: 25565)"
            echo "  --memory <size>             RAM allocation (default: 2G)"
            echo "  --type <paper|fabric|vanilla> Server type (default: paper)"
            echo "  --motd <message>            Server message of the day"
            echo "  --players <count>           Max players (default: 20)"
            echo "  --difficulty <peaceful|easy|normal|hard>"
            echo "  --gamemode <survival|creative|adventure>"
            echo "  --gc <g1gc|zgc>             Garbage collector (default: g1gc)"
            echo "  --no-playit                 Skip playit.gg tunnel setup"
            echo "  --no-chat-filter            Skip chat filter plugin setup"
            echo "  --license <key>             License key for Dad Pack configs"
            echo "  --version <version>         MC version (default: latest)"
            echo "  --help                      Show this help"
            exit 0
            ;;
        *) error "Unknown option: $1"; exit 1 ;;
    esac
done

# ─── OS Detection ─────────────────────────────────────────────────────────────
detect_os() {
    OS="unknown"
    DISTRO="unknown"
    PKG_MGR="unknown"
    INIT_SYSTEM="unknown"

    case "$(uname -s)" in
        Linux*)
            OS="linux"
            if grep -qi microsoft /proc/version 2>/dev/null; then
                OS="wsl"
                info "Detected: Windows Subsystem for Linux"
            fi
            if command -v apt-get &>/dev/null; then
                DISTRO="debian"
                PKG_MGR="apt"
            elif command -v dnf &>/dev/null; then
                DISTRO="fedora"
                PKG_MGR="dnf"
            elif command -v pacman &>/dev/null; then
                DISTRO="arch"
                PKG_MGR="pacman"
            elif command -v zypper &>/dev/null; then
                DISTRO="suse"
                PKG_MGR="zypper"
            fi
            if command -v systemctl &>/dev/null; then
                INIT_SYSTEM="systemd"
            fi
            ;;
        Darwin*)
            OS="macos"
            PKG_MGR="brew"
            INIT_SYSTEM="launchd"
            if ! command -v brew &>/dev/null; then
                warn "Homebrew not found. Installing..."
                /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
            fi
            ;;
        *)
            error "Unsupported OS: $(uname -s)"
            exit 1
            ;;
    esac

    success "OS: $OS | Distro: $DISTRO | Package Manager: $PKG_MGR | Init: $INIT_SYSTEM"
}

# ─── Dependency Installation ──────────────────────────────────────────────────
install_package() {
    local pkg="$1"
    if command -v "$pkg" &>/dev/null; then
        success "$pkg already installed"
        return 0
    fi

    info "Installing $pkg..."
    case "$PKG_MGR" in
        apt)    sudo apt-get update -qq && sudo apt-get install -y -qq "$pkg" ;;
        dnf)    sudo dnf install -y -q "$pkg" ;;
        pacman) sudo pacman -S --noconfirm --quiet "$pkg" ;;
        zypper) sudo zypper install -y "$pkg" ;;
        brew)   brew install "$pkg" ;;
        *)      error "Cannot install $pkg: unknown package manager"; return 1 ;;
    esac
    success "$pkg installed"
}

install_java() {
    step "Installing Java (Adoptium Temurin)"

    # Check if Java 21+ is available
    if command -v java &>/dev/null; then
        local java_ver
        java_ver=$(java -version 2>&1 | head -1 | cut -d'"' -f2 | cut -d'.' -f1)
        if ! [[ "$java_ver" =~ ^[0-9]+$ ]]; then
            warn "Could not determine Java version, will install Java 21"
        elif [[ "$java_ver" -ge 21 ]]; then
            success "Java $java_ver already installed"
            return 0
        else
            warn "Java $java_ver found, but 21+ required"
        fi
    fi

    info "Installing Adoptium Temurin JDK 21..."
    case "$PKG_MGR" in
        apt)
            # Adoptium APT repo
            if ! apt-cache show temurin-21-jdk &>/dev/null 2>&1; then
                info "Adding Adoptium APT repository..."
                sudo apt-get update -qq
                sudo apt-get install -y -qq wget apt-transport-https gpg
                wget -qO - https://packages.adoptium.net/artifactory/api/gpg/key/public | gpg --dearmor | sudo tee /etc/apt/keyrings/adoptium.gpg > /dev/null
                echo "deb [signed-by=/etc/apt/keyrings/adoptium.gpg] https://packages.adoptium.net/artifactory/deb $(awk -F= '/^VERSION_CODENAME/{print$2}' /etc/os-release) main" | sudo tee /etc/apt/sources.list.d/adoptium.list > /dev/null
                sudo apt-get update -qq
            fi
            if sudo apt-get install -y -qq temurin-21-jdk 2>/dev/null; then
                success "Adoptium Temurin 21 installed via APT"
            else
                warn "Adoptium repo unavailable, falling back to distro OpenJDK"
                sudo apt-get install -y -qq openjdk-21-jre-headless
            fi
            ;;
        dnf)
            # Adoptium DNF repo
            if ! dnf list temurin-21-jdk &>/dev/null 2>&1; then
                info "Adding Adoptium DNF repository..."
                cat <<-'REPO' | sudo tee /etc/yum.repos.d/adoptium.repo > /dev/null
[Adoptium]
name=Adoptium
baseurl=https://packages.adoptium.net/artifactory/rpm/$(. /etc/os-release && echo "$ID")/$(rpm -E %{rhel})/$(uname -m)
enabled=1
gpgcheck=1
gpgkey=https://packages.adoptium.net/artifactory/api/gpg/key/public
REPO
            fi
            if sudo dnf install -y -q temurin-21-jdk 2>/dev/null; then
                success "Adoptium Temurin 21 installed via DNF"
            else
                warn "Adoptium repo unavailable, falling back to distro OpenJDK"
                sudo dnf install -y -q java-21-openjdk-headless
            fi
            ;;
        pacman)
            sudo pacman -S --noconfirm jre-openjdk-headless
            ;;
        brew)
            brew install --cask temurin@21
            ;;
        *)
            # Fallback: use SDKMAN
            warn "Using SDKMAN to install Temurin Java..."
            if [[ ! -d "$HOME/.sdkman" ]]; then
                curl -fsSL "https://get.sdkman.io" | bash
                # shellcheck source=/dev/null
                source "$HOME/.sdkman/bin/sdkman-init.sh"
            fi
            sdk install java 21.0.2-tem
            ;;
    esac

    # Verify
    if java -version 2>&1 | grep -q "21"; then
        success "Java 21 installed successfully"
    else
        error "Java installation failed"
        exit 1
    fi
}

# Detect JAVA_HOME and java binary
detect_java() {
    if [[ -n "${JAVA_HOME:-}" && -x "$JAVA_HOME/bin/java" ]]; then
        JAVA_CMD="$JAVA_HOME/bin/java"
    else
        JAVA_CMD="java"
    fi
    export JAVA_CMD
}

install_dependencies() {
    step "Installing Dependencies"
    install_package "curl"
    install_package "jq"
    install_package "screen"

    if [[ "$MC_EDITION" == "java" ]]; then
        install_java
    fi

    detect_java
}

# ─── Server Download ──────────────────────────────────────────────────────────
get_paper_download_url() {
    local version="$1"

    if [[ "$version" == "latest" ]]; then
        version=$(curl -fsSL "https://api.papermc.io/v2/projects/paper" | jq -r '.versions[-1]') \
            || { error "Failed to fetch Paper versions"; exit 1; }
        info "Latest Paper version: $version"
    fi

    local builds_json
    builds_json=$(curl -fsSL "https://api.papermc.io/v2/projects/paper/versions/$version/builds") \
        || { error "Failed to fetch Paper builds for version $version"; exit 1; }
    local build
    build=$(echo "$builds_json" | jq -r '.builds[-1].build')
    local filename
    filename=$(echo "$builds_json" | jq -r '.builds[-1].downloads.application.name')

    if [[ -z "$build" || "$build" == "null" || -z "$filename" || "$filename" == "null" ]]; then
        error "Could not determine Paper download URL for version $version"
        exit 1
    fi

    echo "https://api.papermc.io/v2/projects/paper/versions/$version/builds/$build/downloads/$filename"
}

get_vanilla_download_url() {
    local version="$1"
    local manifest
    manifest=$(curl -fsSL "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json")

    if [[ "$version" == "latest" ]]; then
        version=$(echo "$manifest" | jq -r '.latest.release')
        info "Latest Vanilla version: $version"
    fi

    local version_url
    version_url=$(echo "$manifest" | jq -r --arg v "$version" '.versions[] | select(.id==$v) | .url')
    if [[ -z "$version_url" || "$version_url" == "null" ]]; then
        error "Minecraft version '$version' not found"
        exit 1
    fi
    local version_meta
    version_meta=$(curl -fsSL "$version_url") \
        || { error "Failed to fetch version metadata"; exit 1; }

    echo "$version_meta" | jq -r '.downloads.server.url'
}

download_server() {
    step "Downloading Minecraft Server"

    mkdir -p "$MC_DIR" || { error "Failed to create directory $MC_DIR"; exit 1; }
    cd "$MC_DIR" || { error "Failed to enter directory $MC_DIR"; exit 1; }

    if [[ -f "server.jar" ]]; then
        warn "server.jar already exists. Backing up to server.jar.bak"
        cp server.jar server.jar.bak
    fi

    local download_url=""

    case "$MC_SERVER_TYPE" in
        paper)
            info "Fetching Paper MC server..."
            download_url=$(get_paper_download_url "$MC_VERSION")
            ;;
        vanilla)
            info "Fetching Vanilla MC server..."
            download_url=$(get_vanilla_download_url "$MC_VERSION")
            ;;
        fabric)
            info "Fetching Fabric MC server..."
            local fabric_installer
            fabric_installer=$(curl -fsSL "https://meta.fabricmc.net/v2/versions/installer" | jq -r '.[0].url')
            curl -fsSL -o fabric-installer.jar "$fabric_installer"
            java -jar fabric-installer.jar server -mcversion "${MC_VERSION}" -downloadMinecraft
            rm -f fabric-installer.jar
            mv fabric-server-launch.jar server.jar || warn "Could not rename fabric-server-launch.jar to server.jar — check Fabric installer output"
            success "Fabric server downloaded"
            return 0
            ;;
        *)
            error "Unknown server type: $MC_SERVER_TYPE"
            exit 1
            ;;
    esac

    if [[ -n "$download_url" ]]; then
        info "Downloading from: $download_url"
        curl -fsSL -o server.jar "$download_url"
        success "Server JAR downloaded"
    fi
}

# ─── Plugin Installation ─────────────────────────────────────────────────────
download_plugin() {
    local name="$1"
    local url="$2"
    local filename="$3"
    local dest="$MC_DIR/plugins/$filename"

    if [[ -f "$dest" ]]; then
        success "$name already downloaded"
        return 0
    fi

    info "Downloading $name..."
    if curl -fsSL -o "$dest" "$url"; then
        success "$name downloaded"
    else
        warn "Failed to download $name from $url — you can install it manually"
    fi
}

install_plugins() {
    # Plugins only work with Paper
    if [[ "$MC_SERVER_TYPE" != "paper" ]]; then
        return 0
    fi

    step "Installing Plugins"
    local plugins_dir="$MC_DIR/plugins"
    mkdir -p "$plugins_dir"

    # Geyser (Bedrock cross-play)
    download_plugin "Geyser" \
        "https://download.geysermc.org/v2/projects/geyser/versions/latest/builds/latest/downloads/spigot" \
        "Geyser-Spigot.jar"

    # Floodgate (Bedrock auth — no Java account needed)
    download_plugin "Floodgate" \
        "https://download.geysermc.org/v2/projects/floodgate/versions/latest/builds/latest/downloads/spigot" \
        "Floodgate-Spigot.jar"

    # Parkour (A5H73Y) — from GitHub releases
    local parkour_url
    parkour_url=$(curl -fsSL "https://api.github.com/repos/A5H73Y/Parkour/releases/latest" | jq -r '.assets[0].browser_download_url // empty') || true
    if [[ -n "$parkour_url" ]]; then
        download_plugin "Parkour" "$parkour_url" "Parkour.jar"
    else
        warn "Could not find Parkour download URL — install manually from https://github.com/A5H73Y/Parkour/releases"
    fi

    # Multiverse-Core — from Hangar (PaperMC plugin repo)
    local mv_url
    mv_url=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/latestrelease" 2>/dev/null) || true
    if [[ -n "$mv_url" ]]; then
        download_plugin "Multiverse-Core" \
            "https://hangar.papermc.io/api/v1/projects/Multiverse-Core/versions/${mv_url}/PAPER/download" \
            "Multiverse-Core.jar"
    else
        warn "Could not resolve Multiverse-Core version — install manually from https://hangar.papermc.io/Multiverse/Multiverse-Core"
    fi

    # WorldEdit — from Hangar
    local we_url
    we_url=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/WorldEdit/latestrelease" 2>/dev/null) || true
    if [[ -n "$we_url" ]]; then
        download_plugin "WorldEdit" \
            "https://hangar.papermc.io/api/v1/projects/WorldEdit/versions/${we_url}/PAPER/download" \
            "WorldEdit.jar"
    else
        warn "Could not resolve WorldEdit version — install manually from https://hangar.papermc.io/EngineHub/WorldEdit"
    fi

    success "Plugin installation complete"
}

# ─── Configuration ─────────────────────────────────────────────────────────────
accept_eula() {
    echo "eula=true" > "$MC_DIR/eula.txt"
    success "EULA accepted"
}

generate_rcon_password() {
    # Generate a random RCON password (alphanumeric, 24 chars)
    MC_RCON_PASSWORD=$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 24 || true)
    if [[ -z "$MC_RCON_PASSWORD" ]]; then
        # Fallback if /dev/urandom isn't available
        MC_RCON_PASSWORD="rcon-$(date +%s)-$$"
    fi
}

deploy_configs() {
    step "Deploying Configs"

    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd)" || script_dir=""
    local config_url="https://raw.githubusercontent.com/KevinTCoughlin/mc-dad-server/main/configs"

    # Deploy base config files to server root
    for cfg in server.properties bukkit.yml spigot.yml; do
        if [[ -n "$script_dir" && -f "$script_dir/configs/$cfg" ]]; then
            cp "$script_dir/configs/$cfg" "$MC_DIR/$cfg"
        else
            curl -fsSL "$config_url/$cfg" -o "$MC_DIR/$cfg"
        fi
    done

    # Paper configs go in config/ subdirectory
    mkdir -p "$MC_DIR/config"
    for cfg in paper-global.yml paper-world-defaults.yml; do
        if [[ -n "$script_dir" && -f "$script_dir/configs/$cfg" ]]; then
            cp "$script_dir/configs/$cfg" "$MC_DIR/config/$cfg"
        else
            curl -fsSL "$config_url/$cfg" -o "$MC_DIR/config/$cfg"
        fi
    done

    # Parkour plugin config
    if [[ "$MC_SERVER_TYPE" == "paper" ]]; then
        mkdir -p "$MC_DIR/plugins/Parkour"
        if [[ -n "$script_dir" && -f "$script_dir/configs/parkour-config.yml" ]]; then
            cp "$script_dir/configs/parkour-config.yml" "$MC_DIR/plugins/Parkour/config.yml"
        else
            curl -fsSL "$config_url/parkour-config.yml" -o "$MC_DIR/plugins/Parkour/config.yml"
        fi
    fi

    # Generate RCON password
    generate_rcon_password

    # Substitute template variables in server.properties
    sed -i.bak \
        -e "s|%%MC_PORT%%|${MC_PORT}|g" \
        -e "s|%%MC_MOTD%%|${MC_MOTD}|g" \
        -e "s|%%MC_DIFFICULTY%%|${MC_DIFFICULTY}|g" \
        -e "s|%%MC_GAMEMODE%%|${MC_GAMEMODE}|g" \
        -e "s|%%MC_MAX_PLAYERS%%|${MC_MAX_PLAYERS}|g" \
        -e "s|%%MC_WHITELIST%%|${MC_WHITELIST}|g" \
        -e "s|%%MC_RCON_PASSWORD%%|${MC_RCON_PASSWORD}|g" \
        "$MC_DIR/server.properties"
    rm -f "$MC_DIR/server.properties.bak"

    success "Configs deployed with tuned PaperMC defaults"
    info "RCON password saved to server.properties (port 25575)"
}

# ─── Chat Filter (ChatSentry + Blocked Words) ───────────────────────────────
setup_chat_filter() {
    if [[ "$MC_CHAT_FILTER" != "true" ]]; then
        info "Skipping chat filter setup (--no-chat-filter)"
        return 0
    fi

    if [[ "$MC_SERVER_TYPE" != "paper" ]]; then
        info "Chat filter requires Paper server type — skipping for $MC_SERVER_TYPE"
        return 0
    fi

    step "Setting Up Chat Filter"

    local plugins_dir="$MC_DIR/plugins"
    mkdir -p "$plugins_dir"

    # Copy blocked words list
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd)" || script_dir=""
    local words_dest="$MC_DIR/blocked-words.txt"

    if [[ -n "$script_dir" && -f "$script_dir/blocked-words.txt" ]]; then
        cp "$script_dir/blocked-words.txt" "$words_dest"
    else
        curl -fsSL "https://raw.githubusercontent.com/KevinTCoughlin/mc-dad-server/main/blocked-words.txt" -o "$words_dest"
    fi
    success "Blocked words list deployed"

    # Download ChatSentry plugin from Hangar
    local chatsentry_version
    chatsentry_version=$(curl -fsSL "https://hangar.papermc.io/api/v1/projects/ChatSentry/latestrelease" 2>/dev/null) || true
    if [[ -n "$chatsentry_version" ]]; then
        download_plugin "ChatSentry" \
            "https://hangar.papermc.io/api/v1/projects/ChatSentry/versions/${chatsentry_version}/PAPER/download" \
            "ChatSentry.jar"
    else
        warn "Could not resolve ChatSentry version — chat filter plugin not installed"
        warn "You can install it manually from https://hangar.papermc.io/HexedHero/ChatSentry"
        return 0
    fi

    # Generate minimal ChatSentry config pointing to our blocked words list
    local sentry_dir="$plugins_dir/ChatSentry"
    mkdir -p "$sentry_dir"

    cat > "$sentry_dir/config.yml" << 'CHATEOF'
# ChatSentry config — generated by MC Dad Server installer
# Docs: https://github.com/HexedHero/ChatSentry

chat-filter:
  enabled: true
  blocked-words-file: "../../blocked-words.txt"
  action: BLOCK
  warn-message: "&cThat language isn't allowed on this server!"

# Disable features we don't need
anti-spam:
  enabled: false
anti-caps:
  enabled: false
anti-unicode:
  enabled: false
anti-flood:
  enabled: false
CHATEOF

    success "ChatSentry configured with blocked words filter"
}

# ─── Parkour First-Run Setup Script ──────────────────────────────────────────
create_parkour_setup_script() {
    if [[ "$MC_SERVER_TYPE" != "paper" ]]; then
        return 0
    fi

    cat > "$MC_DIR/setup-parkour.sh" << 'PARKOUREOF'
#!/usr/bin/env bash
set -euo pipefail
#
# Parkour World Setup — run this ONCE after first server boot
# Requires: Multiverse-Core and Parkour plugins loaded
#
# Usage: attach to server console (screen -r minecraft) and paste these commands,
#        OR run this script which sends them via screen.
#

SESSION_NAME="minecraft"

send_cmd() {
    screen -S "$SESSION_NAME" -p 0 -X stuff "$1$(printf '\r')"
    sleep 2
}

if ! screen -list | grep -q "$SESSION_NAME"; then
    echo "Server not running! Start it first with: bash run.sh"
    exit 1
fi

echo "Setting up parkour world..."

# Create flat parkour world
send_cmd "mv create parkour normal --world-type flat --no-structures"
sleep 5

# Configure parkour world
send_cmd "mv modify parkour set gamemode adventure"
send_cmd "mv modify parkour set difficulty peaceful"
send_cmd "mv gamerule set minecraft:spawn_mobs false parkour"
send_cmd "mv gamerule set minecraft:advance_weather false parkour"
send_cmd "mv gamerule set minecraft:advance_time false parkour"
send_cmd "mv gamerule set minecraft:fire_damage false parkour"
send_cmd "mv gamerule set minecraft:spawn_monsters false parkour"
send_cmd "mv gamerule set minecraft:spawn_phantoms false parkour"
send_cmd "mv gamerule set minecraft:mob_griefing false parkour"

echo ""
echo "Parkour world created! Next steps:"
echo "  1. Join the server and run: /mv tp parkour"
echo "  2. Fly to where you want the parkour lobby"
echo "  3. Run: /pa setlobby"
echo "  4. Start building courses with: /pa create <name>"
echo ""
echo "Course building cheat sheet:"
echo "  /pa create MyCourse   — start a course at your position"
echo "  /pa checkpoint        — add checkpoint where you stand"
echo "  /pa finish            — set finish line"
echo "  /pa ready MyCourse    — mark course playable"
echo ""
PARKOUREOF
    chmod +x "$MC_DIR/setup-parkour.sh"
    success "Parkour setup script created: setup-parkour.sh"
}

# ─── Service Management Scripts ───────────────────────────────────────────────
create_management_scripts() {
    step "Creating Management Scripts"

    # ─── Start Script (with GC selection + JAVA_HOME) ───
    cat > "$MC_DIR/start.sh" << STARTEOF
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
cd "\$SCRIPT_DIR"

MEMORY="\${MC_MEMORY:-${MC_MEMORY}}"
GC_TYPE="\${MC_GC_TYPE:-${MC_GC_TYPE}}"

# Detect Java
if [[ -n "\${JAVA_HOME:-}" && -x "\$JAVA_HOME/bin/java" ]]; then
    JAVA_CMD="\$JAVA_HOME/bin/java"
else
    JAVA_CMD="java"
fi

# Build JVM flags
JVM_FLAGS=(
    -Xms"\$MEMORY"
    -Xmx"\$MEMORY"
)

if [[ "\${GC_TYPE,,}" == "zgc" ]]; then
    # ZGC — low latency collector (requires Java 21+)
    JVM_FLAGS+=(
        -XX:+UseZGC
        -XX:+ZGenerational
        -XX:+AlwaysPreTouch
        -XX:+DisableExplicitGC
        -XX:+PerfDisableSharedMem
    )
else
    # G1GC — Aikar's flags optimized for Minecraft
    # https://docs.papermc.io/paper/aikars-flags
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

echo "Starting Minecraft server with \${MEMORY} RAM (\${GC_TYPE^^} GC)..."
exec "\$JAVA_CMD" "\${JVM_FLAGS[@]}" -jar server.jar nogui
STARTEOF
    chmod +x "$MC_DIR/start.sh"

    # ─── Stop Script ───
    cat > "$MC_DIR/stop.sh" << 'STOPEOF'
#!/usr/bin/env bash
set -euo pipefail

SESSION_NAME="minecraft"

if screen -list | grep -q "$SESSION_NAME"; then
    screen -S "$SESSION_NAME" -p 0 -X stuff "say Server shutting down in 10 seconds...$(printf '\r')"
    sleep 10
    screen -S "$SESSION_NAME" -p 0 -X stuff "stop$(printf '\r')"
    echo "Stop command sent. Server shutting down..."
    sleep 5
else
    echo "No running Minecraft server found."
fi
STOPEOF
    chmod +x "$MC_DIR/stop.sh"

    # ─── Restart Script ───
    cat > "$MC_DIR/restart.sh" << 'RESTARTEOF'
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Restarting Minecraft server..."
bash "$SCRIPT_DIR/stop.sh"
sleep 5
bash "$SCRIPT_DIR/run.sh"
RESTARTEOF
    chmod +x "$MC_DIR/restart.sh"

    # ─── Run in Screen ───
    cat > "$MC_DIR/run.sh" << 'RUNEOF'
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

SESSION_NAME="minecraft"

if screen -list | grep -q "$SESSION_NAME"; then
    echo "Server is already running! Use: screen -r $SESSION_NAME"
    exit 1
fi

echo "Starting Minecraft server in screen session '$SESSION_NAME'..."
screen -dmS "$SESSION_NAME" bash "$SCRIPT_DIR/start.sh"
echo ""
echo "Server started! Useful commands:"
echo "  Attach to console:  screen -r $SESSION_NAME"
echo "  Detach from console: Ctrl+A then D"
echo "  Stop server:         bash $SCRIPT_DIR/stop.sh"
echo "  Server status:       bash $SCRIPT_DIR/status.sh"
RUNEOF
    chmod +x "$MC_DIR/run.sh"

    # ─── Status Script ───
    cat > "$MC_DIR/status.sh" << 'STATUSEOF'
#!/usr/bin/env bash

SESSION_NAME="minecraft"

echo "═══ Minecraft Server Status ═══"
echo ""

if screen -list | grep -q "$SESSION_NAME"; then
    echo "  Status:  RUNNING"
    echo "  Session: screen -r $SESSION_NAME"
else
    echo "  Status:  STOPPED"
fi

echo ""

# Show resource usage if running
if pgrep -f "server.jar" > /dev/null; then
    PID=$(pgrep -f "server.jar" | head -1)
    echo "  PID:     $PID"
    echo "  Memory:  $(ps -o rss= -p "$PID" 2>/dev/null | awk '{printf "%.0f MB", $1/1024}')"
    echo "  CPU:     $(ps -o %cpu= -p "$PID" 2>/dev/null)%"
fi

echo ""
STATUSEOF
    chmod +x "$MC_DIR/status.sh"

    # ─── Backup Script ───
    cat > "$MC_DIR/backup.sh" << 'BACKUPEOF'
#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKUP_DIR="$SCRIPT_DIR/backups"
SESSION_NAME="minecraft"
MAX_BACKUPS="${MAX_BACKUPS:-5}"

mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/world_${TIMESTAMP}.tar.gz"

# Tell server to save and disable auto-save temporarily
if screen -list | grep -q "$SESSION_NAME"; then
    screen -S "$SESSION_NAME" -p 0 -X stuff "say Backup starting...$(printf '\r')"
    screen -S "$SESSION_NAME" -p 0 -X stuff "save-all$(printf '\r')"
    sleep 3
    screen -S "$SESSION_NAME" -p 0 -X stuff "save-off$(printf '\r')"
    sleep 1
fi

# Create backup
echo "Creating backup: $BACKUP_FILE"
tar -czf "$BACKUP_FILE" -C "$SCRIPT_DIR" world world_nether world_the_end 2>/dev/null || \
tar -czf "$BACKUP_FILE" -C "$SCRIPT_DIR" world 2>/dev/null

# Re-enable auto-save
if screen -list | grep -q "$SESSION_NAME"; then
    screen -S "$SESSION_NAME" -p 0 -X stuff "save-on$(printf '\r')"
    screen -S "$SESSION_NAME" -p 0 -X stuff "say Backup complete!$(printf '\r')"
fi

# Rotate old backups
BACKUP_COUNT=$(find "$BACKUP_DIR" -name 'world_*.tar.gz' | wc -l)
if [[ "$BACKUP_COUNT" -gt "$MAX_BACKUPS" ]]; then
    find "$BACKUP_DIR" -name 'world_*.tar.gz' -printf '%T@ %p\n' 2>/dev/null | sort -n | head -n -"$MAX_BACKUPS" | cut -d' ' -f2- | xargs rm -f 2>/dev/null || \
    ls -1t "$BACKUP_DIR"/world_*.tar.gz | tail -n +"$((MAX_BACKUPS + 1))" | xargs rm -f
    echo "Rotated old backups (keeping $MAX_BACKUPS)"
fi

BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
echo "Backup complete: $BACKUP_FILE ($BACKUP_SIZE)"
BACKUPEOF
    chmod +x "$MC_DIR/backup.sh"

    # ─── Whitelist Helper ───
    cat > "$MC_DIR/whitelist-add.sh" << 'WLEOF'
#!/usr/bin/env bash
set -euo pipefail

SESSION_NAME="minecraft"

if [[ $# -eq 0 ]]; then
    echo "Usage: bash whitelist-add.sh <player_name> [player_name2] ..."
    echo ""
    echo "Add players to the server whitelist."
    exit 1
fi

for player in "$@"; do
    if screen -list | grep -q "$SESSION_NAME"; then
        screen -S "$SESSION_NAME" -p 0 -X stuff "whitelist add $player$(printf '\r')"
        echo "Added $player to whitelist (server running)"
    else
        echo "Server not running. Add manually to whitelist.json or start the server first."
        exit 1
    fi
done
WLEOF
    chmod +x "$MC_DIR/whitelist-add.sh"

    success "Management scripts created: start.sh, stop.sh, run.sh, restart.sh, status.sh, backup.sh, whitelist-add.sh"
}

# ─── Systemd Service (Linux) ──────────────────────────────────────────────────
setup_systemd_service() {
    if [[ "$INIT_SYSTEM" != "systemd" ]]; then
        warn "Skipping systemd setup (not available)"
        return 0
    fi

    step "Setting Up Systemd Service"

    local service_file="/etc/systemd/system/minecraft.service"

    sudo tee "$service_file" > /dev/null << EOF
[Unit]
Description=Minecraft Server (MC Dad Server)
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=$(whoami)
WorkingDirectory=${MC_DIR}
ExecStart=/usr/bin/bash ${MC_DIR}/start.sh
ExecStop=/usr/bin/bash ${MC_DIR}/stop.sh
Restart=on-failure
RestartSec=30
StandardInput=null
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=${MC_DIR}

[Install]
WantedBy=multi-user.target
EOF

    sudo systemctl daemon-reload
    sudo systemctl enable minecraft.service
    success "Systemd service installed and enabled"
    info "Control with: sudo systemctl start|stop|restart|status minecraft"
}

# ─── LaunchAgent (macOS) ──────────────────────────────────────────────────────
setup_launchd_service() {
    if [[ "$OS" != "macos" ]]; then
        return 0
    fi

    step "Setting Up LaunchAgent (macOS)"

    local plist_dir="$HOME/Library/LaunchAgents"
    local plist_file="$plist_dir/com.mc-dad-server.minecraft.plist"

    mkdir -p "$plist_dir"

    cat > "$plist_file" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.mc-dad-server.minecraft</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/bash</string>
        <string>${MC_DIR}/start.sh</string>
    </array>
    <key>WorkingDirectory</key>
    <string>${MC_DIR}</string>
    <key>RunAtLoad</key>
    <false/>
    <key>KeepAlive</key>
    <false/>
    <key>StandardOutPath</key>
    <string>${MC_DIR}/logs/launchd-stdout.log</string>
    <key>StandardErrorPath</key>
    <string>${MC_DIR}/logs/launchd-stderr.log</string>
</dict>
</plist>
EOF

    success "LaunchAgent created"
    info "Load with: launchctl load $plist_file"
    info "Start with: launchctl start com.mc-dad-server.minecraft"
}

# ─── Cron Backup ──────────────────────────────────────────────────────────────
setup_cron_backup() {
    step "Setting Up Automated Backups"

    # Add daily backup cron job at 4 AM
    local cron_line="0 4 * * * /usr/bin/env bash ${MC_DIR}/backup.sh >> ${MC_DIR}/logs/backup.log 2>&1"

    if crontab -l 2>/dev/null | grep -qF "${MC_DIR}/backup.sh"; then
        warn "Backup cron job already exists"
    else
        (crontab -l 2>/dev/null; echo "# mc-dad-server daily backup"; echo "$cron_line") | crontab -
        success "Daily backup scheduled at 4:00 AM"
    fi

    mkdir -p "$MC_DIR/logs"
}

# ─── Firewall ─────────────────────────────────────────────────────────────────
configure_firewall() {
    step "Configuring Firewall"

    if [[ "$OS" == "macos" ]]; then
        info "macOS firewall: you may need to allow Java in System Settings > Privacy & Security > Firewall"
        return 0
    fi

    if command -v ufw &>/dev/null; then
        sudo ufw allow "$MC_PORT/tcp" comment "Minecraft Server"
        if [[ "$MC_SERVER_TYPE" == "paper" ]]; then
            sudo ufw allow 19132/udp comment "Minecraft Bedrock (Geyser)"
        fi
        success "UFW: opened port $MC_PORT/tcp"
        [[ "$MC_SERVER_TYPE" == "paper" ]] && success "UFW: opened port 19132/udp (Geyser/Bedrock)"
    elif command -v firewall-cmd &>/dev/null; then
        sudo firewall-cmd --permanent --add-port="$MC_PORT/tcp"
        if [[ "$MC_SERVER_TYPE" == "paper" ]]; then
            sudo firewall-cmd --permanent --add-port=19132/udp
        fi
        sudo firewall-cmd --reload
        success "Firewalld: opened port $MC_PORT/tcp"
        [[ "$MC_SERVER_TYPE" == "paper" ]] && success "Firewalld: opened port 19132/udp (Geyser/Bedrock)"
    else
        warn "No known firewall detected. You may need to manually open port $MC_PORT"
        [[ "$MC_SERVER_TYPE" == "paper" ]] && warn "Also open port 19132/udp for Bedrock cross-play (Geyser)"
    fi
}

# ─── playit.gg Tunnel (No Port Forwarding!) ──────────────────────────────────
setup_playit() {
    if [[ "$MC_ENABLE_PLAYIT" != "true" ]]; then
        info "Skipping playit.gg setup (--no-playit)"
        return 0
    fi

    step "Setting Up playit.gg (No Port Forwarding Needed!)"

    echo -e "${BOLD}playit.gg lets your kids' friends connect WITHOUT port forwarding.${NC}"
    echo "This is the safest and easiest way to make your server accessible."
    echo ""

    if command -v playit &>/dev/null; then
        success "playit.gg already installed"
    else
        info "Installing playit.gg agent..."
        local playit_arch
        case "$(uname -m)" in
            x86_64|amd64) playit_arch="amd64" ;;
            aarch64|arm64) playit_arch="aarch64" ;;
            armv7l) playit_arch="armv7" ;;
            *) warn "Unsupported architecture for playit.gg: $(uname -m)"; return 0 ;;
        esac
        local playit_os
        case "$OS" in
            macos) playit_os="darwin" ;;
            *) playit_os="linux" ;;
        esac
        local playit_url="https://github.com/playit-cloud/playit-agent/releases/latest/download/playit-${playit_os}-${playit_arch}"
        if curl -fsSL "$playit_url" -o /tmp/playit && \
            sudo install -m 755 /tmp/playit /usr/local/bin/playit && \
            rm -f /tmp/playit; then
            success "playit.gg installed"
        else
            warn "Could not auto-install playit.gg. Visit https://playit.gg to install manually."
        fi
    fi

    echo ""
    info "To set up your tunnel:"
    echo "  1. Run: playit"
    echo "  2. Follow the link to claim your agent"
    echo "  3. Create a Minecraft tunnel pointing to localhost:${MC_PORT}"
    echo "  4. Share the playit.gg address with your kids!"
    echo ""
}

# ─── Dad Pack (Premium Configs) ───────────────────────────────────────────────
install_dad_pack() {
    if [[ -z "$MC_LICENSE_KEY" ]]; then
        return 0
    fi

    step "Dad Pack"

    info "Dad Pack support is coming soon!"
    info "Follow progress at: https://github.com/KevinTCoughlin/mc-dad-server"
    warn "License key provided but Dad Pack is not yet available. Using default configs."
}

# ─── Summary ──────────────────────────────────────────────────────────────────
print_summary() {
    local divider="══════════════════════════════════════════════════════"

    echo ""
    echo -e "${GREEN}${BOLD}${divider}${NC}"
    echo -e "${GREEN}${BOLD}  MC Dad Server - Installation Complete!${NC}"
    echo -e "${GREEN}${BOLD}${divider}${NC}"
    echo ""
    echo -e "  ${BOLD}Server Directory:${NC}  $MC_DIR"
    echo -e "  ${BOLD}Server Type:${NC}       $MC_SERVER_TYPE"
    echo -e "  ${BOLD}Port:${NC}              $MC_PORT (Java)"
    if [[ "$MC_SERVER_TYPE" == "paper" ]]; then
        echo -e "  ${BOLD}Bedrock Port:${NC}     19132 (Geyser)"
    fi
    echo -e "  ${BOLD}Memory:${NC}            $MC_MEMORY"
    echo -e "  ${BOLD}GC:${NC}                ${MC_GC_TYPE^^}"
    echo -e "  ${BOLD}Whitelist:${NC}         $MC_WHITELIST"
    echo -e "  ${BOLD}Difficulty:${NC}        $MC_DIFFICULTY"
    echo -e "  ${BOLD}Game Mode:${NC}         $MC_GAMEMODE"
    echo ""

    if [[ "$MC_SERVER_TYPE" == "paper" ]]; then
        echo -e "  ${CYAN}${BOLD}Plugins Installed:${NC}"
        echo -e "    Geyser + Floodgate  (Bedrock cross-play)"
        echo -e "    Parkour             (obstacle courses)"
        echo -e "    WorldEdit           (fast building)"
        echo -e "    Multiverse-Core     (multiple worlds)"
        if [[ "$MC_CHAT_FILTER" == "true" ]]; then
            echo -e "    ChatSentry          (chat filter)"
        fi
        echo ""
    fi

    echo -e "  ${CYAN}${BOLD}Quick Start:${NC}"
    echo -e "    Start server:      ${BOLD}bash $MC_DIR/run.sh${NC}"
    echo -e "    Stop server:       ${BOLD}bash $MC_DIR/stop.sh${NC}"
    echo -e "    Server status:     ${BOLD}bash $MC_DIR/status.sh${NC}"
    echo -e "    View console:      ${BOLD}screen -r minecraft${NC}"
    echo -e "    Add to whitelist:  ${BOLD}bash $MC_DIR/whitelist-add.sh KidName${NC}"
    echo -e "    Backup world:      ${BOLD}bash $MC_DIR/backup.sh${NC}"
    echo ""

    if [[ "$MC_SERVER_TYPE" == "paper" ]]; then
        echo -e "  ${CYAN}${BOLD}Bedrock Cross-Play (iPad/Switch/Phone):${NC}"
        echo -e "    Kids on Bedrock connect to your IP on port ${BOLD}19132${NC}"
        echo -e "    No Minecraft Java account needed (Floodgate)"
        echo ""
        echo -e "  ${CYAN}${BOLD}Parkour Setup (first time):${NC}"
        echo -e "    ${BOLD}bash $MC_DIR/setup-parkour.sh${NC}"
        echo ""
    fi

    if [[ "$MC_SERVER_TYPE" == "paper" && "$MC_CHAT_FILTER" == "true" ]]; then
        echo -e "  ${CYAN}${BOLD}Chat Filter:${NC}"
        echo -e "    Blocked words list: ${BOLD}$MC_DIR/blocked-words.txt${NC}"
        echo -e "    Edit the list to customize for your family"
        echo ""
    fi

    if [[ "$INIT_SYSTEM" == "systemd" ]]; then
        echo -e "  ${CYAN}${BOLD}Systemd:${NC}"
        echo -e "    sudo systemctl start minecraft"
        echo -e "    sudo systemctl status minecraft"
        echo ""
    fi

    if [[ "$MC_ENABLE_PLAYIT" == "true" ]]; then
        echo -e "  ${CYAN}${BOLD}Multiplayer (No Port Forwarding):${NC}"
        echo -e "    Run ${BOLD}playit${NC} and follow the setup link"
        echo ""
    fi

    echo -e "  ${YELLOW}${BOLD}Tip:${NC} Your kids connect with: ${BOLD}localhost:${MC_PORT}${NC} (same machine)"
    echo -e "       Or your ${BOLD}local IP:${MC_PORT}${NC} (same network)"
    echo ""
    echo -e "${GREEN}${BOLD}${divider}${NC}"
    echo ""
}

# ─── Main ─────────────────────────────────────────────────────────────────────
main() {
    echo ""
    echo -e "${CYAN}${BOLD}"
    echo "  ╔═══════════════════════════════════════╗"
    echo "  ║     MC Dad Server Installer v2.0      ║"
    echo "  ║   Minecraft for Busy Dads, Made Easy  ║"
    echo "  ╚═══════════════════════════════════════╝"
    echo -e "${NC}"

    detect_os
    install_dependencies
    download_server
    accept_eula
    deploy_configs
    setup_chat_filter
    install_plugins
    create_management_scripts
    create_parkour_setup_script
    setup_cron_backup
    configure_firewall

    if [[ "$INIT_SYSTEM" == "systemd" ]]; then
        setup_systemd_service
    elif [[ "$OS" == "macos" ]]; then
        setup_launchd_service
    fi

    setup_playit
    install_dad_pack
    print_summary
}

main "$@"
