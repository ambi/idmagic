---
status: completed
archived_from: 9bac219c947bb6e09ee90a1304dc6ecf95c79a6a
---

# identity-management コンテキストへバックエンド・コンテキストローカリティを横展開する

identity-management context を対象に ADR-089/090/091/093 の 4 レバーを
貫通実装した。`User`/`Group`/`Agent`/`AttributeValue`/`UserAttributeDef`/
`TenantUserAttributeSchema` 業務型と、それらが所有する zog field validation schema
(ADR-093)・enum (`UserStatus`/`RequiredAction`/`AttributeType`/`AttrVisibility`/
`AgentKind`/`AgentStatus`) を `internal/shared/spec` から `identitymanagement/domain/`
へ移設し、`shared/spec` からは完全排除。postgres/memory の repository 実装を
`identitymanagement/adapters/persistence/` へ同居し、postgres 実装は sqlc 生成コード
（動的クエリ 0 件、100% 静的）へ置換。`identitymanagement.Module`（`authentication.Module`
と同型の「port の束」パターン）を新設し、中央 `bootstrap/deps.go` の
`Dependencies`・`shared/adapters/http/server/routes.go` の `Deps` から
`UserRepo`/`GroupRepo`/`AgentRepo` 3 field を `IdentityManagement
identitymanagement.Module` 1 field へ集約した（`server.Deps` 側は
`authentication.Module`/`oauth2.Module` と同じ legacy 互換ブリッジを付与）。
リポジトリ全体で `spec.User` 等参照 103+1 ファイル・`memory.New*Repository` 44 ファイルを
機械的に `idmdomain.*`/`idmmemory.*` へ置換し、T007（第 2 波 import 更新）を T001 と
同時に実施した。副産物として `shared/adapters/persistence/postgres/pgfixtures` の
`SeedUser`/`SeedGroup` を identitymanagement 実装へ向け直し、scim/oauth2 の内部
postgres テストが直接参照していた旧 `sharedpg.UserRepository`/`GroupRepository` も
`idmpg.*` へ更新した。ADR-090 に動的クエリ比率の実測値（identity-management: 100%
静的）を追記し、`ARCHITECTURE.md` の「`identitymanagement` は per-context `domain/` を
持たない」という記述（now-stale）を是正した。

全文は `git show 9bac219c947bb6e09ee90a1304dc6ecf95c79a6a:<このファイルのパス>` で読む。
