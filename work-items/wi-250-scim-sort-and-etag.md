---
status: pending
authors: [tn]
risk: low
created_at: 2026-07-18
depends_on: []
change_kind: feature
initial_context:
  scl:
    Scim:
      - interfaces.ListScimUsers
      - interfaces.ListScimGroups
      - interfaces.UpdateScimUser
      - interfaces.PatchScimUser
      - interfaces.GetScimServiceProviderConfig
  source:
    - backend/scim/usecases/list.go
    - backend/scim/adapters/http/handlers.go
  tests:
    - backend/scim/adapters/http/scim_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Scim, kind: interface, element: ListScimUsers }
  - { context: Scim, kind: interface, element: UpdateScimUser }
  - { context: Scim, kind: interface, element: GetScimServiceProviderConfig }
---

# SCIM sortBy/sortOrder と ETag (楽観的並行性制御) に対応する

## Motivation

RFC 7644 §3.4.2.3 の `sortBy`/`sortOrder` query parameter と §3.14 の ETag による
楽観的並行性制御は、いずれも `ServiceProviderConfig` で明示的に非対応(`sort.supported:
false`、`etag.supported: false`)を広告済みだが実装されていない。sort は
クライアント側の一覧処理を簡単にし、ETag は concurrent PUT/PATCH による
lost update を防ぐ。

## Scope

- `ListScimUsers`/`ListScimGroups` に `sortBy`(対応属性は既存の filter
  allowlist と同じ範囲に閉じる)/`sortOrder`(ascending/descending)を追加する。
- User/Group resource に `meta.version`(ETag)を持たせ、GET 応答の `ETag`
  ヘッダと `If-Match`/`If-None-Match` による PUT/PATCH の条件付き実行に対応する。
  version 不一致は 412 Precondition Failed にする。
- `ServiceProviderConfig.sort.supported`/`etag.supported` を `true` に更新する。

## Out of Scope

- Bulk operation 内での sort/ETag の扱い([[wi-249-scim-bulk-operations]] が
  実装された後に検討する)。

## Plan

- ETag は `idmanagement.User`/`Group` の `UpdatedAt` から導出するハッシュ値と
  するか、専用 version カラムを追加するかを実装前に判断する(既存モデルへの
  影響を最小化する方を優先する)。

## Tasks

- [ ] T001 [SCL] sortBy/sortOrder と ETag/If-Match の契約を `spec/contexts/scim.yaml` に追加する。
- [ ] T002 [Domain] RED: sort と version 比較の test を先に失敗させて実装する。
- [ ] T003 [Usecase/Adapter] RED: sort 付き LIST と If-Match 付き PUT/PATCH の
      HTTP contract test (412 を含む) を先に失敗させて実装する。
- [ ] T004 [Verify] `just yaml-check`、`just test-go`、`just verify-go` を実行する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-go`
- 手動: `sortBy=userName&sortOrder=descending` で LIST 結果の順序が反転することを確認する。
- 手動: 古い ETag を `If-Match` に指定した PUT が 412 になることを確認する。

## Risk Notes

低リスク。ETag の version 導出方法を誤ると偽陰性(実際には変更されているのに
一致と判定)を招きうるため、更新の度に確実に version が変わることをテストで
固定する。
