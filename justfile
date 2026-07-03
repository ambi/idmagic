# App repository command map for humans and AI agents.
#
# This app repo consumes RA/SCL tools from the GitHub-backed core repository
# mounted at .ra-scl/regenerative-architecture. Override RA_SCL_CORE only when
# deliberately testing a local core checkout.

set shell := ["zsh", "-cu"]

ra_scl_core := env_var_or_default("RA_SCL_CORE", ".ra-scl/regenerative-architecture")
ra_scl_tools := ra_scl_core + "/tools"
app_from_tools := "../../.."
go_cache := env_var_or_default("GOCACHE", "/tmp/idmagic-go-cache")
golangci_cache := env_var_or_default("GOLANGCI_LINT_CACHE", "/tmp/idmagic-golangci-cache")

# Show this command map.
default:
    @just --list

# Install RA/SCL tool dependencies and local dependencies.
setup: setup-ra-scl install-ui

# Install or update the GitHub-backed RA/SCL core submodule.
setup-ra-scl:
    git submodule update --init --recursive .ra-scl/regenerative-architecture
    cd {{ra_scl_tools}} && bun install --frozen-lockfile

# Move the RA/SCL core submodule to the latest GitHub default branch.
update-ra-scl:
    git submodule update --init --remote --merge .ra-scl/regenerative-architecture
    cd {{ra_scl_tools}} && bun install --frozen-lockfile

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
    cd {{ra_scl_tools}} && bun run yaml-check -- --schema=scl {{app_from_tools}}/spec/scl.yaml '{{app_from_tools}}/spec/contexts/*.yaml'

# Validate Work Item YAML files.
yaml-check-work-items:
    cd {{ra_scl_tools}} && bun run yaml-check -- --schema=work-item '{{app_from_tools}}/work-items/*.yaml' '{{app_from_tools}}/work-items/done/*.yaml'

# Detect duplicate / mismatched change-record ids.
check-ids:
    cd {{ra_scl_tools}} && bun run yaml-check/src/check-ids.ts --work-items {{app_from_tools}}/work-items --decisions {{app_from_tools}}/decisions

# Regenerate SCL-derived artifacts.
scl-render:
    cd {{ra_scl_tools}} && bun run scl-to-html -- --scl {{app_from_tools}}/spec/scl.yaml --title IdMagic --out {{app_from_tools}}/spec/idmagic.html
    cd {{ra_scl_tools}} && bun run scl-to-html -- --scl {{app_from_tools}}/spec/scl.yaml --decisions {{app_from_tools}}/decisions --work-items {{app_from_tools}}/work-items --title IdMagic --out {{app_from_tools}}/spec/idmagic.full.html
    cd {{ra_scl_tools}} && bun run scl-to-jsonschema -- --scl {{app_from_tools}}/spec/scl.yaml --out {{app_from_tools}}/spec/idmagic.models.schema.json
    cd {{ra_scl_tools}} && bun run scl-to-openapi -- --scl {{app_from_tools}}/spec/scl.yaml --out {{app_from_tools}}/spec/idmagic.openapi.json

# Start the Go API for local UI development.
dev-api:
    ADDR=:8081 ISSUER=http://localhost:5173 go run ./cmd/idmagic

# Start the React UI dev server.
dev-ui:
    cd ui && bun run dev

# Start the Docker Compose development stack.
dev-compose:
    docker compose -f deploy/docker/docker-compose.dev.yaml up --build
