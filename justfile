set shell := ["bash", "-cu"]

bin := "casadellibro-mcp"

# List available recipes
default:
    @just --list

# Tidy go modules
tidy:
    go mod tidy

# (Re)generate gomock mocks
generate:
    go generate ./...

# Format Go sources
fmt:
    gofmt -w .

# Apply go fixes (modernize deprecated APIs)
fix:
    go fix ./...

# go vet static checks
vet:
    go vet ./...

# golangci-lint (vet runs as part of it too)
lint:
    golangci-lint run ./...

# Build the binary into ./bin
build:
    go build -o bin/{{bin}} ./cmd/app

# Run all tests via ginkgo
test:
    go run github.com/onsi/ginkgo/v2/ginkgo -r --randomize-all

# Tests with coverage summary
cover:
    go test ./... -cover

# Run the MCP server over stdio
serve: build
    ./bin/{{bin}} serve

# Scan dependencies for known vulnerabilities
vuln:
    govulncheck ./...

# Full local gate: format, fix, generate, build, vet, lint, test, vuln
check: tidy fmt fix generate build vet lint test vuln

# Alias for CI pipelines (no mutation: no fmt/fix/tidy)
ci: build vet lint test vuln
