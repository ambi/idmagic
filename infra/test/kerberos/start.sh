#!/usr/bin/env bash
set -euo pipefail

mkdir -p /shared

# テスト用の隔離した AD。これらは実環境の資格情報ではない。
if [[ -e /etc/samba/smb.conf ]]; then
  mv /etc/samba/smb.conf /etc/samba/smb.conf.packaged
fi
samba-tool domain provision --realm=EXAMPLE.COM --domain=EXAMPLE --server-role=dc \
  --dns-backend=SAMBA_INTERNAL --adminpass='Test-Admin-Password-123!' \
  --host-name=dc --option='posix:eadb = /var/lib/samba/private/eadb.tdb'
samba-tool user create alice 'Test-Alice-Password-123!'
samba-tool user create svc-sso 'Test-Service-Password-123!'
samba-tool spn add HTTP/sso.example.com svc-sso
samba-tool domain exportkeytab /shared/http.keytab --principal=HTTP/sso.example.com
exec samba --interactive --no-process-group --debug-stdout --debuglevel=1
