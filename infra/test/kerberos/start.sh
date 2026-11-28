#!/usr/bin/env bash
set -euo pipefail

mkdir -p /shared

# テスト用の隔離した AD。これらは実環境の資格情報ではない。
if [[ -e /etc/samba/smb.conf ]]; then
  mv /etc/samba/smb.conf /etc/samba/smb.conf.packaged
fi
tls_options=()
# LDAPS を検証するテストは、SAN を持つ証明書を PEM の環境変数で渡す。Samba が自動で作る証明書は
# CN だけを持ち、SAN を見る TLS の検証を通らないためである。渡さなければ自動の証明書のままにする。
if [[ -n "${TLS_CERT_PEM:-}" ]]; then
  mkdir -p /var/lib/samba/private/tls-test
  printf '%s\n' "$TLS_CERT_PEM" >/var/lib/samba/private/tls-test/cert.pem
  printf '%s\n' "$TLS_KEY_PEM" >/var/lib/samba/private/tls-test/key.pem
  printf '%s\n' "$TLS_CA_PEM" >/var/lib/samba/private/tls-test/ca.pem
  chmod 600 /var/lib/samba/private/tls-test/key.pem
  tls_options=(--option='tls enabled = yes'
    --option='tls certfile = /var/lib/samba/private/tls-test/cert.pem'
    --option='tls keyfile = /var/lib/samba/private/tls-test/key.pem'
    --option='tls cafile = /var/lib/samba/private/tls-test/ca.pem')
fi
samba-tool domain provision --realm=EXAMPLE.COM --domain=EXAMPLE --server-role=dc \
  --dns-backend=SAMBA_INTERNAL --adminpass='Test-Admin-Password-123!' \
  --host-name=dc --option='posix:eadb = /var/lib/samba/private/eadb.tdb' "${tls_options[@]}"
samba-tool user create alice 'Test-Alice-Password-123!'
samba-tool user create svc-sso 'Test-Service-Password-123!'
samba-tool spn add HTTP/sso.example.com svc-sso
samba-tool domain exportkeytab /shared/http.keytab --principal=HTTP/sso.example.com
exec samba --interactive --no-process-group --debug-stdout --debuglevel=1
