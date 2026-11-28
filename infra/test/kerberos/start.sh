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
# svc-sso は msDS-SupportedEncryptionTypes で AES に限る。svc-legacy は未設定のままにし、Samba はそのデフォルトとして
# RC4 でサービスチケットを出す。AES を有効にしていない AD のアカウントを、IdMagic が暗号方式の拒否として区別することを試すためである。
samba-tool user create svc-legacy 'Test-Legacy-Password-123!'
samba-tool spn add HTTP/legacy.example.com svc-legacy
ldbmodify -H /var/lib/samba/private/sam.ldb <<'LDIF'
dn: CN=svc-sso,CN=Users,DC=example,DC=com
changetype: modify
replace: msDS-SupportedEncryptionTypes
msDS-SupportedEncryptionTypes: 24
LDIF
samba-tool domain exportkeytab /shared/legacy.keytab --principal=HTTP/legacy.example.com
# 待ち合わせは http.keytab の有無で判断するので、最後に書く。
samba-tool domain exportkeytab /shared/http.keytab --principal=HTTP/sso.example.com
exec samba --interactive --no-process-group --debug-stdout --debuglevel=1
