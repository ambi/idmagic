---
status: pending
authors: ["tn"]
risk: low
reversibility: reversible
created_at: 2026-10-31
priority: p2
group: frontend-ui
depends_on: []
change_kind: maintenance
spec_impact:
  kind: none
  reason: "確認ダイアログのフォーカスの扱いと Esc だけを揃える。各操作の確認の文言、API の要求、保存する状態、イベントは変えない。"
---

# 管理画面の確認ダイアログをキーボードで操作できる共通部品へ揃える

## 動機
[ユーザーインターフェース設計](../../docs/design/application/user-interface.md#アクセシビリティ)は、すべての操作をキーボードだけで実行できることを求める。
`frontend/src/components/ConfirmActionDialog.tsx` は Base UI の Dialog で作った共通の確認ダイアログであり、17 ファイルが使っている。
一方、次の画面は `aria-modal` を付けたダイアログを手で組み立てており、フォーカスの閉じ込めがなく、初期フォーカスと Esc の扱いも画面ごとに異なる（2026-11-14 に確認）。
背後の要素へフォーカスが移ると、スクリーンリーダーとキーボードの利用者は確認の途中で文脈を失う。

- `admin-workload-identity/AdminWorkloadIdentityShared.tsx`
- `admin-groups/AdminGroupImportResult.tsx`、`admin-groups/AdminGroupMemberImportResult.tsx`
- `admin-users/AdminUserImportResult.tsx`、`admin-users/AdminUserDialogs.tsx`
- `admin-lifecycle-workflows/AdminLifecycleWorkflowsPage.tsx`
- `admin-identity-providers/AdminIdentityProvidersShared.tsx`

細粒度認可の画面は、別の確認ダイアログ `DestructiveConfirmDialog`（`admin-authorization/AdminAuthorizationShared.tsx`）を持つ。
同じ役割の部品が二つあると、片方だけが直る。

## 対象範囲
- 上の画面の確認ダイアログを `ConfirmActionDialog` へ移す。入力欄を持つダイアログのために、`ConfirmActionDialog` が子要素を受けられるようにする。
- `DestructiveConfirmDialog` を `ConfirmActionDialog` へ統合する。[破壊的な操作](../../docs/design/application/user-interface.md#破壊的な操作)が求める対象、結果、取り消せるかどうかの表示を、統合した部品の必須の入力にする。
- 初期フォーカス、Tab の閉じ込め、Esc、閉じた後のフォーカスの復帰を、共通部品のテストで確かめる。

## 対象外
- `StepUpDialog`。認証の入力を持ち、再認証の遷移の扱いが別なので、確認ダイアログへ統合しない。同じ Dialog の土台へ移すかは着手時に判断し、移さないならフォーカスの閉じ込めの不足を別の記録へ残す。
- 確認を求める操作の範囲と、確認の文言の変更。

## 検証
- `mise run verify-ui`
  - reason: 複数の機能スライスの画面へ届くため。
- `mise run test-ui-e2e`
  - reason: ダイアログを操作する既存の E2E が Portal への移動で壊れないことを確かめるため。

## リスク
Dialog は Portal へ描画するので、親の要素を前提にした E2E のセレクターが壊れる可能性がある。
