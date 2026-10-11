---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 初回公開前で実利用契約はない。既存データの移行は不要で、以後の手順に管理操作と同じイベントと通知を伴わせる。
  references: []
initial_context:
  specification: [docs/modules/identity-governance/workflow-run/README.md#REQ-IDGOVERNANCE-003, docs/modules/identity-governance/workflow-run/README.md#REQ-IDGOVERNANCE-005, docs/modules/identity-management/group/README.md#REQ-IDMANAGEMENT-015, docs/modules/identity-management/user/README.md#REQ-IDMANAGEMENT-047]
  source: [backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go, backend/idmanagement/group/usecases/admin_groups.go, backend/idmanagement/user/usecases/admin_users.go, backend/cmd/idmagic-worker/worker.go]
  tests: [backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go, backend/idgovernance/usecases/scenario_examples_test.go]
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/modules/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-003, impact: conforms }
  - { path: docs/modules/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-005, impact: conforms }
  - { path: docs/modules/identity-management/group/README.md, requirement: REQ-IDMANAGEMENT-015, impact: conforms }
  - { path: docs/modules/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-047, impact: conforms }
primary_use_cases:
  - id: worker-identity-actions
    requirement: REQ-IDGOVERNANCE-003
    observable_result: Worker のグループと必須操作の変更がドメインイベントとグループ通知を伴う。
    boundary: adapter
    test: { path: backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go, name: TestWorkerLifecycleWorkflowIdentityActions, task: test-go-package }
    fault_model: Worker が操作ポートを渡し忘れる、または手順が保存先を直接呼ぶ。
  - id: repeated-identity-actions
    requirement: REQ-IDGOVERNANCE-005
    observable_result: 同じ操作の再実行は no_op となりイベントと通知を増やさない。
    boundary: adapter
    test: { path: backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go, name: TestWorkerLifecycleWorkflowIdentityActions, task: test-go-package }
    fault_model: 冪等な操作でも changed と記録しイベントを重複発行する。
  - id: tenant-isolation
    requirement: REQ-IDMANAGEMENT-015
    observable_result: 別テナントの User と Group を変更せずイベントも発行しない。
    boundary: adapter
    test: { path: backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go, name: TestWorkerLifecycleWorkflowIdentityActions, task: test-go-package }
    fault_model: Worker から渡されたテナント境界を操作が取り違える。
---

# ライフサイクルワークフローのグループと必須操作の手順を IdManagement の操作へ通す

## 動機

ライフサイクルワークフローの手順のうち、`enable_user` と `disable_user` は `UserLifecycle` のポートを、`assign_application` と `unassign_application` は `ApplicationAssignments` のポートを通り、管理 API で同じ変更をしたときと同じイベントと通知が伴う。
一方、`add_group_member` と `remove_group_member` は IdManagement の Group の Repository を、`set_required_action` と `clear_required_action` は User の Repository を直接呼ぶ。
これらの手順には、管理 API で同じ変更をしたときのドメインイベント、監査の記録、下流へのプロビジョニングの通知が伴わない。

wi-26063 で IdGovernance の内部設計をコードと照合して見つけ、`docs/modules/identity-governance/design/risks.md` に載せた。

## 対象範囲

- IdManagement に、グループのメンバーシップと必須操作をあるべき状態として冪等に変えるポートを設け、ワークフローの手順をそこへ通す。
- 変更しなかった場合は `no_op` を返す挙動（REQ-IDGOVERNANCE-005）を保つ。
- `docs/modules/identity-governance/design/risks.md` と `workflow-run/design.md` の該当の記述を直す。

## 対象外

- 実行の記録の保持期間。wi-84177 が扱う。

## 設計

`UserLifecycle` と同じく、IdGovernance が定め IdManagement が実装するポートにする。
ポートの実装は、ワークフローの手順による変更が新しい WorkflowRun を計画しないよう、`UserMutationCommitter` を通さない。
これが、アクションからトリガーへの循環を断つ現在の仕組みだからである。

### 着手時の判断

入力は WorkflowRun のテナント、対象 User、各手順の Group または RequiredAction、実行時刻である。
`GroupMembership.SetGroupMembership(ctx, tenantID, groupID, userID, present, now) (changed bool, err error)` と `UserRequiredActions.SetUserRequiredAction(ctx, tenantID, userID, action, required, now) (changed bool, err error)` を利用側の ports に置く。
実装は IdManagement の既存の管理操作と永続化・イベント・通知の境界を共有する。
組み立ては bootstrap と Worker に置く。
実行前の評価には既存の読み取り Repository を使い、変更はポートへ渡す。
時刻は Worker から渡し、Actor は `lifecycle-workflow` に固定する。

保存先を直接呼ぶ案は、管理操作のイベントと通知の規則が重複するため採らない。
IdGovernance から IdManagement の usecases を直接参照する案は、モジュール間の依存規則に反するため採らない。
利用側のポートを組み立て地点で実装へ接続する案は、記録の所有者を変えず、手順の追加の波及を操作境界に収めるため採る。
管理操作の仕様と HTTP 契約は変えず、既存の REQ-IDMANAGEMENT-015、047 と REQ-IDGOVERNANCE-003、005 への適合を宣言する。
グループ通知の失敗後に業務状態が残る現在の保証は、この前提作業では維持する。

## 計画

1. 手順のテストで、イベントが発行されないことを RED として確かめる。
2. ポートを設けて手順を移す。

## タスク

- [ ] T001 [App] ポートを設け、手順を移す。
- [ ] T002 [Verify] 検証する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

メンバーシップの変更が下流へのプロビジョニングを起こすようになり、ワークフローの実行の数だけ下流の呼び出しが増える。
