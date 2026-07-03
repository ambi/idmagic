# App repository command map for humans and AI agents.
#
# This app repo consumes RA/SCL tools from the GitHub-backed core repository
# mounted at .ra-scl/regenerative-architecture. Override RA_SCL_CORE only when
# deliberately testing a local core checkout.

set shell := ["zsh", "-cu"]

ra_cmd := env_var_or_default("RA_CMD", "bun run .ra/regenerative-architecture/tools/ra/src/main.ts")
go_cache := env_var_or_default("GOCACHE", "/tmp/idmagic-go-cache")
golangci_cache := env_var_or_default("GOLANGCI_LINT_CACHE", "/tmp/idmagic-golangci-cache")

# Show this command map.
default:
    @just --list

# Install local dependencies and setup RA submodule.
setup: setup-ra install-ui

# Setup RA submodule, install tools dependencies, and link agent skills.
setup-ra:
    git submodule update --init --recursive
    cd .ra/regenerative-architecture/tools && bun install
    mkdir -p .claude
    ln -sfn ../.ra/regenerative-architecture/.claude/skills .claude/skills
    mkdir -p .agents
    ln -sfn ../.ra/regenerative-architecture/.claude/skills .agents/skills


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

# Run Go tests.
test-go:
    GOCACHE={{go_cache}} go test ./...

# Run race-enabled Go tests.
test-go-race:
    GOCACHE={{go_cache}} go test -race ./...

# Build all Go packages.
build-go:
    GOCACHE={{go_cache}} go build ./...

# Verify UI with format check, lint, typecheck, and build.
verify-ui: format-check-ui lint-ui typecheck-ui build-ui

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

# Start the Go API for local UI development.
dev-api:
    ADDR=:8081 ISSUER=http://localhost:5173 go run ./cmd/idmagic

# Start the React UI dev server.
dev-ui:
    cd ui && bun run dev

# Start the Docker Compose development stack.
dev-compose:
    docker compose -f deploy/docker/docker-compose.dev.yaml up --build
