---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-18
depends_on: []
change_kind: bugfix
initial_context:
  scl:
    Scim:
      - standards.RFC7644.RFC7644-ERROR-RESPONSE
      - interfaces.ListScimUsers
      - interfaces.ListScimGroups
      - scenarios.SCIM clientはUsersとGroups collectionを検索できる
  source:
    - backend/scim/adapters/http/handlers.go
    - backend/scim/usecases/usecases.go
    - backend/scim/ports/repository.go
    - backend/idmanagement/ports/user_repository.go
    - backend/idmanagement/ports/group_repository.go
  tests:
    - backend/scim/adapters/http/scim_test.go
    - backend/scim/usecases
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Scim, kind: standard_requirement, standard: RFC7644, requirement: RFC7644-ERROR-RESPONSE }
  - { context: Scim, kind: interface, element: ListScimUsers }
  - { context: Scim, kind: interface, element: ListScimGroups }
  - { context: Scim, kind: scenario, element: SCIM clientはUsersとGroups collectionを検索できる }
---

# inbound SCIM collection 検索を filter と offset pagination の契約どおりに実装する

## Motivation

現在の ServiceProviderConfig は `filter.supported: true` と `maxResults: 100` を広告し、
SCL も Users / Groups collection に `filter`、`startIndex`、`count` と不正 filter 時の
SCIM protocol error を規定している。しかし実装は Users の `userName eq "..."` を脆弱な
文字列スライスで特別扱いするだけで、Groups の filter は受け渡されず、pagination parameter
も読まれない。主要な SCIM client が利用する lookup / page traversal が成立せず、広告した
capability と実際の wire behavior が乖離している。

## Scope

- `spec/contexts/scim.yaml` の RFC 7644 採用要件、`ListScimUsers`、`ListScimGroups`、collection
  検索 scenario を、サポート属性・演算子・pagination の境界とエラー形式まで明文化する。
- `backend/scim` に RFC 7644 §3.4.2.2 の filter grammar を入力長・深さなどの資源上限付きで
  parse / validate する domain query を追加し、公開する User / Group 属性だけを allowlist で
  評価する。未対応属性・演算子・構文は `400` / `scimType: invalidFilter` とする。
- Users と Groups の両方で filter を適用し、RFC 7644 §3.4.2.4 の 1-origin `startIndex` と
  `count` を検証して、filter 後の `totalResults`、ページの `Resources`、`itemsPerPage` を
  正しく返す。`count` は ServiceProviderConfig が広告する上限を超えないようにする。
- HTTP adapter、usecase、memory / PostgreSQL で同じ検索意味を保つ。最初は安全な
  application-side evaluation を許容し、データ量要件上必要なら allowlist に対応した
  repository query へ移す。
- parser/evaluator の unit test と HTTP contract test を追加し、複合 filter、文字列 escape、
  Group lookup、空結果、page boundary、malformed / unsupported filter、tenant isolation を固定する。

## Out of Scope

- SCIM Bulk、sort、ETag、外部カスタム schema の任意属性検索、属性 projection
  (`attributes` / `excludedAttributes`)。
- outbound SCIM provisioning（[[wi-45-outbound-scim-provisioning]]）。
- `/Schemas` の完全な schema discovery と、PUT / PATCH の未対応 attribute / operation 全般。
  これらは別途 conformance work item として扱う。

## Plan

- SCL で対応範囲を先に固定する。任意 SQL を外部から受け取らず、SCIM grammar を AST にし、
  resource ごとの属性・演算子 allowlist に閉じる。
- filter は pagination の前に評価し、安定した順序を定めてから 1-origin offset を適用する。
  `count=0`、未指定値、負値、広告上限超過の wire semantics は RFC と client interoperability を
  基準に仕様化する。
- invalid input は内部エラーに落とさず、SCIM Error response で返す。ServiceProviderConfig の
  filter advertisement は実装済みの範囲だけを示す。
- PostgreSQL の最適化は、filter AST から SQL を文字列連結しない。必要になった場合は属性ごとの
  固定 query を選択するか、検証済み AST をパラメータへ変換する。

## Tasks

- [ ] T001 [SCL] `ListScimUsers` / `ListScimGroups` と RFC 7644 requirement/scenario に、filter grammar の対応範囲、`startIndex` / `count`、`invalidFilter` を記述し派生物を再生成する。
- [ ] T002 [Domain] RED: SCL の collection interface を参照する filter parser/evaluator test を先に失敗させ、AST、allowlist、文字列 escape、複合条件、資源上限を実装して GREEN にする。
- [ ] T003 [Usecase/Adapter] RED: Users / Groups の filter、filter 後 totalResults、1-origin pagination、境界値、tenant isolation の HTTP contract test を先に失敗させ、query binding・usecase・memory/PostgreSQL の意味を実装して GREEN にする。
- [ ] T004 [Error] RED: malformed / unsupported filter と不正 pagination が `400` の SCIM Error (`invalidFilter` を含む) になる test を先に失敗させ、エラー変換と ServiceProviderConfig の広告を整合させて GREEN にする。
- [ ] T005 [Verify] SCL、work-item、Go の検証を実行し、PostgreSQL adapter を含む collection query の回帰を確認する。

## Verification

- `just yaml-check`
- `just scl-render`
- `just test-go`
- `just verify-go`
- 手動: 有効な Bearer token で `/scim/v2/Users?filter=userName%20eq%20%22alice%22&startIndex=1&count=1` と
  `/scim/v2/Groups?filter=displayName%20eq%20%22engineering%22` を呼び、ListResponse と一致することを確認する。
- 手動: malformed filter が HTTP 500 や黙殺でなく、`scimType: invalidFilter` を持つ HTTP 400 になることを確認する。

## Risk Notes

SCIM filter は外部入力であり、曖昧な parser や SQL 文字列生成は injection、過剰な計算量、
tenant 越境検索を招く。AST 化、属性 allowlist、入力サイズ・ネスト深さ・結果数の上限、全 adapter
共通の contract test でリスクを抑える。ページングの off-by-one は同期クライアントで欠落または
重複を発生させるため、空集合と最終ページを必ず検証する。
