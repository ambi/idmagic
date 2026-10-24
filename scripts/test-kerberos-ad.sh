#!/usr/bin/env bash
set -euo pipefail

if [[ $# -gt 2 ]]; then echo 'usage: test-kerberos-ad.sh [docker-config-directory] [docker-host]' >&2; exit 2; fi
if [[ -n "${1:-}" ]]; then export DOCKER_CONFIG="$1"; fi
if [[ -n "${2:-}" ]]; then export DOCKER_HOST="$2"; fi

fixture_dir=$(mktemp -d "${TMPDIR:-/tmp}/idmagic-kerberos.XXXXXX")
container_id=''
cleanup() {
  if [[ -n "$container_id" ]]; then docker rm -f "$container_id" >/dev/null 2>&1 || true; fi
  rm -rf "$fixture_dir"
}
trap cleanup EXIT

docker build --quiet --tag idmagic-kerberos-test infra/test/kerberos
container_id=$(docker run --detach --publish 127.0.0.1::88/tcp idmagic-kerberos-test)
for ((attempt=0; attempt<90; attempt++)); do
  if docker exec "$container_id" test -s /shared/http.keytab; then break; fi
  if [[ "$(docker inspect --format '{{.State.Running}}' "$container_id")" != true ]]; then
    docker logs "$container_id"
    exit 1
  fi
  sleep 1
done
if ! docker exec "$container_id" test -s /shared/http.keytab; then
  docker logs "$container_id"
  echo 'Samba AD did not become ready.' >&2
  exit 1
fi
docker cp "$container_id:/shared/http.keytab" "$fixture_dir/http.keytab"
KERBEROS_KDC_ADDR=$(docker port "$container_id" 88/tcp)
export KERBEROS_KDC_ADDR
export KERBEROS_KEYTAB_FILE="$fixture_dir/http.keytab"
mise run test-go-test -- ./backend/authentication/internal/negotiate/protocol_kerberos TestVerifierAgainstSambaAD
