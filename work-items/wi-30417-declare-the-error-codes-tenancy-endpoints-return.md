---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-04
priority: p3
depends_on: []
change_kind: maintenance
affected_spec:
  - { path: docs/modules/tenancy/lifecycle/README.md, requirement: REQ-TENANCY-025 }
  - { path: docs/modules/tenancy/branding/README.md, requirement: REQ-TENANCY-032 }
  - { path: docs/modules/tenancy/notification-template/README.md, requirement: REQ-TENANCY-017 }
  - { path: docs/modules/tenancy/notification-template/README.md, requirement: REQ-TENANCY-038 }
---

# Tenancy のエンドポイントが返すエラーコードを TypeSpec に宣言する

## 動機

wi-71372 で Tenancy の機能仕様を書き直したとき、要件には実装とテストが固定している応答を書いた。
そのうち三つのエラーコードは、TypeSpec のどのエラーにも現れない。

| 操作 | 実装が返すもの | TypeSpec の宣言 |
| --- | --- | --- |
| テナントの作成 | 409 `tenant_conflict` | 400 `InvalidRequestError` だけ。409 を宣言していない |
| ブランド設定の更新 | 400 `invalid_branding` | 400 `InvalidRequestError` |
| 通知テンプレートの上書きの保存、プレビュー | 400 `invalid_notification_template` | 400 `InvalidRequestError` |

`check-unspecified-vocabulary` は TypeSpec が宣言するエラーコードから要件を探すので、宣言のないコードは検査の対象にならない。
生成したクライアントは、宣言にない応答を扱えない。

## 対象範囲

- 表の各操作について、実装の応答を TypeSpec に宣言するか、実装を TypeSpec の宣言に合わせるかを決める。
- 決めた側に合わせて、TypeSpec、実装、テストを更新する。
  要件は、実装の現在の応答で書いてある。実装を変える場合は要件と例の付録も改める。

## 対象外

- 表にない操作の宣言の漏れ。
- IdManagement の宣言の漏れ。wi-17379 が扱う。

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
- `mise run check-unspecified-vocabulary`
- `mise run verify`

## リスク

- 実装を宣言に合わせると、管理 UI がエラーコードで分けている表示が変わる。
  管理 UI の分岐を同じ変更で確かめる。
