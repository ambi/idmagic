---
status: completed
authors: [tn]
risk: low
created_at: 2026-07-22
depends_on: []
---

# ライフサイクルワークフローの編集・作成フォーム画面を日英対応にする

## Motivation
`wi-268` でライフサイクルワークフローの**一覧ページ**は `AdminLifecycleWorkflowsPage.i18n.ts` により日英対応化したが、**編集・作成フォーム画面**（`AdminLifecycleWorkflowEditorPage` / `WorkflowDefinitionForm`）は依然すべてハードコードされた日本語のままである。管理コンソールの他画面は `defineDictionary` による日英対応が原則であり、この画面だけロケール切り替えが効かない。ハードコード日本語を排し、辞書経由の日英対応へ揃える。

## Scope
- `frontend/src/features/admin-lifecycle-workflows/WorkflowDefinitionForm.tsx`
  - フォームの全ラベル・プレースホルダ・説明文・トリガー/アクション/状態/表示の選択肢ラベル・アクション操作ボタン（上へ/下へ/削除/追加）・aria-label
  - バリデーションエラーメッセージを返す `validateWorkflowDraft`
  - 表示ラベルヘルパー `workflowStatusLabel` / `workflowTriggerLabel` / `workflowActionLabel`
- `frontend/src/features/admin-lifecycle-workflows/AdminLifecycleWorkflowEditorPage.tsx`
  - 画面タイトル・説明・戻る導線・作成/更新失敗時のエラーメッセージ
- 新規辞書 `frontend/src/features/admin-lifecycle-workflows/WorkflowDefinitionForm.i18n.ts`
- 上記に対応するテスト（`WorkflowDefinitionForm.test.tsx` / `AdminLifecycleWorkflowPages.test.tsx`）

SCL の振る舞い変更を伴わない presentation 層のみの i18n 対応のため `spec/scl.yaml` は更新しない。

## Out of Scope
- ワークフローのトリガー/アクションの種類そのものの追加・変更（振る舞い変更）
- 一覧ページ（`wi-268` で対応済み）
- バックエンドの文言

## Plan
- 一覧ページと同様に `defineDictionary` で JA/EN 辞書を作る。列挙値（trigger/action/status/visibility）はキー接頭辞方式（`trigger_*` / `action_*` など）で持ち、選択肢・説明・ラベルヘルパーを辞書から導出する。
- 現在 export されている純粋ヘルパー（`validateWorkflowDraft` / `workflowStatusLabel` / `workflowTriggerLabel` / `workflowActionLabel`）はユーザー可視文字列を返すため、辞書（`t`）を引数に取る形へ変更する。`workflowInput` / `compactAction` は文字列を生成しないため据え置く。
- テストは日本語ハードコード比較をやめ、辞書値（`dict.ja.*` / `dict.en.*`）を参照する（既存の i18n テスト方針に一致）。

## Tasks
- [x] T001 [App] `WorkflowDefinitionForm.i18n.ts` を新規作成（JA/EN、列挙ラベル・説明・エラー文含む）。
- [x] T002 [App] `WorkflowDefinitionForm.tsx` を辞書利用へ移行。ヘルパー (`workflowStatusLabel`/`workflowTriggerLabel`/`workflowActionLabel`) とバリデーション (`validateWorkflowDraft`) を `t` 受け取りに変更。選択肢配列は値のみの定数から辞書で組み立てる方式に。
- [x] T003 [App] `AdminLifecycleWorkflowEditorPage.tsx` を辞書利用へ移行（作成/編集タイトル・説明・戻る導線・エラーメッセージ）。
- [x] T004 [App] 関連テストを辞書値参照へ更新。フォームは JA/EN 両ロケールのレンダリング検証を追加し、純粋ヘルパー呼び出しへ `t` を渡す形へ修正。
- [x] T005 [Verify] `just verify-ui` と `just test-ui-e2e` を通す。

## Verification
- `just verify-ui`（format-check / lint / typecheck / unit test / build）
- `just test-ui-e2e`（ワークフロー編集導線を含む回帰確認）

## Risk Notes
presentation 層の文字列差し替えのみでリスクは低い。純粋ヘルパーのシグネチャ変更が波及するのはフォーム本体とテストに限られ、型検査で検出できる。

## Completion
- **Completed At**: 2026-07-22
- **Summary**:
  ライフサイクルワークフローの作成・編集フォーム（`WorkflowDefinitionForm`）とエディタ画面（`AdminLifecycleWorkflowEditorPage`）のハードコード日本語を、新規辞書 `WorkflowDefinitionForm.i18n.ts`（JA/EN）による日英対応へ移行した。フォームの全ラベル・プレースホルダ・説明・トリガー/アクション/状態/表示の選択肢・操作ボタン・aria-label・バリデーションエラー・エディタのタイトル/説明/戻る導線/失敗メッセージを辞書化。ユーザー可視文字列を返す純粋ヘルパー（`validateWorkflowDraft` / `workflowStatusLabel` / `workflowTriggerLabel` / `workflowActionLabel`）は辞書を引数に取る形へ変更（`workflowInput` / `compactAction` は文字列を生成しないため据え置き）。SCL の振る舞い変更は無く `spec/scl.yaml` は未更新。
- **Verification Results**:
  - `just verify-ui` (format-check / lint / typecheck / unit test / build) - passed（フォームの JA/EN 両ロケールのレンダリング検証をユニットテストに追加）
  - `just test-ui-e2e` (4 spec, 20 tests) - passed
