---
status: pending
authors: ["tn"]
risk: low
reversibility: reversible
created_at: 2026-10-31
priority: p2
depends_on: []
change_kind: maintenance
spec_impact:
  kind: none
  reason: "確認ダイアログのフォーカスの扱いと Esc だけを揃える。各操作の確認の文言、API の要求、保存する状態、イベントは変えない。"
---

# 管理画面の確認ダイアログをキーボードで操作できる共通部品へ揃える

## 動機
[ユーザーインターフェース設計](../../docs/design/application/user-interface.md#アクセシビリティ)は、すべての操作をキーボードだけで実行できることを求める。
`aria-modal` を付けた既存のダイアログは各画面が手で組み立てており、フォーカスの閉じ込めがなく、初期フォーカスと Esc の扱いも画面ごとに異なる。
背後の要素へフォーカスが移ると、スクリーンリーダーとキーボードの利用者は確認の途中で文脈を失う。

細粒度認可の画面は `AdminAuthorizationShared.tsx` の `DestructiveConfirmDialog` で Base UI の Dialog を使っており、二つ目の機能が使う時点で `components/` へ移す条件に当たる。

## 対象範囲
- `DestructiveConfirmDialog` を基に共通の確認ダイアログを `frontend/src/components/` へ置く。
- 次の画面のダイアログを移す。
  - `admin-workload-identity/AdminWorkloadIdentityShared.tsx`
  - `admin-tenants/AdminTenantGroupAttributesPage.tsx`、`admin-tenants/AdminTenantAttributesPage.tsx`
  - `admin-groups/AdminGroupImportResult.tsx`、`admin-groups/AdminGroupMemberImportResult.tsx`
  - `admin-users/AdminUserImportResult.tsx`、`admin-users/AdminUserDialogs.tsx`
  - `admin-lifecycle-workflows/AdminLifecycleWorkflowsPage.tsx`
  - `admin-identity-providers/AdminIdentityProvidersShared.tsx`

## 対象外
- `StepUpDialog`。認証の入力を持ち、再認証の遷移の扱いが別なので、確認ダイアログへ統合しない。同じ Dialog の土台へ移すかは着手時に判断し、移さないならフォーカスの閉じ込めの不足を別の記録へ残す。
- `window.confirm` を使う画面の置き換え。残る箇所は着手時に棚卸しし、別の記録へ分ける。

## 設計
共通部品の props は、題名、対象の識別名、操作の結果、取り消せるかどうか、確定の文言、実行中、閉じる、確定とする。
[破壊的な操作](../../docs/design/application/user-interface.md#破壊的な操作)が求める三つの内容を必須の props にする。
入力欄を持つダイアログのために子要素を受けられるようにし、確認の文言を自由な子要素だけで組み立てる形にはしない。

## 計画
1. 共通部品を移し、細粒度認可の画面を先に切り替える。
2. 残りの画面を一つずつ移し、各画面のテストを通す。

## タスク
- [ ] T001 [UI] 初期フォーカス、Tab の閉じ込め、Esc、閉じた後のフォーカスの復帰を共通部品のテストで確かめる。
- [ ] T002 [UI] 各画面を移し、確定と取り消しの作用が変わらないことを確かめる。

## 検証
- `mise run verify-ui`
  - reason: 複数の機能スライスの画面へ届くため。
- `mise run test-ui-e2e`
  - reason: ダイアログを操作する既存の E2E が Portal への移動で壊れないことを確かめるため。

## リスク
Dialog は Portal へ描画するので、親の要素を前提にした E2E のセレクターが壊れる可能性がある。
