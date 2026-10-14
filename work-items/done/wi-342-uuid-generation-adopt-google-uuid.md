---
status: completed
archived_from: 4cc593ff19e8aac14703224ccdd1b632f5fafd1e
---

# 自前 UUIDv4 生成 (`spec.NewUUIDv4`) を `google/uuid` に置き換える

`backend/shared/spec/uuid.go` の `NewUUIDv4` を、`crypto/rand` 直叩きの自前実装から
`github.com/google/uuid` の `uuid.NewRandom().String()` ベースに置き換えた。
`github.com/google/uuid` はすでに他箇所(`shared/storage/db_postgres`,
`authentication/federation/handlers_http`, `saml/handlers_http`)で使われている
go.mod 直接依存だったため、依存追加作業は不要だった。シグネチャ
`(string, error)` および生成される UUID のフォーマット(v4, RFC 4122)は不変。
`spec.NewUUIDv4()` を呼ぶ約40箇所は無改修で動作する。

全文は `git show 4cc593ff19e8aac14703224ccdd1b632f5fafd1e:<このファイルのパス>` で読む。
