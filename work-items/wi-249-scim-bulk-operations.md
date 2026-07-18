---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-18
depends_on: []
change_kind: feature
initial_context:
  scl:
    Scim:
      - standards.RFC7644.RFC7644-RESOURCE-OPERATIONS
      - standards.RFC7644.RFC7644-ERROR-RESPONSE
      - interfaces.CreateScimUser
      - interfaces.UpdateScimUser
      - interfaces.PatchScimUser
      - interfaces.DeleteScimUser
      - interfaces.CreateScimGroup
      - interfaces.GetScimServiceProviderConfig
  source:
    - backend/scim/adapters/http/handlers.go
    - backend/scim/usecases
  tests:
    - backend/scim/adapters/http/resource_contract_test.go
  stop_before_reading:
    - frontend
affected_spec:
  - { context: Scim, kind: standard_requirement, standard: RFC7644, requirement: RFC7644-RESOURCE-OPERATIONS }
  - { context: Scim, kind: interface, element: GetScimServiceProviderConfig }
  - { context: Scim, kind: interface, element: CreateScimUser }
---

# SCIM Bulk operations (/Bulk) に対応する

## Motivation

RFC 7644 §3.7 の `/Bulk` エンドポイントは、大量の User/Group の作成・更新・削除を
1 リクエストにまとめて送れる。現状 `ServiceProviderConfig.bulk.supported` は
`false` を正しく広告しているが、大規模 IdP 同期(初期移行、大量組織変更)では
個別リクエストの往復コストが問題になりうる。wi-238/wi-239 はいずれも明示的に
Bulk を対象外とした。

## Scope

- `POST /scim/v2/Bulk` エンドポイントを追加し、`Operations` 配列内の各要素
  (method/path/bulkId/data)を、既存の CreateScimUser/UpdateScimUser/PatchScimUser/
  DeleteScimUser/CreateScimGroup 等の usecase に委譲して処理する。
- `bulkId` による操作間参照(例: 同一 Bulk request 内で作成した User の id を
  後続の Group member 追加で参照する)を解決する。
- `failOnErrors` としきい値、`maxOperations`/`maxPayloadSize` の
  `ServiceProviderConfig.bulk` 広告を実装値と一致させる。

## Out of Scope

- Bulk 内での filter/query 操作(RFC は method GET を Bulk に含めない)。
- 非同期・バックグラウンド実行(RFC は同期応答を前提とする。将来的に
  パフォーマンス上必要なら別途 ADR で検討)。

## Plan

- 各 Bulk operation は既存の単体 usecase メソッド(CreateUser/UpdateUser/...)を
  そのまま呼び出し、新しい mutation ロジックを重複実装しない。
- 部分失敗時の atomicity は [[wi-239-scim-inbound-resource-contract-conformance]] の
  ADR-122(validate-first + 補償クリーンアップ)と同じ方針を踏襲し、Bulk 全体を
  1つの DB transaction にはしない。`failOnErrors` のしきい値超過で残りの
  operation を打ち切る。

## Tasks

- [ ] T001 [SCL] `/Bulk` の契約(request/response 形状、`bulkId` 解決、
      `failOnErrors`、`ServiceProviderConfig.bulk` 広告値)を `spec/contexts/scim.yaml` に追加する。
- [ ] T002 [Domain] RED: bulkId 参照解決・operation validation の test を先に失敗させて実装する。
- [ ] T003 [Usecase/Adapter] RED: `/Bulk` の HTTP contract test(部分失敗、`failOnErrors`、
      上限超過)を先に失敗させて実装する。
- [ ] T004 [Verify] `just yaml-check`、`just test-go`、`just verify-go` を実行する。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-go`
- 手動: 複数 User の create と、その `bulkId` を参照する Group member 追加を含む
  Bulk request を送り、想定通り解決されることを確認する。
- 手動: `failOnErrors` を超えた場合に残りの operation が実行されないことを確認する。

## Risk Notes

Bulk は1リクエストで大量の副作用を起こせるため、誤操作時の影響範囲が単体
API より大きい。`maxOperations`/`maxPayloadSize` を厳格に強制し、tenant
isolation を Bulk 内の全 operation で漏れなく検証する。atomicity は
[[wi-239-scim-inbound-resource-contract-conformance]] の ADR-122 と同じ限界
(真の transaction ではない)を継承する。
