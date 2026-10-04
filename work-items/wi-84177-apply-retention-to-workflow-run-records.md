---
status: pending
authors: [tn]
risk: low
reversibility: irreversible
created_at: 2026-10-04
priority: p3
depends_on: []
change_kind: maintenance
spec_impact:
  kind: none
  reason: "着手時点では保持期間は内部設計の方針であり、規範の要件はない。要件にするかを設計で判断し、要件にするなら spec_impact を affected_spec へ改める。"
---

# ライフサイクルワークフローの実行の記録に保持期間を適用する

## 動機

旧 IdGovernance の内部設計は、`WorkflowRun`、`WorkflowStep`、`LifecycleNotificationDelivery` が Job の記録と同じ 30 日の保持期間に従うと書いていた。
しかし、実行の記録も Job の記録も、消す処理はない（`idmagic-batch retention-sweep` の対象に入っていない）。
記録は消えずに増え続ける。

wi-26063 で IdGovernance の内部設計をコードと照合して見つけ、`docs/domain/identity-governance/design/risks.md` に載せた。

## 対象範囲

- 実行の記録と Job の記録の保持期間を決め、保持期間の掃引で消す。
- 保持期間を要件にするかを判断する。
- `docs/domain/identity-governance/design/risks.md` の該当の行を消し、`docs/design/data/lifecycle.md` に保持期間を載せる。

## 対象外

- 監査の記録の保持期間。wi-514 が扱う。

## 設計

保持期間の掃引（`backend/cmd/internal/bootstrap/retention.go`）に、終端に達した WorkflowRun と、それに関連する手順と通知の配送、終端に達した Job を加える案を第一の候補とする。
終端でない記録は消さない。
実行の詳細の参照（管理 API）が、消した記録を参照したときの応答を着手時に確かめる。

## 計画

1. 保持期間と対象を決める。
2. 掃引に加え、終端でない記録を消さないことをテストで確かめる。

## タスク

- [ ] T001 [Design] 保持期間と対象を決める。
- [ ] T002 [App] 掃引に加える。
- [ ] T003 [Verify] 検証する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

消した記録は戻せない。
保持期間を決める前に、運用者が実行の履歴をどれだけ遡って調べるかを確かめる。
