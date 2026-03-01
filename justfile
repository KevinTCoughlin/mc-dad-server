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
tools:
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install mvdan.cc/gofumpt@latest
    go install github.com/goreleaser/goreleaser/v2@latest

# Clean build artifacts
clean:
    rm -rf {{binary}} dist/ coverage.out

# --- Container recipes ---

# Build container image (--format docker required for SHELL instruction support)
container-build:
    podman build --format docker -t {{binary}}:latest .

# Start containerized server (BUILDAH_FORMAT needed for SHELL instruction in Containerfile)
container-up:
    BUILDAH_FORMAT=docker podman compose up --build -d

# Stop containerized server
container-down:
    podman compose down

# Follow container logs
container-logs:
    podman compose logs -f

# Exec into running container
container-shell:
    podman exec -it minecraft bash

# Send command to running container via stdin FIFO
container-fifo cmd:
    @echo "{{cmd}}" | podman exec -i minecraft bash -c 'cat > /tmp/mc-input'

# Build image, install quadlet, restart service
container-deploy:
    podman build --format docker -t {{binary}}:latest .
    mkdir -p ~/.config/containers/systemd
    cp quadlet/minecraft.container ~/.config/containers/systemd/
    systemctl --user daemon-reload
    systemctl --user restart minecraft

# Show container service status
container-status:
    systemctl --user status minecraft
