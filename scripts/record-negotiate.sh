#!/usr/bin/env bash
# 隔離した Samba AD を起動し、手元の Chrome が送る Negotiate のトークンを記録する（mise run record-negotiate）。
# 記録は手元の意図的な作業であり、CI のゲートには入れない。docker、待受け、Chrome を使うので、サンドボックスの外で実行する。
set -euo pipefail

chrome="${CHROME_BINARY:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
if [[ ! -x "$chrome" ]]; then echo "Chrome is not found at $chrome; set CHROME_BINARY" >&2; exit 2; fi

fixture_dir=$(mktemp -d "${TMPDIR:-/tmp}/idmagic-record-negotiate.XXXXXX")
container_id=''
cleanup() {
  if [[ -n "$container_id" ]]; then docker rm -f "$container_id" >/dev/null 2>&1 || true; fi
  rm -rf "$fixture_dir"
}
trap cleanup EXIT

docker build --quiet --tag idmagic-kerberos-test infra/test/kerberos >/dev/null
container_id=$(docker run --detach --publish 127.0.0.1::88/tcp idmagic-kerberos-test)
for ((attempt=0; attempt<90; attempt++)); do
  if docker exec "$container_id" test -s /shared/http.keytab; then break; fi
  sleep 1
done
docker exec "$container_id" test -s /shared/http.keytab
docker cp "$container_id:/shared/http.keytab" "$fixture_dir/http.keytab" >/dev/null
KERBEROS_KDC_ADDR=$(docker port "$container_id" 88/tcp)
export KERBEROS_KDC_ADDR CHROME_BINARY="$chrome" KERBEROS_KEYTAB_FILE="$fixture_dir/http.keytab"
go test -tags kerberosrecord -count=1 -run '^TestRecordChromeNegotiate$' -v ./backend/authentication/internal/negotiate/protocol_kerberos
