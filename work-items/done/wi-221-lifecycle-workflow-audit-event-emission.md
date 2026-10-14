---
status: completed
archived_from: 8e7af53a3da65c1f92eebfdcce0d11396df47f04
---

# lifecycle workflow の監査イベント発行を完成させる

`LifecycleWorkflowCreated`/`Updated`/`Enabled`/`Disabled`/`RunStarted`/`RunSucceeded`/
`RunPartiallyFailed`/`RunFailed`/`RunCanceled`/`StepFailed` の 9 event 型を追加し、対応する
usecase (Create/Update/Enable/Disable と run/step 実行経路、queued run cancel 経路) から発行
するようにした。`CancelQueuedByWorkflow` を canceled run を返す契約に変更し、Disable/Delete が
canceled run ごとに `RunCanceled` を発行する。監査検索 registry に `workflow.id` /
`workflow_run.id` / `workflow_step.id` を追加し、`ExtractSearchAttributes` が
`workflowId`/`runId`/`stepIndex` payload から抽出するようにした。run 終端 status 判定を
ADR-113 の「成功 0 件なら failed」ロジックに修正した。

全文は `git show 8e7af53a3da65c1f92eebfdcce0d11396df47f04:<このファイルのパス>` で読む。
