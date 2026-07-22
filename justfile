# App repository command map for humans and AI agents.
#
# This app repo consumes RA/SCL tools from the embedded tools/ directory.

set shell := ["zsh", "-cu"]

ra_cmd := env("RA_CMD", "bun run tools/ra/src/main.ts")
go_cache := env("GOCACHE", "/tmp/idmagic-go-cache")
golangci_cache := env("GOLANGCI_LINT_CACHE", "/tmp/idmagic-golangci-cache")
git_commit := `git rev-parse HEAD 2>/dev/null || echo "unknown"`
build_date := `date -u +'%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "unknown"`
version := env("VERSION", "0.0.0-dev")
ldflags := "-X github.com/ambi/idmagic/backend/shared/version.Version=" + version + " -X github.com/ambi/idmagic/backend/shared/version.GitCommit=" + git_commit + " -X github.com/ambi/idmagic/backend/shared/version.BuildDate=" + build_date

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
    cd frontend && bun install --frozen-lockfile

# Run the standard app verification suite.
verify: yaml-check traceability-strict test-tools typecheck-tools verify-go verify-ui

# Report workspace traceability findings without failing on unbaselined debt.
traceability-report:
    {{ra_cmd}} traceability --json --revision={{git_commit}}

# Reject unbaselined traceability drift and expired debt baselines.
traceability-strict:
    {{ra_cmd}} traceability --strict --json --revision={{git_commit}}

# Run embedded RA/SCL tooling tests.
test-tools:
    cd tools && bun test

# Type-check embedded RA/SCL tooling.
typecheck-tools:
    cd tools && bun run typecheck

# Format embedded RA/SCL tooling.
format-tools:
    cd tools && bun run format

# Lint embedded RA/SCL tooling.
lint-tools:
    cd tools && bun run lint

# Verify Go backend with lint and race-enabled tests.
verify-go: lint-go test-go-race

# Run Go lint.
lint-go:
    GOCACHE=/tmp/idmagic-go-cache GOLANGCI_LINT_CACHE={{golangci_cache}} golangci-lint run ./...

# Clear only the repository-local temporary linter cache when it retains paths
# from a removed worktree and makes a fresh lint run unreliable.
clean-lint-cache:
    rm -rf {{golangci_cache}}

# Format Go backend code.
format-go:
     GOCACHE=/tmp/idmagic-go-cache GOLANGCI_LINT_CACHE={{golangci_cache}} golangci-lint fmt ./...

# Run Go tests.
test-go:
    GOCACHE={{go_cache}} go test ./...

# Run Go tests for one package during a layer-local red/green cycle.
test-go-package package:
    GOCACHE={{go_cache}} go test {{package}}

# Plan or apply an explicit environment seed. Example: just seed development development dry_run
seed environment profile mode="dry_run" manifest="" count="0":
    GOCACHE={{go_cache}} go run ./backend/cmd/idmagic-seed --environment {{environment}} --profile {{profile}} --mode {{mode}} --manifest "{{manifest}}" --count {{count}}

# Opt-in throughput measurement. The count must remain within the seed safety policy.
seed-throughput environment="development" count="10000" batch_size="250":
    GOCACHE={{go_cache}} go run ./backend/cmd/idmagic-seed --environment {{environment}} --profile performance --mode apply --count {{count}} --batch-size {{batch_size}}

# Run one externally scheduled, one-shot operational batch locally.
batch task *args:
    GOCACHE={{go_cache}} go run ./backend/cmd/idmagic-batch {{task}} {{args}}

# Synchronize Go module requirements and checksums.
go-mod-tidy:
    GOCACHE={{go_cache}} go mod tidy

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

# Regenerate sqlc-generated postgres query code from sqlc.yaml (ADR-090).
sqlc-generate:
    sqlc generate

# Verify UI with format check, lint, typecheck, and build.
verify-ui: format-check-ui lint-ui typecheck-ui test-ui-unit build-ui

# Run UI format check.
format-check-ui:
    cd frontend && bun run format:check

# Format UI.
format-ui:
    cd frontend && bun run format

# Run UI lint.
lint-ui:
    cd frontend && bun run lint

# Run UI typecheck.
typecheck-ui:
    cd frontend && bun run typecheck

# Run UI unit tests.
test-ui-unit:
    cd frontend && bun run test:unit

# Run UI unit tests with coverage.
test-ui-cover:
    cd frontend && bun run test:unit:coverage

# Build UI.
build-ui:
    cd frontend && bun run build

# Run UI E2E tests.
test-ui-e2e:
    cd frontend && bun run test:e2e

# Validate SCL and Work Item YAML.
yaml-check: yaml-check-scl yaml-check-work-items check-ids yaml-check-architecture yaml-check-traceability

