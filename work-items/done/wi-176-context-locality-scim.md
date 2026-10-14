---
status: completed
archived_from: 3eee0774dfa7d90aac7b1dcf9c7d474b17d7f5ef
---

# scim コンテキストへバックエンド・コンテキストローカリティを横展開する

`scim_models.go` は元々 `shared/spec` 非依存で domain 移設は不要と確認した。
`shared/adapters/persistence/{postgres,memory}` の SCIM token/user-ref/group-ref
repository を `backend/scim/adapters/persistence/{postgres,memory}` へ同居し、postgres 側
の全 12 クエリを sqlc 生成（`sqlcgen`）へ置換した。`scim/module.go` を新設して
`Deps`/`bootstrap` から `ScimRepo` フィールドを撤去し、`scim.Module{Repo: ...}` 経由の
DI・ルート登録に統一した（IdentityManagement の HTTP handler が必要とする `ScimRepo` は
`d.Scim.Repo` として同一の port 型で受け渡す）。SCIM `filter` query parameter は
UserRepository 側のアプリケーション内フィルタで処理されており、scim 自身の repository に
可変 WHERE クエリは存在しなかった。振る舞い・HTTP route・DB schema・公開 API は変更していない。

全文は `git show 3eee0774dfa7d90aac7b1dcf9c7d474b17d7f5ef:<このファイルのパス>` で読む。
