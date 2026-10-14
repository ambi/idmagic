---
status: completed
archived_from: c4dda9e1ee528d6bb48824f7089561648c6174cc
---

# application コンテキストでバックエンド・コンテキストローカリティ回復を貫通実装する（パイロット）

application context を対象に ADR-089/090/091 の 4 レバーを貫通実装した。
業務型を `internal/application/domain/` へ移設し `internal/shared/spec` から排除、
postgres/memory の repository 実装を `internal/application/adapters/persistence/` へ同居、
application の postgres 実装を sqlc 生成コード（`sqlcgen` package、動的クエリ 1 件のみ手書き
pgx エスケープハッチ）へ置換、`application.Module` を新設して中央 `server/routes.go` の
`Deps` と `bootstrap/deps.go` の `Dependencies` から application 由来 7 field を 1 field
(`Application application.Module`) へ集約した。副産物として `shared/adapters/persistence/postgres`
の `RowScanner` / `shared/adapters/persistence/memory` の `TenantKey` を export し、
per-context postgres テスト向けの再利用可能な `pgtest`(embedded-postgres harness) /
`pgfixtures`(Tenant/User/Group/Client シード) package を新設した。
ADR-090 に動的クエリ比率の実測値(96% 静的 / 4% エスケープハッチ)を追記し sqlc 継続を確認した。

全文は `git show c4dda9e1ee528d6bb48824f7089561648c6174cc:<このファイルのパス>` で読む。
