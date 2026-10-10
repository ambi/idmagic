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
  - { path: docs/modules/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-086 }
---

# データエクスポートの結果のイベントを、エクスポートが実際に至った結果に対してだけ発行する

## 動機

wi-83002 で DataExportLifecycle に状態遷移表を加えたとき、エクスポートの状態と、発行する結果のイベントが食い違う組が見つかった。
仕様には現在の挙動を書いた。

| 状況 | 状態 | 発行するイベント |
| --- | --- | --- |
| 生成が失敗し、`Jobs` の試行の上限に達していない | `queued` に戻る | `DataExportFailed` |
| 取り消した後に、取り消す前に始めた生成が成功した | `canceled` のまま | `DataExportSucceeded` |
| 取り消した後に、取り消す前に始めた生成が失敗した | `canceled` のまま | `DataExportFailed` |

生成のハンドラーが、ジョブの状態を確定する前に結果のイベントを発行するためである。
監査の記録を読む人は、`failed` にも `succeeded` にもなっていないエクスポートを、失敗した、または成功したと読む。

## 対象範囲

- `DataExportSucceeded` と `DataExportFailed` を、エクスポートが `succeeded` または `failed` に至ったときだけ発行する。
- 再試行に戻る失敗を記録する必要があるかを決め、要るなら別のイベントか `Jobs` の `JobRetried` で表す。
- REQ-IDMANAGEMENT-086 と、DataExportLifecycle の遷移の表と状態遷移表を改める。

## 対象外

- `Jobs` の再試行の方針の変更。

## 計画

1. 結果のイベントを発行する時点を、ジョブの状態の確定の後へ移す設計を決める。
2. 要件、状態遷移表、実装、テストを更新する。

## タスク

- [ ] T001 [Plan] 発行の時点と、再試行に戻る失敗の記録の扱いを決める。
- [ ] T002 [Spec] 要件と状態遷移表を改める。
- [ ] T003 [App] 実装とテストを更新する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed`
- `mise run verify`

## リスク

- イベントの数が減ると、それを数える利用者の集計が変わる。
  リリースノートで知らせる。
