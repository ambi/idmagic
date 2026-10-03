---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: [wi-96676-transcribe-implicit-specifications-of-identity-management]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/identity-management/principals/user/lifecycle.md, requirement: REQ-IDMANAGEMENT-046 }
  - { path: docs/domain/identity-management/bulk-transfer/user-csv/README.md, requirement: REQ-IDMANAGEMENT-006 }
---

# IdManagement の実装を状態遷移表に合わせる

## 動機

IdManagement の既存コードを書き起こしたとき、状態遷移表と実装が食い違う点を二つ見つけた。
状態遷移表は規範なので、現在の挙動を規則として書き起こさず、この work item で扱う。

| 状態機械 | 状態遷移表 | 実装 |
| --- | --- | --- |
| `UserLifecycle` | `PendingDeletion` から出る遷移は、`UserRestored` による `Active` と、`UserDeleted` による `Deleted` だけである | 無効化は `PendingDeletion` の User を `Disabled` にし、再有効化は `PendingDeletion` の User を猶予期間も確かめずに `Active` にする。どちらも `UserRestored` を発行しない |
| `DataExportLifecycle` | `succeeded` から `expired` への遷移のガードは、`completed_at` からの経過である | 期限を `created_at` に 30 日を加えた時刻で判定する |

## 対象範囲

- `PendingDeletion` の User の無効化と再有効化を拒否するか、遷移表に遷移を加えるかを決め、実装か表をそろえる。
- エクスポートの期限の基準を `completed_at` と `created_at` のどちらにするかを決め、実装か表をそろえる。

## 対象外

- 状態の追加と削除。

## 計画

1. 二つの食い違いについて、実装と表のどちらを正とするかを決める。
2. 表を変える場合は仕様を先に変え、実装を変える場合は表を固定するテストを先に書く。

## タスク

- [ ] T001 [Plan] 二つの食い違いの正を決める。
- [ ] T002 [Spec] 表を変える場合は状態遷移表を更新する。
- [ ] T003 [App] 実装を変える場合は、表を固定するテストを書いてから直す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- 削除予約中の User の無効化を拒否すると、その操作に頼る管理者の手順が失敗する。