# Validate SCL YAML files.
yaml-check-scl:
    {{ra_cmd}} yaml-check --scl

# Validate Work Item YAML files.
yaml-check-work-items:
    {{ra_cmd}} yaml-check --work-items

# Detect duplicate / mismatched change-record ids.
check-ids:
    {{ra_cmd}} yaml-check --ids

# Validate ARCHITECTURE.md against the workspace it describes.
yaml-check-architecture:
    {{ra_cmd}} yaml-check --architecture

# Validate traceability manifest and execution evidence YAML.
yaml-check-traceability:
    {{ra_cmd}} yaml-check --traceability

# Regenerate SCL-derived artifacts.
scl-render:
    {{ra_cmd}} render

# Regenerate only embedded tool SCL HTML artifacts.
scl-render-tools:
    {{ra_cmd}} render --tools-only

# Start the local dev stack (Go API + React UI together with live reload).
dev:
    ./dev.sh

# Start the lightweight API + UI stack without durable jobs.
dev-memory:
    ./dev.sh memory

# Start the Go API for local UI development.
dev-api:
    ADDR=:8081 ISSUER=http://localhost:5173 WEBAUTHN_RP_ID="${WEBAUTHN_RP_ID:-localhost}" WEBAUTHN_RP_ORIGINS="${WEBAUTHN_RP_ORIGINS:-http://localhost:5173}" WEBAUTHN_RP_DISPLAY_NAME="${WEBAUTHN_RP_DISPLAY_NAME:-IdMagic Local}" go run ./backend/cmd/idmagic

# Start the React UI dev server.
dev-ui:
    cd frontend && bun run dev

# Start the Docker Compose development stack.
dev-compose:
    docker compose -f infra/docker/docker-compose.dev.yaml up --build

# Validate the Docker Compose development stack configuration.
check-compose:
    docker compose -f infra/docker/docker-compose.dev.yaml config --quiet

# Render and schema-validate one Kubernetes environment overlay. Image digests
# in production are release placeholders until the release pipeline supplies them.
check-k8s overlay="dev":
    docker run --rm -v "{{justfile_directory()}}:/workspace:ro" -w /workspace registry.k8s.io/kustomize/kustomize:v5.6.0 build infra/k8s/overlays/{{overlay}} | docker run --rm -i ghcr.io/yannh/kubeconform:v0.6.7 -strict -summary

# Apply a validated Kubernetes environment overlay. Secrets and a real release
# digest must exist before using the production overlay.
deploy-k8s overlay="dev":
    kubectl apply -k infra/k8s/overlays/{{overlay}}

# Recover the prior Kubernetes Deployment revision after checking its cause.
rollback-k8s deployment="idmagic-api":
    kubectl rollout undo deployment/{{deployment}}

# Validate the Prometheus input rules and Grafana JSON before packaging their
# Kubernetes consumers. Prometheus Operator CRDs are intentionally optional.
check-monitoring:
    docker run --rm --entrypoint promtool -v "{{justfile_directory()}}:/workspace:ro" prom/prometheus:v2.55.1 check rules /workspace/infra/docker/prometheus-rules.yml
    jq empty infra/docker/grafana-dashboard.json
    docker run --rm -v "{{justfile_directory()}}:/workspace:ro" -w /workspace registry.k8s.io/kustomize/kustomize:v5.6.0 build infra/k8s/monitoring > /dev/null
    docker run --rm -v "{{justfile_directory()}}:/workspace:ro" -w /workspace registry.k8s.io/kustomize/kustomize:v5.6.0 build infra/k8s/monitoring/operator > /dev/null

# Apply monitoring assets; ServiceMonitor remains opt-in for clusters with
# Prometheus Operator installed.
deploy-monitoring:
    kubectl apply -k infra/k8s/monitoring

deploy-monitoring-operator:
    kubectl apply -k infra/k8s/monitoring/operator

# Execute the tenant-local OAuth smoke against a deliberately seeded target.
# Compose users should pass host.docker.internal; Linux CI should pass its service URL.
k6-smoke base_url="http://host.docker.internal:8080/realms/default" browser_origin="http://localhost:8080":
    docker run --rm -e IDMAGIC_BASE_URL={{base_url}} -e IDMAGIC_BROWSER_ORIGIN={{browser_origin}} -v "{{justfile_directory()}}/load/k6:/scripts:ro" grafana/k6:0.54.0 run /scripts/oauth-smoke.js

# Parse and inspect the k6 module without sending traffic to a target.
check-k6:
    docker run --rm -v "{{justfile_directory()}}/load/k6:/scripts:ro" grafana/k6:0.54.0 inspect /scripts/oauth-smoke.js

# Run the OAuth2 / OIDC demo against a running server (default http://localhost:8080).
demo:
    ./demo.sh
