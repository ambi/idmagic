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
  - { path: docs/domain/audit/event-search/README.md, requirement: REQ-AUDIT-002, impact: conforms }
---

# 監査の記録の追記の失敗を検出して補う

## 動機

起動処理の配信点（`backend/cmd/internal/bootstrap/audit_event_record.go` の `NewEmitFunc`）は、ドメインイベントを監査の記録へ写す変換や追記に失敗すると、「照合が必要（reconciliation required）」とログに残すだけである。
照合の仕組みはなく、操作の結果は残り、監査の記録だけが欠けたまま検出も補充もされない。
IdManagement の設計も、Aggregate の保存の後に監査の記録を追記する経路がこの欠落を生みうると書いている（`docs/domain/identity-management/design/audit-events.md`）。

wi-26063 で Audit の内部設計をコードと照合して見つけ、`docs/domain/audit/design/risks.md` に載せた。

## 対象範囲

- 監査の記録の欠落を検出し、補う仕組みを入れる。
- `docs/domain/audit/design/risks.md` の該当の行を消す。

## 対象外

- 監査の記録の保持期間。wi-514 が扱う。
- 監査の記録の改ざんの検知。wi-287 が扱う。

## 設計

製品は専用のイベントの基盤を持たない判断をしている（`docs/domain/system/design/decisions.md`）。
その判断の再検討の条件に「イベントが失われないことを保証する必要が生じ、それが発行の意図を耐久的に残すことでしか達成できないと判明したとき」がある。
着手時に、追記の失敗を耐久的に残す方式（状態の変更と同じトランザクションに発行の意図を書く）と、Aggregate の状態から欠落を照合する方式を比べ、前者を選ぶならシステムの判断の見直しとして扱う。

## 計画

1. 方式を決める。
2. 追記を失敗させるテストで、欠落が残ることを RED として確かめ、実装する。

## タスク

- [ ] T001 [Design] 方式を決める。
- [ ] T002 [App] 実装する。
- [ ] T003 [Verify] 検証する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

発行の意図を耐久的に残す方式は、すべての状態の変更のトランザクションに書き込みを加える。
