---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-586-frontend-architecture-and-user-interface-design]
change_kind: refactor
spec_impact:
  kind: none
  reason: "確認の表示手段を画面の部品へ揃えるだけで、確認を求める操作と結果は変えない。"
---

# 破壊的な操作の確認を `window.confirm` から画面のダイアログへ移す

## Motivation

[ユーザーインターフェース設計](../docs/design/application/user-interface.md#破壊的な操作)は、破壊的な操作の確認に画面の部品として作ったダイアログを使い、対象、結果、取り消せるかどうかを示すと定める。
次の箇所は、ブラウザーの `window.confirm` で確認している。

| ファイル | 操作 |
| --- | --- |
| `frontend/src/features/admin-settings/NotificationTemplatesTab.tsx` | 通知テンプレートの初期化 |
| `frontend/src/features/system-tenants/TenantEndpointStyleEditor.tsx` | テナントのエンドポイント形式の変更 |
| `frontend/src/features/admin-jobs/AdminJobsPage.tsx` | ジョブの取り消し |
| `frontend/src/features/admin-jobs/SystemJobsPage.tsx` | ジョブの取り消し |
| `frontend/src/features/admin-users/AdminUsersShared.tsx` | ユーザーの全セッションの取り消し |

`window.confirm` は、ブラウザーの言語と見た目で表示され、対象と結果を構造立てて示せない。
また、E2E テストでは確認を自動で受け入れる設定が要る。

## Scope

- 5 か所それぞれについて、`window.confirm` を選んだ理由があるかを確認する。
- 理由がなければ、確認ダイアログの共通部品を `frontend/src/components/` に作り、5 か所をそれに置き換える。
- 理由があれば、例外として[ユーザーインターフェース設計](../docs/design/application/user-interface.md#破壊的な操作)に条件を書く。
- 置き換えた操作の単体テストで、ダイアログに対象と結果が表示され、取り消すと操作が実行されないことを確かめる。

## Out of Scope

- 既存の専用ダイアログ（ユーザーの削除など）を共通部品へ置き換えること。
- 確認を求める操作の範囲の変更。

## Design

共通部品は、タイトル、対象の識別名、結果の説明、取り消せるかどうか、確定と取り消しのボタンの文言を受け取る。
Base UI のダイアログ部品を使い、フォーカスの移動と Escape キーによる取り消しを部品に任せる。
`window.confirm` の同期的な戻り値に依存している呼び出し側（`confirmCancel={() => window.confirm(...)}`）は、確定のコールバックを受け取る形へ変える。

## Plan

1. 5 か所の導入時のコミットを読み、`window.confirm` を選んだ理由を確認する。
2. 共通部品を作り、単体テストで表示と取り消しを確かめる。
3. 1 か所ずつ置き換え、各画面のテストを RED から GREEN にする。

## Tasks

- [ ] T001 [Design] `window.confirm` を選んだ理由を確認する。
- [ ] T002 [Acceptance] 置き換える操作の単体テストで、ダイアログが表示されないことによる RED を確認する。
- [ ] T003 [App] 共通部品を作り、5 か所を置き換える。
- [ ] T004 [Verify] `mise run verify-ui` と `mise run test-ui-e2e` を通す。

## Verification

- `mise run verify-ui`
- `mise run test-ui-e2e`

## Risk Notes

E2E テストが `window.confirm` を自動で受け入れる前提で書かれている場合、置き換え後にダイアログを操作する手順が必要になる。
