---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# idmagic の package 構成と scl.yaml components の整合性を取る

SCL components を Tenancy / Authentication / OAuth2 の 3 つに再編し、
idmagic の package 構造と 1:1 に対応させた。Go 側は invented vocabulary の
internal/administration/ を解体し、Authentication / OAuth2 配下に再配置した。
LoginSession を扱う SessionStore port を SCL の所有関係に合わせて
oauth2/ports から authentication/ports へ移動した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
