---
status: pending
authors: [tn]
risk: medium
created_at: 2026-10-31
priority: p2
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-081 }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-094 }
---

# 採否を判断したうえで Agent の責任者を移管できるようにする

## 動機

所有者が止まると Agent を `Disabled` にする REQ-IDMANAGEMENT-081 は安全側だが、退職のたびに業務の Agent が予告なく止まる。
[Entra](https://learn.microsoft.com/en-us/entra/id-governance/agent-id-governance-overview) は責任者の退職時に上長へ移管し、関係者へ通知する。
ただし自動移管は、確認していない人へ責任と権限を渡す懸念があるので、採る形を先に決める。

## 対象範囲

- 判断を最初に行う。自動移管、移管先の確認、事前通知だけの案を比べる。
- 所有者の停止の予定を検知できる経路（削除の予約、ライフサイクル）での通知。
- 管理者が移管先を確認し、監査へ残す操作。

## 対象外

- 複数の責任者と owner の区別。必要になってから扱う。
- 確認なしに Agent を動かし続ける挙動。「誰も責任を持たない Agent を残さない」判断を弱める案は、T001 で明確な理由がなければ採らない。

## 設計

第一候補は次のとおりである。

| 案 | 判断 |
| --- | --- |
| 事前通知と管理者の確認による移管（第一候補） | 責任の空白を作らず、業務の停止を減らす |
| 上長への自動移管 | 継続性は高いが、上長の情報の信頼性に依存し、本人の確認がない |
| 現状の無効化だけ | 単純だが、予告のない停止が残る |

移管後も、前の所有者の停止で無効になった Agent を自動で再開しない。
再開は既存の再有効化の操作で行う。

## 計画

1. 利用場面と案を比べて決める。
2. 仕様を更新し、通知と移管を実装する。

## タスク

- [ ] T001 [Decision] 移管の案と通知の時期を確定する。
- [ ] T002 [Spec] 移管の規範と TypeSpec を追加する。
- [ ] T003 [Acceptance] 予約の通知、移管、移管なしの無効化の RED を観測する。
- [ ] T004 [App] 実装する。
- [ ] T005 [Verify] 並行する所有者の停止と移管を検証する。

## 検証

- `mise run test-go-package -- ./backend/idmanagement/...`。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

移管の操作は Agent の権限を新しい所有者の監督下へ移す。
移管先を同じテナントの `Active` な User に限り、監査で追えるようにする。
