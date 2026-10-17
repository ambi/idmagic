---
status: completed
---

# 実装が返すエラーコードを、返す Context の TypeSpec で宣言する

既存の応答に合わせ、invalid_icon、invalid_sign_in_policy、agent_workload_binding_pattern_conflict を TypeSpec のエラーの和へ加えた。
WorkloadIdentity の二つの not-found の宣言はすでに対応済みだった。
JobsUnavailableError の名前を保って IdManagement へ移し、REQ-IDMANAGEMENT-039 に未構成時の503と無作用を記録して語彙の負債から外した。
本番コードの振る舞いは変えず、三種類のエクスポート開始の Problem Details と、ジョブとイベントを残さないことを固定した。
保存済みの公開前ベースラインとの互換性検査は123件を指摘するが、main と今回の診断は完全に同一であり、新しい互換性違反はない。
公開状態に従う必須の互換性検査は成功し、ベースラインは更新していない。

全文は `git log -1 -- work-items/active/wi-12645-declare-the-error-codes-the-implementation-returns.md` が示すコミットの親、またはそのコミットで読む。
