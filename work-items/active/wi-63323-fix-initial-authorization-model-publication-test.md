---
status: pending
authors: [tn]
risk: medium
created_at: 2026-11-06
priority: p1
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/authorization/features/model.md, requirement: REQ-AUTHORIZATION-011 }
---

# 認可モデル未作成時のテンプレート公開試験の失敗を解消する

## 動機

`AdminAuthorizationModelEditPage` の `publishes the document viewer template when no model exists yet` は、確認操作の後に `Publish` ボタンを取得できず失敗する。
[[wi-32030-separate-openldap-tests-from-default-go-verification]] の影響検査で見つかり、対象ファイルだけの実行でも再現する。
同項目はフロントエンドを変更していない。

## 対象範囲

- モデル未作成時にテンプレートを選び、差分を確認して公開する経路と、試験の非同期処理の待機を照合する。
- 画面の振る舞いと試験のどちらに原因があるかを特定し、確認と公開の証拠を保って修正する。

## 対象外

- OpenLDAP 試験の実行区分。
- 原因を確かめずに試験をスキップすることと、公開の表明を削ること。

## 設計

既存の `REQ-AUTHORIZATION-011` を判断の起点にする。
確認画面の到達、最新モデルが 404 の場合の扱い、公開可能な状態を観測し、画面の不具合か待機不足かを切り分けてから修正範囲を決める。
待機時間の固定値を増やす案は、待つ条件を表現できないため採らない。

## 計画

1. 対象試験で `Publish` を取得できない状態を再現する。
2. 公開可能な状態へ進む条件を確認し、仕様との照合後に修正する。
3. 対象ファイルと UI 検証を通す。

## タスク

- [ ] T001 [Acceptance] モデル未作成時の確認と公開の試験を使って原因を特定する。
- [ ] T002 [App] 必要な修正を実装する。
- [ ] T003 [Verify] 対象試験と UI 検証を通す。

## 検証

- `mise run test-ui-unit-file -- src/features/admin-authorization/AdminAuthorizationModelEditPage.test.tsx`：単独での再現と修正の確認。
- `mise run verify-ui`：画面と試験への影響の確認。

## リスク

待機不足だと決めつけて表明を弱めると、公開できない画面の不具合を隠す。
期待する画面状態と POST の内容を保って原因を判定する。
