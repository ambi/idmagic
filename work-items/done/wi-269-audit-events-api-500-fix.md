---
status: pending
authors: [Antigravity]
risk: low
created_at: 2026-07-20
depends_on: []
---

# 監査イベント取得APIの 500 Internal Server Error 修正

## Motivation
管理画面の監査イベント画面を開くと Internal Server Error が発生している。`GET /api/admin/audit_events` が 500 を返していることが原因とみられ、ユーザーが監査ログを確認できない状態になっているため、早急に修正が必要。

## Scope
- `backend/` の監査イベント取得API (`GET /api/admin/audit_events`) に関連するハンドラやユースケース、リポジトリ層

## Out of Scope
- 監査イベントのスキーマ変更
- 監査イベント自体の記録ロジックの変更

## Plan
- まずAPIの処理においてどこでパニックやエラーが発生しているかを特定する。
- 原因を修正し、正常にレスポンスが返るようにする。

## Tasks
- [ ] T001 [App] 監査イベント取得APIのエラー原因を調査・特定する。
- [ ] T002 [App] 原因となっているコードを修正する。
- [ ] T003 [Verify] 監査イベント画面を開き、データが正常に取得・表示されることを確認する。

## Verification
- `just dev` でサーバーを起動し、監査イベント画面にアクセスして正常に表示されるか確認する。
- `just test-go` でバックエンドのテストが通ることを確認する。

## Risk Notes
既存データの読み取り処理の修正であるためリスクは低い。
