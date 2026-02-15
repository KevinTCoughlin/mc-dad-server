default:
    @just --list

# Run all tests
test:
    go test -race ./...

# Run linter
lint:
    golangci-lint run

# Format code
fmt:
    gofumpt -w .

# Run go vet
vet:
    go vet ./...

# Run all checks (fmt + vet + lint + test)
check: fmt vet lint test

# Build the binary
build:
    go build -ldflags "-X main.version=dev -X main.commit=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)" -o mc-dad-server ./cmd/mc-dad-server/

# Cross-compile all release targets
build-all:
    CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "-X main.version=dev" -o dist/mc-dad-server-linux-amd64 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "-X main.version=dev" -o dist/mc-dad-server-linux-arm64 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=linux   GOARCH=arm   go build -ldflags "-X main.version=dev" -o dist/mc-dad-server-linux-armv7 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "-X main.version=dev" -o dist/mc-dad-server-darwin-amd64 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "-X main.version=dev" -o dist/mc-dad-server-darwin-arm64 ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=dev" -o dist/mc-dad-server-windows-amd64.exe ./cmd/mc-dad-server/
    CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags "-X main.version=dev" -o dist/mc-dad-server-windows-arm64.exe ./cmd/mc-dad-server/

# Install dev tools (golangci-lint, gofumpt, goreleaser)
tools:
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install mvdan.cc/gofumpt@latest
    go install github.com/goreleaser/goreleaser/v2@latest

# Clean build artifacts
clean:
    rm -rf mc-dad-server dist/
