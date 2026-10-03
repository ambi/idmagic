---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: [wi-17076-limit-specifications-to-external-contracts-written-once]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/identity-management/user/README.md, requirement: REQ-IDMANAGEMENT-044 }
  - { path: docs/domain/identity-management/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-050 }
---

# 猶予期間を過ぎた User を定期ジョブで完全削除し、途中で失敗した完全削除を再実行で完了できるようにする

## 動機

User の完全削除には、二つの欠陥がある。

| 対象 | 今の挙動 | 起きる問題 |
| --- | --- | --- |
| 猶予期間を過ぎた User | 管理者がユーザー一覧を取得したときにだけ完全削除する | 一覧を取得しないテナントでは、期限を過ぎた User の個人情報が残り続ける。読み取りの操作が副作用を兼ねている |
| 途中で失敗した完全削除 | Tombstone を保存した後に関連する記録の削除や使用量の減算が失敗すると、再実行しても何もせずに成功する | 残った記録と使用量を、手作業で直すほかない |

## 対象範囲

- 猶予期間を過ぎた `PendingDeletion` の User を、定期ジョブで完全削除する。
- ユーザー一覧の取得から完全削除を外す。
- 完全削除を、途中で失敗しても再実行で残りの手順を完了できるようにする。
- REQ-IDMANAGEMENT-044 と REQ-IDMANAGEMENT-050、UserLifecycle の状態遷移表、`user/design.md` の信頼性の節を改める。

## 対象外

- Agent や Group の完全削除。

## 設計

定期ジョブは `Jobs` Context の仕組みで実行する。
完全削除の再実行は、Tombstone を保存した後の手順（関連する記録の削除、使用量の減算、`UserDeleted` の発行と通知）を、完了したかどうかを判定できる形にして、未完了の手順だけを行う。
判定の方法（Tombstone に完了の印を記録するか、各手順を冪等にするか）は着手時に決める。

## 計画

1. 要件の差分と状態遷移表の差分を書く。
2. 定期ジョブを加え、一覧の取得から完全削除を外す。
3. 完全削除を再実行で完了できるようにする。

## タスク

- [ ] T001 [Spec] 要件、状態遷移表、設計の差分を書く。
- [ ] T002 [Acceptance] 一覧を取得しなくても期限を過ぎた User が完全削除されることの RED を確認する。
- [ ] T003 [App] 定期ジョブと、再実行で完了する完全削除を実装する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-work-items`
- `mise run verify`

## リスク

- 定期ジョブの間隔だけ、猶予期間を過ぎた User が残る。
  間隔を要件に書き、利用者が観測できる上限として示す。
