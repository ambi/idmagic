---
status: pending
authors: [tn]
risk: low
created_at: 2026-10-31
priority: p3
depends_on: []
change_kind: feature
affected_spec:
  - { path: docs/modules/oauth2/features/admin-access.md, requirement: REQ-OAUTH2-064 }
---

# リソースアクセスの規則を専用の入力欄で編集する

## 動機

管理コンソールの `access_rules` は JSON の入力であり、`requires_approved_operation` にも専用の入力がない。
承認の要否の入力を誤ると、高影響の操作が承認なしで許可され得る。

## 対象範囲

- 資源の種類、関係、スコープ、明細の型、承認の要否を入力欄で編集する。
- 承認が必要なのに明細の型がない状態を、送信前に示す。

## 対象外

- 条件式の編集（[[wi-11371-evaluate-runtime-conditions-on-resource-access]] の範囲で判断する）。
- API の変更。

## 設計

検証の正はサーバーに置き、画面は同じ規則を先に示すだけにする。
既存の管理画面の部品を使う。

## 計画

既存の JSON 入力を置き換え、E2E で保存結果を確かめる。

## タスク

- [ ] T001 [UI] 入力欄へ置き換える。
- [ ] T002 [Acceptance] 承認の要否の保存と、不整合の表示を E2E で確かめる。
- [ ] T003 [Verify] キーボード操作とエラーの読み上げを確かめる。

## 検証

- `mise run test-ui-e2e-file`。ブラウザーを使うのでサンドボックスの外で実行する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

入力欄で表せない規則を保存すると値を失う。
API の全項目を入力欄へ対応させる。
