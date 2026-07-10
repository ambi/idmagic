# App repository command map for humans and AI agents.
#
# This app repo consumes RA/SCL tools from the embedded tools/ directory.

set shell := ["zsh", "-cu"]

ra_cmd := env_var_or_default("RA_CMD", "bun run tools/ra/src/main.ts")
go_cache := env_var_or_default("GOCACHE", "/tmp/idmagic-go-cache")
golangci_cache := env_var_or_default("GOLANGCI_LINT_CACHE", "/tmp/idmagic-golangci-cache")
git_commit := `git rev-parse HEAD 2>/dev/null || echo "unknown"`
build_date := `date -u +'%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "unknown"`
version := env_var_or_default("VERSION", "0.0.0-dev")
ldflags := "-X github.com/ambi/idmagic/internal/shared/version.Version=" + version + " -X github.com/ambi/idmagic/internal/shared/version.GitCommit=" + git_commit + " -X github.com/ambi/idmagic/internal/shared/version.BuildDate=" + build_date

# Show this command map.
default:
    @just --list

# Install local dependencies and setup RA.
setup: setup-ra install-ui

# Setup RA tools dependencies and link agent skills.
setup-ra:
    cd tools && bun install
    mkdir -p .agents/skills
    mkdir -p .claude
    ln -sfn ../.agents/skills .claude/skills


# Install UI dependencies.
install-ui:
    cd ui && bun install --frozen-lockfile

# Run the standard app verification suite.
verify: yaml-check verify-go verify-ui

# Verify Go backend with lint and race-enabled tests.
verify-go: lint-go test-go-race

# Run Go lint.
lint-go:
    GOLANGCI_LINT_CACHE={{golangci_cache}} golangci-lint run ./...

# Format Go backend code.
format-go:
     GOLANGCI_LINT_CACHE={{golangci_cache}} golangci-lint fmt ./...

# Run Go tests.
test-go:
    GOCACHE={{go_cache}} go test ./...

# Run race-enabled Go tests.
test-go-race:
    GOCACHE={{go_cache}} go test -race ./...

# Run Go tests with coverage.
test-go-cover:
    GOCACHE={{go_cache}} go test -coverprofile=coverage.out -covermode=atomic ./...
    go tool cover -func=coverage.out

# Run Go fuzz targets for a package.
test-go-fuzz package fuzztime="30s":
    GOCACHE={{go_cache}} go test -run=Fuzz -fuzz=Fuzz -fuzztime={{fuzztime}} {{package}}

# Build all Go packages.
build-go:
    GOCACHE={{go_cache}} go build -ldflags '{{ldflags}}' ./...

# Verify UI with format check, lint, typecheck, and build.
verify-ui: format-check-ui lint-ui typecheck-ui test-ui-unit build-ui

# Run UI format check.
format-check-ui:
    cd ui && bun run format:check

# Format UI.
format-ui:
    cd ui && bun run format

# Run UI lint.
lint-ui:
    cd ui && bun run lint

# Run UI typecheck.
typecheck-ui:
    cd ui && bun run typecheck

# Run UI unit tests.
test-ui-unit:
    cd ui && bun run test:unit

# Run UI unit tests with coverage.
test-ui-cover:
    cd ui && bun run test:unit:coverage

# Build UI.
build-ui:
    cd ui && bun run build

# Run UI E2E tests.
test-ui-e2e:
    cd ui && bun run test:e2e

# Validate SCL and Work Item YAML.
yaml-check: yaml-check-scl yaml-check-work-items check-ids

# Validate SCL YAML files.
yaml-check-scl:
    {{ra_cmd}} yaml-check --scl

# Validate Work Item YAML files.
yaml-check-work-items:
    {{ra_cmd}} yaml-check --work-items

# Detect duplicate / mismatched change-record ids.
check-ids:
    {{ra_cmd}} yaml-check --ids

# Regenerate SCL-derived artifacts.
scl-render:
    {{ra_cmd}} render

# Start the local dev stack (Go API + React UI together with live reload).
dev:
    ./dev.sh

# Start the Go API for local UI development.
dev-api:
    ADDR=:8081 ISSUER=http://localhost:5173 WEBAUTHN_RP_ID="${WEBAUTHN_RP_ID:-localhost}" WEBAUTHN_RP_ORIGINS="${WEBAUTHN_RP_ORIGINS:-http://localhost:5173}" WEBAUTHN_RP_DISPLAY_NAME="${WEBAUTHN_RP_DISPLAY_NAME:-IdMagic Local}" go run ./cmd/idmagic

# Start the React UI dev server.
dev-ui:
    cd ui && bun run dev

# Start the Docker Compose development stack.
dev-compose:
    docker compose -f deploy/docker/docker-compose.dev.yaml up --build

# Run the OAuth2 / OIDC demo against a running server (default http://localhost:8080).
demo:
    ./demo.sh
