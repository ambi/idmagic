---
status: completed
archived_from: 169a9dc9eb6453ccb8a38dc2d48f726ca201f3df
---

# tenancy コンテキストへバックエンド・コンテキストローカリティを横展開する

tenancy context を対象に ADR-089/090/091 の型紙を横展開シリーズ最終 WI として
貫通実装した。`Tenant`/`TenantStatus`/`PasswordPolicyOverride` 業務型と、それらの zog field
validation schema を `internal/shared/spec` から `tenancy/domain/` へ移設し、`shared/spec`
からは完全排除。`DefaultTenantID`/`DefaultRealm` の 2 定数は `shared/spec` の AuthZEN 述語
からも参照される import cycle 制約により `shared/kernel`（新設）へ昇格し、`tenancy/domain`
はそこからの re-export とした——本シリーズで初めて実際に kernel 昇格を要した具体例。
postgres/memory の repository 実装（`tenants.go`）を `tenancy/adapters/persistence/` へ
同居し、postgres 実装は sqlc 生成コード（動的クエリ 0 件、100% 静的）へ置換。
`tenancy.Module`（`identitymanagement.Module` と同型の「port の束」）を新設し、中央
`bootstrap/deps.go` の `Dependencies`・`shared/adapters/http/server/routes.go` の `Deps`
から `TenantRepo`/`AttrSchemaRepo` を集約（`server.Deps` は既存モジュール同様の legacy
互換ブリッジ付き）。9 依存元 context・bootstrap・shared/adapters 配下の計 100 以上の
ファイルを `tenancydomain.*`（tenancy 自身は無 alias `domain.*`）へ機械的に更新した。

全文は `git show 169a9dc9eb6453ccb8a38dc2d48f726ca201f3df:<このファイルのパス>` で読む。
