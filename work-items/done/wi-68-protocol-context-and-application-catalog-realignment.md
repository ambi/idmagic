---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Protocol bounded context と Application Catalog の境界を整理する

SCL / ADR / 実装ディレクトリ / 管理 UI の protocol 境界を ADR-064 に合わせて整理した。
`Federation` bounded context 名を廃止し、WS-Federation passive と WS-Trust active STS は
`WsFederation` に集約した。claim mapping / NameID / attribute release は `ClaimRelease`、
key material lifecycle / JWKS 管理は `SigningKeys`、運用者向け上位概念は将来の
`ApplicationCatalog` として SCL に反映した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
