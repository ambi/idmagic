---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-04
priority: p2
group: api-contract
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/identity-management/features/data-export.md, requirement: REQ-IDMANAGEMENT-039 }
  - { path: docs/modules/identity-management/features/data-export.md, requirement: REQ-IDMANAGEMENT-041 }
  - { path: docs/modules/identity-management/features/data-export.md, requirement: REQ-IDMANAGEMENT-087 }
  - { path: docs/modules/identity-management/features/data-export.md, requirement: REQ-IDMANAGEMENT-088 }
  - { path: docs/modules/identity-management/features/user-csv.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: docs/modules/identity-management/features/group-csv.md, requirement: REQ-IDMANAGEMENT-026 }
  - { path: docs/modules/identity-management/features/group-csv.md, requirement: REQ-IDMANAGEMENT-029 }
  - { path: docs/modules/tenancy/features/lifecycle.md, requirement: REQ-TENANCY-025 }
  - { path: docs/modules/tenancy/features/branding.md, requirement: REQ-TENANCY-032 }
  - { path: docs/modules/tenancy/features/notification-template.md, requirement: REQ-TENANCY-017 }
  - { path: docs/modules/tenancy/features/notification-template.md, requirement: REQ-TENANCY-038 }
---

# IdManagement と Tenancy の操作が返すステータスとエラーコードを TypeSpec に宣言する

## 動機

wi-83002 と wi-71372 で IdManagement と Tenancy の機能仕様を書き直したとき、要件には実装とテストが返すステータスとエラーコードを書いた。
TypeSpec は、それらの多くを宣言していない（2026-11-14 にも、表のエラーコードは TypeSpec に一つも現れなかった）。

| モジュール | 操作 | 実装が返すもの | TypeSpec の宣言 |
| --- | --- | --- | --- |
| IdManagement | エクスポートの開始 | 422 `invalid_columns`、422 `invalid_filter`、429 `active_job_quota_exceeded` | 400 `InvalidRequestError` |
| IdManagement | エクスポートの参照、ダウンロード、取り消し | 404 `data_export_not_found`、409 `data_export_not_downloadable`、409 `data_export_not_cancelable` | 400 `InvalidRequestError` |
| IdManagement | User、Group、メンバーシップのインポートの適用 | 404 `*_import_not_found`、409 `preview_not_ready`、409 `preview_digest_mismatch` | 400 `InvalidRequestError` |
| Tenancy | テナントの作成 | 409 `tenant_conflict` | 400 `InvalidRequestError` だけ。409 を宣言していない |
| Tenancy | ブランド設定の更新 | 400 `invalid_branding` | 400 `InvalidRequestError` |
| Tenancy | 通知テンプレートの上書きの保存、プレビュー | 400 `invalid_notification_template` | 400 `InvalidRequestError` |

生成したクライアントは、宣言にない応答を扱えない。
`check-unspecified-vocabulary` は TypeSpec が宣言するエラーコードから要件を探すので、宣言のないコードは検査の対象にもならない。
二つのモジュールの漏れは同じ種類で、同じ手順で直すので一つの作業にする。

## 対象範囲

- 表の各操作について、実装の応答を TypeSpec に宣言するか、実装を TypeSpec の宣言に合わせるかを決める。
- 決めた側に合わせて、TypeSpec、実装、テストを更新する。
  要件は実装の現在の応答で書いてある。実装を変える場合は要件と例の付録も改める。

## 対象外

- 表にない操作の宣言の漏れ。漏れを機械的に見つける検査は wi-12283 が扱う。

## 検証

- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run check-unspecified-vocabulary`
- `mise run verify`

## リスク

- 実装を宣言に合わせると、管理 UI がステータスやエラーコードで分けている表示が変わる。
  管理 UI の分岐を同じ変更で確かめる。
