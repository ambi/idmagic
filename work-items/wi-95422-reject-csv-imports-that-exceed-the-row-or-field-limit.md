---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: [wi-96676-transcribe-implicit-specifications-of-identity-management]
change_kind: bugfix
affected_spec:
  - { path: docs/domain/identity-management/user-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-004, impact: conforms }
  - { path: docs/domain/identity-management/group-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-026, impact: conforms }
  - { path: docs/domain/identity-management/group-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-029, impact: conforms }
---

# 行数または項目の上限を超える CSV のインポートを、一部も適用せずに拒否する

## 動機

EX-IDMANAGEMENT-004-02、EX-IDMANAGEMENT-026-02、EX-IDMANAGEMENT-029-02 は、CSV が実効 `CsvTransferPolicy` の `max_bytes`、`max_rows`、`max_field_bytes` のいずれかを超えると、インポートの投入を拒否すると定める。
実装が投入の時点で同期的に拒否するのは `max_bytes` だけである。
`max_rows` と `max_field_bytes` は、プレビューのジョブの中で解析が上限に達した位置で初めて検出される。
ジョブはその位置までの行を計画したうえで、上限の超過を一件のエラーとして記録し、成功として終わる。

IdManagement の既存コードを書き起こしたとき、`max_rows` を 1 にした User のインポートで 2 行を投入し、プレビューが成功して `created_rows=1`、`error_total=1` を返すこと、そのプレビューの適用が 1 行目の User を作成することを観測した。
上限の超過は、利用者には「ファイルが拒否された」ではなく「先頭の行だけが取り込まれた」として現れる。
Group とメンバーシップのインポートのジョブも、同じ分岐で CSV のエラーを記録して成功する。

## 対象範囲

- `max_rows` と `max_field_bytes` の超過で、プレビューのジョブを失敗させ、適用できない状態にする。
- 適用のジョブも、解析が上限に達したときに先行する行を確定しない形にする。
- User、Group、メンバーシップの三つのインポートで同じ契約にする。

## 対象外

- 上限のデフォルト値の変更。

## 計画

1. 三つのインポートで、上限を超えるファイルのプレビューが成功しないことを確かめるテストを書き、RED を確かめる。
2. 上限の超過をファイル全体の拒否として扱い、プレビューを適用できない状態にする。

## タスク

- [ ] T001 [Acceptance] 上限を超えるファイルのプレビューと適用のテストを三つのインポートについて書き、RED を確かめる。
- [ ] T002 [App] 上限の超過をファイル全体の拒否にする。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- 適用のジョブで上限の超過を検出する時点では、先行する行がすでに確定している可能性がある。プレビューで拒否すれば適用へ進めないので、プレビューでの拒否を主にする。
