# App repository command map for humans and AI agents.
#
# Repository tooling is embedded under tools/ and exposed through this command map.


golangci_cache := env("GOLANGCI_LINT_CACHE", "/tmp/idmagic-golangci-cache")
git_commit := `git rev-parse HEAD 2>/dev/null || echo "unknown"`
build_date := `date -u +'%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "unknown"`
version := env("VERSION", "0.0.0-dev")
ldflags := "-X github.com/ambi/idmagic/backend/shared/version.Version=" + version + " -X github.com/ambi/idmagic/backend/shared/version.GitCommit=" + git_commit + " -X github.com/ambi/idmagic/backend/shared/version.BuildDate=" + build_date

# Show this command map.
default:
    @just --list

# Install local dependencies and setup repository tools.
setup: setup-tools install-ui

# Setup repository tool dependencies and link agent skills.
setup-tools:
    cd tools && bun install
    ln -sfn tools/node_modules node_modules
    mkdir -p .agents/skills
    mkdir -p .claude
    ln -sfn ../.agents/skills .claude/skills

# Compile the TypeSpec product contract and emit OpenAPI.
compile-spec:
    cd tools && bunx tsp compile ../spec/main.tsp --config ../spec/tspconfig.yaml

# Generate runtime route metadata from the compiled TypeSpec OpenAPI document.
generate-contract:
    cd tools && bun run generate-contract/src/main.ts

# Check that generated runtime route metadata matches TypeSpec.
check-generated-contract:
    cd tools && bun run generate-contract/src/main.ts --check


# Install UI dependencies.
install-ui:
    cd frontend && bun install --frozen-lockfile

# Run the standard app verification suite. Every leaf check runs in parallel and
# is timed individually; a duration-sorted table prints at the end so the current
# bottleneck (and whether the grouping still makes sense) is always visible.
verify:
    #!/usr/bin/env sh
    set -u
    checks="check check-api-compat test-tools typecheck-tools lint-go test-go format-check-ui lint-ui test-ui-unit build-ui"
    tmp=$(mktemp -d)
    t0=$(date +%s)
    for c in $checks; do
        ( s=$(date +%s)
          just "$c" >"$tmp/$c.log" 2>&1
          r=$?
          e=$(date +%s)
          dur=$(( e - s ))
          printf '%s %d %d\n' "$c" "$r" "$dur" >"$tmp/$c.meta"
        ) &
    done
    wait
    total=$(( $(date +%s) - t0 ))
    rc=0
    for c in $checks; do
        read -r _ cstatus _ <"$tmp/$c.meta"
        if [ "$cstatus" -ne 0 ]; then
            rc=1
            echo "===== FAILED: $c ====="
            cat "$tmp/$c.log"
        fi
    done
    echo ""
    echo "── verify timings (all checks run in parallel) ──"
    for c in $checks; do
        read -r cname cstatus cdur <"$tmp/$c.meta"
        [ "$cstatus" -eq 0 ] && mark="ok  " || mark="FAIL"
        printf '%d %s %s\n' "$cdur" "$mark" "$cname"
    done | sort -rn | while read -r d mark n; do
        printf '  %3ds  %s  %s\n' "$d" "$mark" "$n"
    done
    printf '  %3ds  ── total wall clock (sum if serial: run `just verify-serial`)\n' "$total"
    rm -rf "$tmp"
    exit $rc

# Run the full verification suite serially (clean ordered output; slower, no timings).
verify-serial: check check-api-compat test-tools typecheck-tools lint-go test-go-race format-check-ui lint-ui test-ui-unit build-ui

# Validate specifications, then test and type-check embedded tooling.
verify-spec: check test-tools typecheck-tools

# Run embedded repository tooling tests.
test-tools:
    cd tools && bun test

# Type-check embedded repository tooling.
typecheck-tools:
    cd tools && bun run typecheck

# Format embedded repository tooling.
format-tools:
    cd tools && bun run format

# Lint embedded repository tooling.
lint-tools:
    cd tools && bun run lint

# Verify Go backend with lint and race-enabled tests.
verify-go: lint-go test-go-race

# Run Go lint.
lint-go:
    GOLANGCI_LINT_CACHE={{golangci_cache}} golangci-lint run ./...

# Clear only the repository-local temporary linter cache when it retains paths
# from a removed worktree and makes a fresh lint run unreliable.
clean-lint-cache:
    rm -rf {{golangci_cache}}

# Format Go backend code.
format-go:
     GOLANGCI_LINT_CACHE={{golangci_cache}} golangci-lint fmt ./...

# Run Go tests.
test-go:
    go test ./...

# Run Go tests for one package during a layer-local red/green cycle.
test-go-package package:
    go test {{package}}

# Plan or apply an explicit environment seed. Example: just seed development development dry_run
seed environment profile mode="dry_run" manifest="" count="0":
    go run ./backend/cmd/idmagic-seed --environment {{environment}} --profile {{profile}} --mode {{mode}} --manifest "{{manifest}}" --count {{count}}

