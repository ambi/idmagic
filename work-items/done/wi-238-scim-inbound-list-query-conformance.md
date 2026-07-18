---
status: completed
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

- [x] T001 [SCL] `ListScimUsers` / `ListScimGroups` と RFC 7644 requirement/scenario に、filter grammar の対応範囲、`startIndex` / `count`、`invalidFilter` を記述し派生物を再生成した(`spec/contexts/scim.yaml`、`just scl-render`)。
- [x] T002 [Domain] RED: `backend/scim/domain/filter_test.go` (`TestParseFilterRejectsUnsupported` 等) と `query_test.go` を `domain.ParseFilter` / `domain.NormalizePage` 未実装のコンパイル失敗で先に fail 確認 (interfaces.ListScimUsers/ListScimGroups) → `filter.go` (AST・allowlist・文字列 escape・複合条件・資源上限) と `query.go` (1-origin pagination 正規化) を実装して GREEN。
- [x] T003 [Usecase/Adapter] RED→GREEN: `backend/scim/adapters/http/scim_test.go` の `TestScimListUsersFilterAndPagination` / `TestScimListGroupsFilter` と `backend/scim/usecases/list_test.go` の tenant isolation test で、filter・filter 後 totalResults・1-origin pagination・境界値 (startIndex 超過、count=0)・tenant isolation を固定。`usecases.ListQuery`/`ListResult`、HTTP query binding、memory/PostgreSQL 共通の application-side evaluation を実装。
- [x] T004 [Error] `TestScimListUsersFilterAndPagination` 内の invalidFilter/invalidValue サブテストで、malformed/unsupported filter と不正 pagination (負の count、非整数 startIndex) が `400` の SCIM Error になることを固定。`handlers.go` の `writeListError` でエラー変換、ServiceProviderConfig の `filter.maxResults` を `domain.MaxResults` に統一。
- [x] T005 [Verify] `just yaml-check`、`just scl-render`、`just test-go`、`just verify-go`、`just verify` を実行しグリーンを確認。ARCHITECTURE.md の `go-source-lines` 複雑度予算超過 (usecases.go 859 行) を `usecases.go`/`users.go`/`groups.go`/`list.go` へ関心事分割して解消 (debt 宣言なしで予算内に収めた)。

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

## Completion

- **Completed At**: 2026-07-18
- **Summary**:
  RFC 7644 §3.4.2.2 filter grammar の recursive-descent parser/evaluator (`backend/scim/domain/filter.go`) を
  属性・演算子 allowlist 付きで追加し、User (`userName`, `active`, `name.*`, `emails.value`, `id`) と
  Group (`displayName`, `id`) それぞれの許可範囲に閉じた。`and`/`or`/`not`/丸括弧、`eq`/`ne`/`co`/`sw`/`ew`/`pr`、
  JSON 相当の文字列 escape、入力長・ネスト深さの資源上限をサポートする。`backend/scim/domain/query.go` に
  RFC 7644 §3.4.2.4 の 1-origin `startIndex` 正規化と `count`(省略時既定値・0・clamp・負値エラー)を実装し、
  `ListScimUsers`/`ListScimGroups` の usecase (`usecases.ListQuery`/`ListResult`)・HTTP adapter で
  filter 適用後の `totalResults`/`Resources`/`itemsPerPage` を返すよう usecases.go を再実装した。
  malformed/unsupported filter は `scimType: invalidFilter`、不正 pagination は `invalidValue` の
  HTTP 400 SCIM Error にマップする。ServiceProviderConfig の `filter.maxResults` を実装済み上限
  (`domain.MaxResults = 100`) と一致させた。application-side evaluation により memory/PostgreSQL
  両 adapter で同一の検索意味を保つ (tenant-scoped repository の `FindAll`/`ListByTenant` はそのまま)。
  `usecases.go` の複雑度予算超過を `usecases.go`/`users.go`/`groups.go`/`list.go` への関心事分割で解消した。
- **Affected Guarantees State**:
  `/scim/v2/Users` と `/scim/v2/Groups` は allowlist 範囲の filter、1-origin `startIndex`、
  `count` (既定値・0・上限 clamp・負値拒否) を wire behavior として実際にサポートする。
  許可外属性・演算子・構文エラー・資源上限超過は内部エラーではなく `invalidFilter` の
  SCIM protocol error になる。tenant isolation は list / filter match の両方で維持される。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just scl-render` — passed
  - `just test-go` — passed
  - `just verify-go` — passed
  - `just verify` — passed
  - 手動: `/scim/v2/Users?filter=userName%20eq%20%22alice@example.com%22&startIndex=1&count=1` と
    `/scim/v2/Groups?filter=displayName%20eq%20%22engineering%22` が期待通りの ListResponse を返すことを確認。
  - 手動: `/scim/v2/Users?filter=userName%20eq` (malformed) が HTTP 400 / `scimType: invalidFilter` を返すことを確認。
- **Evidence**:
  - 実行日: 2026-07-18
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Claude Code (Sonnet 5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。`backend/scim/domain/filter_test.go`、`backend/scim/domain/query_test.go`、
    `backend/scim/adapters/http/scim_test.go` (`TestScimListUsersFilterAndPagination`、`TestScimListGroupsFilter`)、
    `backend/scim/usecases/list_test.go` (tenant isolation) が回帰用の contract test として残る。
