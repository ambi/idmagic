---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-18
depends_on: [wi-246-scim-multivalued-core-attributes-and-nested-group-members]
change_kind: feature
initial_context:
  scl:
    Scim:
      - standards.RFC7644.RFC7644-PATCH
      - interfaces.ListScimUsers
      - interfaces.ListScimGroups
      - interfaces.PatchScimUser
      - interfaces.PatchScimGroup
  source:
    - backend/scim/domain/filter.go
    - backend/scim/domain/mutation.go
  tests:
    - backend/scim/domain/filter_test.go
    - backend/scim/domain/mutation_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Scim, kind: standard_requirement, standard: RFC7644, requirement: RFC7644-PATCH }
  - { context: Scim, kind: interface, element: ListScimUsers }
  - { context: Scim, kind: interface, element: PatchScimUser }
---

# SCIM 複合 value フィルタ (bracket 構文) を LIST filter と PATCH path の両方に対応する

## Motivation

RFC 7644 §3.4.2.2 の `attrPath "[" valFilter "]"` 記法(例
`emails[type eq "work"].value`、`members[value eq "..."]`)は、multi-valued 複合
属性の特定要素だけを対象にする。この grammar は LIST の `filter` query parameter
([[wi-238-scim-inbound-list-query-conformance]]、[[wi-244-scim-filter-grammar-extended-conformance]]
がいずれも明示的に対象外とした)と PATCH の `path`([[wi-239-scim-inbound-resource-contract-conformance]]
が対象外とした)の両方で使われる同一の grammar 拡張であり、1つの実装で両方の
呼び出し元に供給できる。

[[wi-246-scim-multivalued-core-attributes-and-nested-group-members]] が
multi-valued 属性(複数 emails 等)を実装するまでは、この bracket 構文で絞り込む
対象自体が存在しないため、本 WI はそれに依存する。

## Scope

- `backend/scim/domain` に bracket 構文の `valFilter` パーサーを追加し、
  `emails[type eq "work"].value` のような path を、既存の attribute allowlist と
  組み合わせて解決する。
- `ListScimUsers`/`ListScimGroups` の `filter` と、`PatchScimUser`/`PatchScimGroup`
  の PATCH `path` の両方でこの拡張 grammar を共有する。
- 資源上限(入力長・ネスト深さ)を既存の `domain.MaxFilterLength`/`MaxFilterDepth`
  と同じ考え方で適用する。

## Out of Scope

- bracket 構文以外の filter/PATCH 拡張([[wi-244-scim-filter-grammar-extended-conformance]]
  が扱う `gt`/`ge`/`lt`/`le`、schema URN プレフィックス等)。

## Plan

- 既存の `backend/scim/domain/filter.go` の AST・allowlist 機構を拡張する形にし、
  新しい parser を作らない。LIST filter と PATCH path の両方から同じ評価ロジックを
  呼べるよう、attribute allowlist の型を multi-valued 複合属性のサブ条件を表現
  できるように広げる。

## Tasks

- [ ] T001 [SCL] bracket 構文の対応範囲を `spec/contexts/scim.yaml` に明記する。
- [ ] T002 [Domain] RED: bracket 構文 parser/evaluator の test (複合条件、資源上限、
      allowlist 外属性の拒否)を先に失敗させて実装する(未信頼入力を parse する
      複雑な文法のため、fuzz/property test 採用の要否を検討し判断根拠を Risk Notes に
      記す — ADR-121)。
- [ ] T003 [Usecase/Adapter] RED: LIST filter と PATCH path 双方の HTTP contract test を
      先に失敗させて実装する。
- [ ] T004 [Verify] `just yaml-check`、`just test-go`、`just verify-go` を実行する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-go`
- 手動: `/scim/v2/Users?filter=emails[type eq "work"].value eq "..."` が期待通りの
  部分集合を返すことを確認する。
- 手動: PATCH で `members[value eq "..."]` を path に指定した remove operation が
  該当メンバーだけを削除することを確認する。

## Risk Notes

`backend/scim/domain/filter.go` は既に外部 untrusted input を parse する
security-sensitive なコードであり、bracket 構文はネスト・組み合わせ爆発を招きやすい
文法拡張である。既存の資源上限機構を必ず適用し、fuzz test 採用を検討する
(ADR-121: 文法が複雑・高リスクな parser は要検討・要記録)。
