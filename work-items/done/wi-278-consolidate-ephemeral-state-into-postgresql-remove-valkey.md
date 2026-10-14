---
status: completed
archived_from: 7df29b72d842198d17f07c29396ea4261e9f9f0e
---

# 揮発性の認証/OAuth2 一時状態を全て PostgreSQL に統合し、Valkey を廃止する

揮発性の認証 / OAuth2 一時状態 9 ストア（authorization request / code / PAR / device code /
JTI replay（DPoP・client assertion）/ access-token denylist / WebAuthn challenge /
login throttle / SAML AuthnRequest replay）を全て PostgreSQL に統合し、Valkey を
コード・依存・インフラから完全撤去した（ADR-139）。port 契約は不変のまま context ごとに
`db_postgres` アダプタを新設（SETNX→`INSERT ON CONFLICT`、Lua CAS→`UPDATE ... RETURNING`、
GetDel→`DELETE ... RETURNING`、WATCH→tx＋`SELECT FOR UPDATE`）。TTL の正しさは全 read の
`expires_at > now()` 述語が担保し、GC は `idmagic-worker` の周期 `ephemeralSweepLoop`（60s）で
best-effort に空間回収する。bootstrap を `assemblePostgres` に切替え、`PERSISTENCE` は
`postgres` に統一（旧 `postgres_valkey` は alias を残さず削除）、readiness は Postgres 単一依存に。
SCL（system/authentication/jobs）・スキーマ（9 テーブル、UNLOGGED/LOGGED 別、暫定）・
ARCHITECTURE・README・GCP デプロイ資材・dev 環境（miniredis 撤去）・`go.mod`
（go-redis/miniredis 除去）を同期。UNLOGGED/LOGGED と GC 間隔の staging 実測確定（旧 T013）は
[[wi-282]] に分離した。

全文は `git show 7df29b72d842198d17f07c29396ea4261e9f9f0e:<このファイルのパス>` で読む。
