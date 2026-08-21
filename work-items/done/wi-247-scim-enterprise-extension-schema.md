---
status: completed
authors: [tn]
risk: low
created_at: 2026-07-18
depends_on: []
change_kind: feature
initial_context:
  scl:
    Sourcing:
      - standards.RFC7643.RFC7643-CORE-RESOURCES
      - interfaces.CreateScimUser
      - interfaces.UpdateScimUser
      - interfaces.PatchScimUser
      - interfaces.GetScimSchemas
      - interfaces.GetScimResourceTypes
  source:
    - backend/sourcing/scim/domain/mutation.go
    - backend/sourcing/scim/domain/discovery.go
    - backend/sourcing/scim/handlers_http/handlers.go
  tests:
    - backend/sourcing/scim/domain/discovery_test.go
    - backend/sourcing/scim/handlers_http/resource_contract_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { path: spec/contexts/sourcing/standards.md, requirement: RFC7643-CORE-RESOURCES }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Contract.GetScimSchemas }
  - { path: spec/contexts/sourcing/main.tsp, symbol: IdMagic.Contract.CreateScimUser }
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

- [x] T001 [Spec] enterprise extension の対応属性・schema 契約を `spec/contexts/sourcing/SPECIFICATION.md` に追加する。
- [x] T002 [Domain] RED: enterprise extension の parse/validation test を先に失敗させて実装する。
- [x] T003 [Usecase/Adapter] RED: `/Schemas` と CRUD/PATCH の HTTP contract test を先に失敗させて実装する。
- [x] T004 [Verify] `just check-spec`、`just test-go`、`just verify-go` を実行する。

## Verification

- `just check-spec`
- `just test-go`
- `just verify-go`
- 手動: enterprise extension 付きの User を作成し、GET で `employeeNumber`/`department`/
  `manager` が往復することを確認する。

## Risk Notes

低リスク。既存の汎用 attribute map を再利用できれば idmanagement 側のモデル変更は
不要。`manager` を内部 User への参照として扱う場合は、参照先が同一 tenant に存在する
ことの検証を怠らない(tenant 越境参照の防止)。

## Completion

- **Completed At**: 2026-08-12
- **Summary**:
  SCIM enterprise extension (`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User`) の
  `employeeNumber`・`department`・`manager` サブセットに discovery (`/Schemas`、
  `/ResourceTypes` の `schemaExtensions`) と CRUD/PATCH (`CreateScimUser`・
  `UpdateScimUser`・`PatchScimUser`) で対応した。当初 Plan のとおり
  `idmanagement.User.Attributes` の既存 builtin キー (`employee_number`・`department`・
  `manager_sub`) をそのまま再利用し、idmanagement 側のモデル変更は不要だった。
  `manager` は SCIM id 参照として受け取り、`ScimRepo.FindUserRefByScimID` (tenant-scoped)
  で内部 `User.sub` へ validate-first で解決する。存在しない・別 tenant の SCIM id は
  `invalidValue` にして拒否し、そのユーザーは作成・更新されない(Risk Notes の tenant
  境界懸念に対応)。PATCH の `path` は bare 名 (`employeeNumber` など) と enterprise
  extension URN で修飾した完全パスの両方を受け付け、`manager` の PATCH value は
  `{value: "..."}` オブジェクトと素の文字列(Entra ID が送る形)の両方を受け付ける。
  response の `schemas` と拡張オブジェクトは、いずれかの enterprise extension 属性を
  保持する場合だけ付与する(値が無ければ何も広告しない)。
  `spec/contexts/sourcing/SPECIFICATION.md` に Standards 行
  (`RFC7643-ENTERPRISE-EXTENSION`, partial)、Design 節への属性マッピングと
  tenant-scoped manager 解決の説明、新規 scenario `REQ-SOURCING-007` を追加した。
  `main.tsp` の該当 op docstring も同期した。
- **Out of Scope (as planned)**:
  任意 custom/private extension schema の動的登録、`costCenter`/`division`/
  `organization` 属性、`ListScimUsers`/`ListScimGroups` の filter (`RFC7644-FILTERING`)
  での enterprise extension 属性対応(既存 `TestParseFilterUnknownURNPrefixRejected`
  はそのまま維持し、enterprise extension URN の filter prefix は未対応のまま)。
- **Verification Results**:
  - `just check-spec` - passed (7 normative scenario id(s) in sourcing SPECIFICATION.md)
  - `just check-api-compat` - passed (no breaking changes vs frozen baseline)
  - `just test-go` - passed
  - `just verify-go` - passed (lint-go, race-enabled test-go)
  - 手動: `TestScimCreateUserEnterpriseExtension` の
    `employeeNumber and department round-trip` サブテストで、POST 直後の応答に加えて
    別コードパスの `GET /scim/v2/Users/{id}` でも `employeeNumber`/`department` が
    往復することを確認した(WI記載の手動確認項目を自動テスト化)。`manager` の往復は
    `manager resolves to an existing tenant User and round-trips its scim id`
    サブテストで確認した。
  - Verification 記載の `just yaml-check` は現行 `justfile` に存在しないレシピ名の
    誤記だったため、`just check-spec` に更新した(Tasks/Verification 両セクション)。
