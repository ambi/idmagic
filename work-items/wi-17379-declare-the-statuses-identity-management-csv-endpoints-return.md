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
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-039 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-041 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-087 }
  - { path: docs/domain/identity-management/data-export/README.md, requirement: REQ-IDMANAGEMENT-088 }
  - { path: docs/domain/identity-management/user-csv/README.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-026 }
  - { path: docs/domain/identity-management/group-csv/README.md, requirement: REQ-IDMANAGEMENT-029 }
---

# IdManagement の CSV のインポートとエクスポートの操作が返すステータスを TypeSpec に宣言する

## 動機

wi-83002 で機能仕様を書き直したとき、CSV のインポートとエクスポートの拒否を、実装とテストが返すステータスとエラーコードで書いた。
TypeSpec は、それらの多くを宣言していない。

| 操作 | 実装が返すもの | TypeSpec の宣言 |
| --- | --- | --- |
| エクスポートの開始 | 422 `invalid_columns`、422 `invalid_filter`、429 `active_job_quota_exceeded` | 400 `InvalidRequestError` |
| エクスポートの参照、ダウンロード、取り消し | 404 `data_export_not_found`、409 `data_export_not_downloadable`、409 `data_export_not_cancelable` | 400 `InvalidRequestError` |
| User、Group、メンバーシップのインポートの適用 | 404 `*_import_not_found`、409 `preview_not_ready`、409 `preview_digest_mismatch` | 400 `InvalidRequestError` |

`data_export_not_found`、`data_export_not_downloadable`、`data_export_not_cancelable`、`preview_not_ready`、`preview_digest_mismatch` は、TypeSpec のどのエラーにも現れない。
生成したクライアントは、宣言にない応答を扱えない。

## 対象範囲

- 表の各操作について、実装の応答を TypeSpec に宣言するか、実装を TypeSpec の宣言に合わせるかを決める。
- 決めた側に合わせて、TypeSpec、実装、テストを更新する。
  要件は、実装の現在の応答で書いてある。実装を変える場合は要件も改める。

## 対象外

- 表にない操作の宣言の漏れ。

## 計画

1. 操作ごとに、宣言を足すか実装を変えるかを決める。
2. TypeSpec、実装、テスト、要件を更新する。

## タスク

- [ ] T001 [Plan] 操作ごとに合わせる側を決める。
- [ ] T002 [Contract] TypeSpec を更新する。
- [ ] T003 [App] 実装を変える操作の実装とテストを更新する。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run verify`

## リスク

- 実装を宣言に合わせると、管理 UI がステータスで分けている表示が変わる。
  管理 UI の分岐を同じ変更で確かめる。
