---
id: wi-118-simplify-internal-http-dependencies-and-scim-layout
title: "Simplify internal HTTP dependencies and align SCIM layout"
created_at: 2026-07-04
authors: [tn]
status: pending
risk: medium
---

# Motivation
`internal` 配下の bounded context 分割は維持できているが、HTTP adapter の依存が
`internal/shared/adapters/http/support.Deps` に集中し、context ごとの責務境界が読み取りにくく
なっている。特に `support` が HTTP 横断部品だけでなく repository、usecase、token issuer、
health probe 用関数まで保持しているため、変更時に影響範囲を局所化しにくい。

また `internal/scim` は他 context と異なり、handler、route、usecase、model が直下に並ぶ構成で、
新しい作業者や AI agent が既存の context 構造から類推しにくい。HTTP 依存と SCIM 構造を整理し、
今後の機能追加時に小さな単位で変更できる状態にする。

# Scope
- `internal/shared/adapters/http/support.Deps` を廃止または HTTP 横断設定だけの小さな型に縮小する。
- 各 context の HTTP adapter に専用 deps 型を定義し、handler が実際に使う依存だけを渡す。
- `internal/shared/adapters/http/server.Register` で context ごとの deps を組み立て、各 `RegisterRoutes` に渡す。
- `support` には request id、recover、response/error helpers、tenant middleware、security headers、CSRF、cancellation などの HTTP 横断部品だけを残す。
- `internal/scim` を他 context と同じ `adapters/http`、`domain`、`ports`、`usecases` 構造へ揃える。
- `shared/adapters` の移動先名は `internal/infra` を第一候補として記録するが、この work item では実移動しない。
- SCL の公開仕様、HTTP route、HTTP method、JSON shape、認可挙動、永続化 schema は変更しない。

# Out of Scope
- `bootstrap` の分解。
- `contexts/` ルートへの全面移動。
- `shared/adapters` の `internal/infra` への実移動。
- API 仕様、SCL、DB schema、認可ポリシー、UI 挙動の変更。
- 動作改善や機能追加を伴う refactor。

# Verification
- `go test ./...`
- `just yaml-check-work-items`
- `just check-ids`
- HTTP handler tests と server route tests が、route path と response shape を変えずに通ることを確認する。
- SCIM 既存 tests が、package 移動後も同じ振る舞いで通ることを確認する。

# Risk Notes
HTTP route 登録と DI の組み替えを触るため、compile error や handler test の fixture 更新が広範囲に出る可能性がある。公開挙動を変えない構造整理として進め、context ごとの deps 分割、SCIM 移動、将来の `internal/infra` 命名記録を別々に確認できる小さな差分へ分けることでリスクを下げる。
