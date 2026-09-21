---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p1
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 既存の 409 応答を生成クライアントが型付きの応答として扱えるようになり、API 利用者に見える契約が広がる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-595-declare-uniqueness-conflict-responses.md }
initial_context:
  specification:
    - docs/design/application/api-guidelines.md
    - docs/domain/workloadidentity/scenarios.feature.md#REQ-WORKLOADIDENTITY-008
  typespec:
    - IdMagic.IdManagement.Operations.CreateAdminUser
    - IdMagic.IdManagement.Operations.UpdateAdminUser
    - IdMagic.IdManagement.Operations.CreateGroup
    - IdMagic.IdManagement.Operations.UpdateGroup
    - IdMagic.IdManagement.Operations.RegisterAgent
    - IdMagic.IdManagement.Operations.UpdateAgent
    - IdMagic.WorkloadIdentity.Operations.RegisterWorkloadTrustBundle
    - IdMagic.WorkloadIdentity.Operations.UpdateWorkloadTrustBundle
    - IdMagic.IdGovernance.Operations.CreateLifecycleWorkflow
  source:
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
    - backend/idmanagement/group/handlers_http/admin_group_handler.go
    - backend/idmanagement/agent/handlers_http/admin_agent_handler.go
    - backend/workloadidentity/handlers_http/routes.go
    - backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go
  tests:
    - backend/workloadidentity/usecases/admin_trust_bundles_test.go
    - backend/workloadidentity/handlers_http/routes_test.go
  stop_before_reading:
    - frontend
primary_use_cases:
  - id: register-trust-bundle-conflict
    requirement: REQ-WORKLOADIDENTITY-008
    observable_result: 同一テナントで重複した信頼設定の登録は Problem Details の 409 を返し、クライアント契約は名前と発行者の競合型を列挙する。
    unit_test: { path: backend/workloadidentity/usecases/admin_trust_bundles_test.go, name: TestRegisterWorkloadTrustBundle, task: test-go-race }
    e2e_test: { path: backend/workloadidentity/handlers_http/routes_test.go, name: TestRegisterTrustBundleReportsUniquenessConflict, task: test-go-race }
    unit_fault_model: ユースケースが同一テナント内の重複名または重複発行者を受理する。
    e2e_fault_model: HTTP ハンドラーが競合エラーを 409 の Problem Details へ変換しない。
affected_spec:
  - { path: docs/domain/workloadidentity/scenarios.feature.md, requirement: REQ-WORKLOADIDENTITY-008 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.CreateAdminUser }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateAdminUser }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.CreateGroup }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateGroup }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.RegisterAgent }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateAgent }
  - { path: spec/contexts/workloadidentity/main.tsp, symbol: IdMagic.WorkloadIdentity.Operations.RegisterWorkloadTrustBundle }
  - { path: spec/contexts/workloadidentity/main.tsp, symbol: IdMagic.WorkloadIdentity.Operations.UpdateWorkloadTrustBundle }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.CreateLifecycleWorkflow }
---

# 名前の重複で返す 409 を TypeSpec に宣言する

## Motivation

[API ガイドライン](../../docs/design/application/api-guidelines.md)の「ステータスコードの宣言」は、ハンドラーが返すステータスコードをすべて TypeSpec に宣言すると定める。
利用者名、グループ名、エージェント名、トラストバンドルの名前と発行者、ライフサイクルワークフローの名前の重複に対し、ハンドラーは 409 を返す。
しかし、対応する作成と更新の API 操作は 409 を宣言していない。
生成したクライアントは 409 を未知のステータスコードとして扱い、名前の重複を利用者に伝えられない。

`mise run check-status-drift` がこの宣言漏れを検出しないのは、409 を返す分岐がエラーをマッピングするヘルパー（`writeAdminUserError` など）の中にあり、検査がヘルパーを追跡しないためである。

## Scope

- `CreateAdminUser`、`UpdateAdminUser`、`CreateGroup`、`UpdateGroup`、`RegisterAgent`、`UpdateAgent`、`RegisterWorkloadTrustBundle`、`UpdateWorkloadTrustBundle`、`CreateLifecycleWorkflow` に、実際に返すエラーコードの 409 を宣言する。
- 同じヘルパーを経由する他の API 操作についても、返すステータスコードと宣言を突き合わせる。
- API ガイドラインの適用状況を改める。

## Out of Scope

- `mise run check-status-drift` がヘルパーを追跡できるようにする改修。
- 409 を返す条件そのものの変更。

## Design

エラーコードごとに `model <Name>Error is ProblemDetails;` を宣言し、API 操作の 409 の union に加える。
ヘルパーの `switch` を読み、ヘルパーを呼ぶ API 操作ごとに到達しうる分岐を判定する。
到達しない分岐まで宣言すると、クライアントに到達しない分岐を実装させるため、ユースケースが返しうるエラーに限って宣言する。

## Plan

1. `mise run check-status-drift -- --list-unresolved` で部分解析の API 操作を列挙し、上記のヘルパーを経由するものを特定する。
2. API 操作ごとに、ユースケースが返しうるエラーと宣言を突き合わせる。

## Tasks

- [x] T001 [Design] ヘルパーを経由する API 操作と到達しうる分岐を列挙する。
- [x] T002 [Spec] 409 の宣言を追加する。
- [x] T003 [Acceptance] 名前の重複で 409 を返すことを HTTP の境界で固定する。
- [x] T004 [Docs] API ガイドラインの適用状況を改める。
- [x] T005 [Verify] 変更を検証する。

## Verification

- `mise run check-status-drift`
- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

宣言の追加は後方互換な変更だが、生成したクライアントの union が変わる。フロントエンドの型検査で影響を確かめる。

## Completion

- **Completed At**: 2026-09-21
- **Summary**:
  `mise run spec-diff` は `REQ-WORKLOADIDENTITY-008` の競合エラー型への訂正と、利用者、グループ、エージェント、トラストバンドル、ライフサイクルワークフローに対する 6 個の TypeSpec Problem Details 宣言の追加を報告した。9 操作が実際に返す 409 を、対応する名前・発行者の競合型として契約へ加えた。
- **Primary Use Case Evidence**:
  - id: register-trust-bundle-conflict
    unit_red: "N/A: ユースケースの一意性判定はこの契約修正の前から実装済みであり、既存テストは GREEN だった。"
    e2e_red: "N/A: HTTP ハンドラーの 409 Problem Details 変換も契約修正の前から実装済みであり、追加した境界テストは既存の振る舞いを固定した。"
    unit_fault_injection: "`ErrTrustBundleNameConflict` を返さない変異を与えると、`TestRegisterWorkloadTrustBundle` は `err = <nil>, want ErrTrustBundleNameConflict` で失敗した。"
    e2e_fault_injection: "重複名の変換を 409 から 400 へ替えると、`TestRegisterTrustBundleReportsUniquenessConflict` は `duplicate registration status=400, want 409` で失敗した。"
- **Verification Results**:
  - `mise run check-spec`、`mise run check-contract-drift`、`mise run check-api-compat`、`mise run check-status-drift` - 成功
  - `mise run test-go-changed`、`mise run lint-go`、対象 Unit / HTTP テスト - 成功
  - `mise run verify` - 成功
- **Left Undone**:
  - `mise run check-status-drift` がエラーマッピングヘルパーを完全解析する改修は対象外のまま残る。