# Opt-in throughput measurement. The count must remain within the seed safety policy.
seed-throughput environment="development" count="10000" batch_size="250":
    go run ./backend/cmd/idmagic-seed --environment {{environment}} --profile performance --mode apply --count {{count}} --batch-size {{batch_size}}

# Run one externally scheduled, one-shot operational batch locally.
batch task *args:
    go run ./backend/cmd/idmagic-batch {{task}} {{args}}

# Synchronize Go module requirements and checksums.
go-mod-tidy:
    go mod tidy

# Run race-enabled Go tests.
test-go-race:
    go test -race ./...

# Run Go tests with coverage.
test-go-cover:
    go test -coverprofile=coverage.out -covermode=atomic ./...
    go tool cover -func=coverage.out

# Run Go fuzz targets for a package.
test-go-fuzz package fuzztime="30s":
    go test -run=Fuzz -fuzz=Fuzz -fuzztime={{fuzztime}} {{package}}

# Build all Go packages.
build-go:
    go build -ldflags '{{ldflags}}' ./...

# Regenerate sqlc-generated postgres query code from sqlc.yaml.
sqlc-generate:
    sqlc generate

# Verify UI with format check, lint, unit tests, and build. build-ui runs tsc, so
# a separate typecheck-ui step would only duplicate the type check.
verify-ui: format-check-ui lint-ui test-ui-unit build-ui

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

# Run one UI unit-test file.
test-ui-unit-file file:
    cd frontend && bun test {{file}}

# Run UI unit tests with coverage.
test-ui-cover:
    cd frontend && bun run test:unit:coverage

# Build UI.
build-ui:
    cd frontend && bun run build

# Run UI E2E tests.
test-ui-e2e:
    cd frontend && bun run test:e2e

# Validate specification sources, records, and forbidden dependencies.
check: check-spec check-work-items check-ids check-boundaries

# Compile TypeSpec and validate canonical specification documents.
check-spec: compile-spec check-generated-contract
    cd tools && bun run workspace/src/check-workspace.ts --documents
    cd tools && bun run render-spec-docs/src/main.ts --check

# Validate work-item records (Markdown with YAML frontmatter).
check-work-items:
    cd tools && bun run workspace/src/check-workspace.ts --work-items

# Detect duplicate / mismatched change-record ids.
check-ids:
    cd tools && bun run workspace/src/check-workspace.ts --ids

# Reject outward source dependencies inferred directly from repository paths.
check-boundaries:
    cd tools && bun run check/src/check-boundaries.ts

# Detect breaking changes vs the frozen OpenAPI release baseline.
check-api-compat:
    cd tools && bun run check-api-compat/src/main.ts

# Render the browsable specification and API documentation from canonical sources.
render-spec-docs:
    cd tools && bun run render-spec-docs/src/main.ts

# Regenerate standard artifacts from TypeSpec. Generated files are untracked.
spec-render: compile-spec generate-contract render-spec-docs

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

# Start the Docker Compose development stack, detached.
dev-compose:
    docker compose -f infra/docker/docker-compose.dev.yaml up --build -d

# Follow logs for the Docker Compose development stack.
logs-compose:
    docker compose -f infra/docker/docker-compose.dev.yaml logs -f

# Stop and remove the Docker Compose development stack.
down-compose:
    docker compose -f infra/docker/docker-compose.dev.yaml down

# Re-apply infra/schema/postgres.sql without recreating the rest of the stack.
schema-compose:
    docker compose -f infra/docker/docker-compose.dev.yaml run --rm schema

# Validate the Docker Compose development stack configuration.
check-compose:
    docker compose -f infra/docker/docker-compose.dev.yaml config --quiet

# Verify infra/schema/postgres.sql converges under psqldef against an empty
# database (apply -> dry-run -> apply -> dry-run, both dry-runs expected to be
# no-ops). Runs in an isolated, disposable compose project (wi-308).
check-schema:
    ./infra/schema/check-convergence.sh

# Take a pg_dump backup (custom format + sha256 checksum). Requires
# PGHOST/PGPORT/PGUSER/PGPASSWORD/PGDATABASE to be exported (wi-101).
backup-postgres output_dir:
    ./infra/backup/backup-postgres.sh {{output_dir}}

# Restore a pg_dump backup into an empty target database. db_name must match
# $PGDATABASE exactly (non-production guard, wi-101).
restore-postgres backup_file db_name:
    ./infra/backup/restore-postgres.sh {{backup_file}} --yes-restore-into-this-database {{db_name}}

# Run a full local backup -> simulated db loss -> restore -> consistency-check
# drill against a disposable compose project, and print elapsed time as a
# local RPO/RTO estimate (wi-101).
restore-drill:
    ./infra/backup/restore-drill.sh

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
    docker run --rm -v "{{justfile_directory()}}:/workspace:ro" -w /workspace registry.k8s.io/kustomize/kustomize:v5.6.0 build infra/k8s/monitoring/loki > /dev/null

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
