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
      - standards.RFC7643.RFC7643-CORE-RESOURCES
      - interfaces.CreateScimUser
      - interfaces.UpdateScimUser
      - interfaces.PatchScimUser
      - interfaces.GetScimSchemas
      - interfaces.GetScimResourceTypes
  source:
    - backend/scim/domain/mutation.go
    - backend/scim/domain/discovery.go
    - backend/scim/adapters/http/handlers.go
  tests:
    - backend/scim/domain/discovery_test.go
    - backend/scim/adapters/http/resource_contract_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Scim, kind: standard_requirement, standard: RFC7643, requirement: RFC7643-CORE-RESOURCES }
  - { context: Scim, kind: interface, element: GetScimSchemas }
  - { context: Scim, kind: interface, element: CreateScimUser }
---

# SCIM enterprise extension schema (urn:...:extension:enterprise:2.0:User) に対応する

## Motivation

Entra ID、Workday 連携などの実運用 IdP は
`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User` の `employeeNumber`、
`department`、`manager` などを付与して送ることが多い。現状 idmagic はこの拡張
schema を discovery(`/Schemas`)にも request body 処理にも一切持たず、送られてきても
黙殺する。

## Scope

- 対応する enterprise extension 属性を小さく明示する(まず `employeeNumber`、
  `department`、`manager` 程度に絞る。値の永続化先は `idmanagement.User.Attributes`
  (既存の汎用 attribute map)を使うか検討する)。
- `/Schemas` に enterprise extension schema を追加し、`/ResourceTypes` の
  `schemaExtensions` を更新する。
- CreateScimUser / UpdateScimUser / PatchScimUser の body 中の
  `urn:ietf:params:scim:schemas:extension:enterprise:2.0:User` キー配下を読み書きする。

## Out of Scope

- 任意の custom/private extension schema の動的登録(汎用スキーマ拡張機構)。
  これは本 WI よりずっと大きい設計課題であり、必要になった時点で別途 ADR/WI とする。
- `costCenter`、`division`、`organization` 等の追加属性(まず小さい subset で始める)。

## Plan

- `idmanagement.User.Attributes`(`map[string]idmdomain.AttributeValue`)が既に
  汎用属性の入れ物として存在するため、まずこれを再利用できるか確認してから
  専用フィールドの追加を検討する。

## Tasks

- [ ] T001 [SCL] enterprise extension の対応属性・schema 契約を `spec/contexts/scim.yaml` に追加する。
- [ ] T002 [Domain] RED: enterprise extension の parse/validation test を先に失敗させて実装する。
- [ ] T003 [Usecase/Adapter] RED: `/Schemas` と CRUD/PATCH の HTTP contract test を先に失敗させて実装する。
- [ ] T004 [Verify] `just yaml-check`、`just test-go`、`just verify-go` を実行する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-go`
- 手動: enterprise extension 付きの User を作成し、GET で `employeeNumber`/`department`/
  `manager` が往復することを確認する。

## Risk Notes

低リスク。既存の汎用 attribute map を再利用できれば idmanagement 側のモデル変更は
不要。`manager` を内部 User への参照として扱う場合は、参照先が同一 tenant に存在する
ことの検証を怠らない(tenant 越境参照の防止)。
