---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-003, impact: conforms }
  - { path: docs/modules/identity-governance/workflow-run/README.md, requirement: REQ-IDGOVERNANCE-005, impact: conforms }
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
