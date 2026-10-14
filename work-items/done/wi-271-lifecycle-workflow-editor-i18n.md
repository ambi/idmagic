---
status: completed
archived_from: a5a4102cbb7193dcce1ce2bc5d071f052abde613
---

# ライフサイクルワークフローの編集・作成フォーム画面を日英対応にする

ライフサイクルワークフローの作成・編集フォーム（`WorkflowDefinitionForm`）とエディタ画面（`AdminLifecycleWorkflowEditorPage`）のハードコード日本語を、新規辞書 `WorkflowDefinitionForm.i18n.ts`（JA/EN）による日英対応へ移行した。フォームの全ラベル・プレースホルダ・説明・トリガー/アクション/状態/表示の選択肢・操作ボタン・aria-label・バリデーションエラー・エディタのタイトル/説明/戻る導線/失敗メッセージを辞書化。ユーザー可視文字列を返す純粋ヘルパー（`validateWorkflowDraft` / `workflowStatusLabel` / `workflowTriggerLabel` / `workflowActionLabel`）は辞書を引数に取る形へ変更（`workflowInput` / `compactAction` は文字列を生成しないため据え置き）。SCL の振る舞い変更は無く `spec/scl.yaml` は未更新。

全文は `git show a5a4102cbb7193dcce1ce2bc5d071f052abde613:<このファイルのパス>` で読む。
