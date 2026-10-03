---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-03
priority: p2
depends_on: [wi-96676-transcribe-implicit-specifications-of-identity-management]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/identity-management/bulk-transfer/user-csv/README.md, requirement: REQ-IDMANAGEMENT-007, impact: conforms }
  - { path: docs/domain/identity-management/bulk-transfer/group-csv/README.md, requirement: REQ-IDMANAGEMENT-027, impact: conforms }
  - { path: docs/domain/identity-management/bulk-transfer/group-csv/README.md, requirement: REQ-IDMANAGEMENT-030, impact: conforms }
---

# CSV のエクスポートが転送ポリシーのどの上限を超えても `csv_transfer_limit_exceeded` で失敗させる

## 動機

EX-IDMANAGEMENT-007-03、EX-IDMANAGEMENT-027-03、EX-IDMANAGEMENT-030-03 は、生成結果が実効 `CsvTransferPolicy` のいずれかの上限を超えると、エクスポートが `csv_transfer_limit_exceeded` で失敗すると定める。
実装は、成果物の大きさの超過だけを `csv_transfer_limit_exceeded` にし、行数と項目の大きさの超過を `export_failed` にする。

IdManagement の既存コードを書き起こしたとき、行数の上限を 1 にした User のエクスポートが `export_failed` で、成果物の大きさの上限を 20 バイトにしたエクスポートが `csv_transfer_limit_exceeded` で失敗することを観測した。
既存のテストは、ジョブの失敗の記録に `csv_transfer_limit_exceeded` を直接書き込んでおり、生成が返すコードを読んでいなかった。

## 対象範囲

- 行数と項目の大きさの超過も `csv_transfer_limit_exceeded` にする。
- 生成が返すコードを読むテストで、三つの上限を固定する。

## 対象外

- 上限のデフォルト値の変更。

## 計画

1. 三つの上限のそれぞれでエクスポートの生成が返すコードを読むテストを書き、RED を確かめる。
2. 失敗のコードの対応を直す。

## タスク

- [ ] T001 [Acceptance] 三つの上限の超過で生成が返すコードを読むテストを書き、RED を確かめる。
- [ ] T002 [App] 失敗のコードの対応を直す。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- `export_failed` を条件にした利用者の処理があれば、コードの変更で分岐が変わる。リリースノートで知らせる。
