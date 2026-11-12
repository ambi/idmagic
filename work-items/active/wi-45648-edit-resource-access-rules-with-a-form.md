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

MCP リソースサーバーの編集画面（`frontend/src/features/admin-mcp-resource-servers/AdminMcpResourceServersShared.tsx` の `AccessRuleRows`）は、起票後にツール名、表示名、対象の引数、資源の種類、関係、スコープを行で編集できるようになった（2026-11-13 に確認）。
残りの項目は、詳細設定に折りたたんだ JSON の入力でしか編集できない。

| 項目 | 現在の編集手段 |
| --- | --- |
| `requires_approved_operation` | JSON だけ。一覧の画面（`AdminAgentResourcesPage`）は表示だけを行う |
| `authorization_detail_type` | JSON だけ |
| `limits` | JSON だけ |
| `token_exchange_resources` | JSON だけ |
| `condition` | JSON だけ |

承認の要否の入力を誤ると、高影響の操作が承認なしで許可され得る。
条件の評価（[[wi-11371-evaluate-runtime-conditions-on-resource-access]]）は完了したので、条件も同じ画面の判断に含める。

## 対象範囲

- 承認の要否、明細の型、実行の上限、交換先のリソースを、規則の行の入力欄で編集する。
- 承認が必要なのに明細の型がない状態を、送信前に示す。
- 条件は、入力欄で表せる形の範囲を T001 で決める。表せない条件を持つ規則は、行で読み取り専用に示し、JSON の入力で編集させる。

## 対象外

- API の変更。
- 承認の起票の経路（[[wi-62682-start-human-approval-from-the-mcp-gateway]]）。この項目は規則の入力だけを扱う。

## 設計

検証の正はサーバーに置き、画面は同じ規則を先に示すだけにする。
既存の `AccessRuleRows` を拡張し、行にない項目を保つ現在の方針（JSON を正とする）を変えない。
詳細設定の JSON の入力は、表せない値の逃げ道として残す。

## 計画

既存の行の編集を拡張し、E2E で保存結果を確かめる。

## タスク

- [ ] T001 [Design] 条件を入力欄で表す範囲と、表せない規則の表示を決める。
- [ ] T002 [UI] 承認の要否、明細の型、上限、交換先を行の入力欄へ加える。
- [ ] T003 [Acceptance] 承認の要否の保存と、不整合の表示を E2E で確かめる。行で編集した後も、行にない項目が保存に残ることを確かめる。
- [ ] T004 [Verify] キーボード操作とエラーの読み上げを確かめる。

## 検証

- `mise run test-ui-e2e-file`。ブラウザーを使うのでサンドボックスの外で実行する。
- 最終検査は `mise run verify-changed -- <着手時の基準リビジョン>` とする。

## リスク

入力欄で表せない規則を保存すると値を失う。
API の全項目を入力欄へ対応させる。
