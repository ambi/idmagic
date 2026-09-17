---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p1
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: bugfix
affected_spec:
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

[API ガイドライン](../docs/design/application/api-guidelines.md)の「ステータスコードの宣言」は、ハンドラーが返すステータスコードをすべて TypeSpec に宣言すると定める。
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

- [ ] T001 [Design] ヘルパーを経由する API 操作と到達しうる分岐を列挙する。
- [ ] T002 [Spec] 409 の宣言を追加する。
- [ ] T003 [Acceptance] 名前の重複で 409 を返すことを HTTP の境界で固定する。
- [ ] T004 [Docs] API ガイドラインの適用状況を改める。
- [ ] T005 [Verify] 変更を検証する。

## Verification

- `mise run check-status-drift`
- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

宣言の追加は後方互換な変更だが、生成したクライアントの union が変わる。フロントエンドの型検査で影響を確かめる。
