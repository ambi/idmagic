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
  - { path: docs/domain/identity-management/group-csv/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-027, impact: conforms }
---

# 管理 API からの Group の CSV エクスポートでテナント定義の属性の列を選べるようにする

## 動機

REQ-IDMANAGEMENT-027 の EX-IDMANAGEMENT-027-01 は、管理者がインポート可能な組み込み列と `custom:cost_center` を機械可読ヘッダーでエクスポートできると定める。
しかし、管理 API の開始の経路（`/api/admin/v1/groups/exports`）は、エクスポートの依存に Group の CSV の方言を配線していない。
そのため開始は種別に依存しない許可一覧で列を検証し、`custom:<key>` の列を `invalid_columns` で拒否する。

IdManagement の既存コードを書き起こしたとき、テナントの Group 属性スキーマに `cost_center` を定義した状態で、ハンドラーと同じ組み立ての依存から `StartDataExport` を呼び、`"custom:cost_center" is not allowlisted for target "group"` で拒否されることを観測した。
`worker` 側の生成は Group の方言を配線しているので、開始さえ通れば生成はできる。

## 対象範囲

- 管理 API の開始の依存に Group の CSV の方言を配線し、`custom:<key>` の列を受け付ける。
- 管理 API の入口から EX-IDMANAGEMENT-027-01 を固定するテストを加える。

## 対象外

- User とメンバーシップのエクスポートの列の変更。

## 計画

1. 管理 API の入口から `custom:<key>` の列で開始するテストを書き、RED を確かめる。
2. 開始の依存に Group の CSV の方言を配線する。

## タスク

- [ ] T001 [Acceptance] 管理 API の入口から `custom:<key>` の列で Group のエクスポートを開始するテストを書き、RED を確かめる。
- [ ] T002 [App] 開始の依存に Group の CSV の方言を配線する。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run check`
- `mise run verify`

## リスク

- 方言の検証は既存の許可一覧より列の語彙が広い。許可一覧が拒否していた列のうち、方言も拒否すべきものを取りこぼさないかを確かめる。
