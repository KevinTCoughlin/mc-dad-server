# mc-dad-server justfile

version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
commit  := `git rev-parse --short HEAD 2>/dev/null || echo unknown`
binary  := "mc-dad-server"
module  := "github.com/KevinTCoughlin/mc-dad-server"
ldflags := "-s -w -X main.version=" + version + " -X main.commit=" + commit

default:
    @just --list

# Run all tests with race detector
test:
    go test -race ./...

# Run linter
lint:
    golangci-lint run

# Format code with gofumpt
fmt:
    gofumpt -w .

# Run go vet
vet:
    go vet ./...

# Run all checks (fmt + vet + lint + test)
check: fmt vet lint test

# Build the binary
build:
    go build -ldflags "{{ldflags}}" -o {{binary}} ./cmd/mc-dad-server/

# Build and run with optional args
run *ARGS: build
    ./{{binary}} {{ARGS}}

# Cross-compile all release targets
build-all:
    CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-linux-amd64 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-linux-arm64 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=linux   GOARCH=arm   go build -ldflags "{{ldflags}}" -o dist/{{binary}}-linux-armv7 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-darwin-amd64 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-darwin-arm64 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-windows-amd64.exe ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags "{{ldflags}}" -o dist/{{binary}}-windows-arm64.exe ./cmd/mc-dad-server/

# Install to GOPATH/bin
install:
    go install -ldflags "{{ldflags}}" ./cmd/mc-dad-server/

# Tidy go modules
tidy:
    go mod tidy

# Run tests with coverage report
coverage:
    go test -race -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Release snapshot via goreleaser (local, no publish)
release:
    goreleaser release --snapshot --clean

# Install dev tools (golangci-lint, gofumpt, goreleaser)
# golangci-lint is pinned to the version CI runs, and built with this repo's
# toolchain: a golangci-lint built by an older Go refuses to load a config
# targeting a newer one.
tools:
    GOTOOLCHAIN=$(go version | cut -d' ' -f3) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
    go install mvdan.cc/gofumpt@latest
    go install github.com/goreleaser/goreleaser/v2@latest

# Clean build artifacts
clean:
    rm -rf {{binary}} dist/ coverage.out

# --- Container recipes ---
#
# ENGINE picks the container runtime: Apple's `container` CLI by default on
# macOS, Podman elsewhere. Override with CONTAINER_ENGINE=podman|docker|container.
# Apple's `container` has no compose, so on it `container-up` runs the same
# ports/volumes/limits as compose.yml with `container run`, and
# `container-deploy` drives the launchd agent written by `setup-container`.

ENGINE := env("CONTAINER_ENGINE", if os() == "macos" { "container" } else { "podman" })
launchd_label := "com.mc-dad-server.minecraft-container"

# Build container image (--format docker required for SHELL instruction support on Podman)
container-build:
    #!/usr/bin/env bash
    set -euo pipefail
    case "{{ENGINE}}" in
      podman) podman build --format docker -t {{binary}}:latest . ;;
      *)      {{ENGINE}} build -t {{binary}}:latest . ;;
    esac

# Start containerized server (BUILDAH_FORMAT needed for SHELL instruction in Containerfile)
container-up:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ "{{ENGINE}}" != container ]; then
      BUILDAH_FORMAT=docker {{ENGINE}} compose up --build -d
      exit
    fi
    [ -f .env ] || { echo "error: .env not found (cp .env.example .env and set RCON_PASSWORD)" >&2; exit 1; }
    container system status >/dev/null 2>&1 || container system start
    container build -t {{binary}}:latest .
    # Apple's container creates named volumes empty and root-owned; seed new
    # ones from the image like Podman/Docker do (plugins, uid 1000 ownership).
    for v in worlds plugins logs backups; do
      if ! container volume inspect "minecraft-$v" >/dev/null 2>&1; then
        container volume create "minecraft-$v" >/dev/null
        container run --rm --progress none --user 0 --entrypoint /bin/sh -v "minecraft-$v:/seed" {{binary}}:latest \
          -c "if [ -d /minecraft/$v ]; then cp -a /minecraft/$v/. /seed/; fi; chown 1000:1000 /seed"
      fi
    done
    set -a; . ./.env; set +a
    cpus="${CPUS:-2}"; cpus="${cpus%%.*}"  # container wants whole CPUs; compose allows 2.0
    container rm -f minecraft >/dev/null 2>&1 || true
    container run -d --progress none --name minecraft \
      -p "${PORT:-25565}:25565/tcp" \
      -p "${BEDROCK_PORT:-19132}:19132/udp" \
      -p 127.0.0.1:25575:25575/tcp \
      -v minecraft-worlds:/minecraft/worlds \
      -v minecraft-plugins:/minecraft/plugins \
      -v minecraft-logs:/minecraft/logs \
      -v minecraft-backups:/minecraft/backups \
      -e MEMORY="${MEMORY:-2G}" -e GC_TYPE="${GC_TYPE:-g1gc}" -e PORT=25565 -e RCON_PORT=25575 \
      -e RCON_PASSWORD="${RCON_PASSWORD:?set RCON_PASSWORD in .env}" \
      --cap-drop ALL \
      -m "${MEM_LIMIT:-3g}" -c "$cpus" \
      {{binary}}:latest

# Stop containerized server
container-down:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ "{{ENGINE}}" = container ]; then
      container stop -t 60 minecraft 2>/dev/null || true
      container rm minecraft 2>/dev/null || true
    else
      {{ENGINE}} compose down
    fi

# Follow container logs
container-logs:
    #!/usr/bin/env bash
    if [ "{{ENGINE}}" = container ]; then exec container logs -f minecraft; fi
    exec {{ENGINE}} compose logs -f

# Exec into running container
container-shell:
    {{ENGINE}} exec -it minecraft bash

# Send command to running container via stdin FIFO
container-fifo cmd:
    @echo "{{cmd}}" | {{ENGINE}} exec -i minecraft bash -c 'cat > /tmp/mc-input'

# Build image and restart the managed service (Quadlet on Linux, launchd agent with Apple container)
container-deploy:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ "{{ENGINE}}" = container ]; then
      container build -t {{binary}}:latest .
      plist="$HOME/Library/LaunchAgents/{{launchd_label}}.plist"
      [ -f "$plist" ] || { echo "error: $plist not found; run 'mc-dad-server setup-container' first" >&2; exit 1; }
      launchctl print "gui/$(id -u)/{{launchd_label}}" >/dev/null 2>&1 || launchctl bootstrap "gui/$(id -u)" "$plist"
      launchctl kickstart -k "gui/$(id -u)/{{launchd_label}}"
      exit
    fi
    just ENGINE={{ENGINE}} container-build
    mkdir -p ~/.config/containers/systemd
    cp quadlet/minecraft.container ~/.config/containers/systemd/
    systemctl --user daemon-reload
    systemctl --user restart minecraft

# Show container service status
container-status:
    #!/usr/bin/env bash
    if [ "{{ENGINE}}" = container ]; then
      launchctl print "gui/$(id -u)/{{launchd_label}}" 2>/dev/null | grep -E '^\s+(state|last exit code|pid) =' || echo "launchd agent not loaded"
      container list --all | awk 'NR==1 || $1=="minecraft"'
      exit
    fi
    systemctl --user status minecraft
