# IdManagement の状態遷移

## DataExportLifecycle

リソースエクスポートのライフサイクル。`queued` で受理され、`worker` プロセスが `running` で CSV を生成する。成功するとダウンロード可能な `succeeded`、失敗すると不完全なファイルをダウンロードできない `failed` で終了する。終了前は `canceled` で取り消せる。`succeeded` は保持期限を過ぎると `expired` へ遷移し、ファイル本体を完全削除してメタデータと監査記録だけを残す。`succeeded` から `expired` への遷移ガードは、Jobs のデフォルトの記録保持期間と同じ 30 日の経過を要求する。

| State | Kind | Meaning |
|---|---|---|
| queued | initial | 受理済み。`worker` の実行を待つ |
| running | — | `worker` が CSV を生成している |
| succeeded | — | 生成が完了し、保持期限までダウンロードできる |
| failed | terminal | 生成に失敗した。不完全なファイルはダウンロードできない |
| canceled | terminal | 終了前に取り消した |
| expired | terminal | 保持期限を過ぎ、ファイル本体を完全削除した。メタデータと監査記録だけが残る |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DataExportStarted | — | running |  |
| running | DataExportSucceeded | — | succeeded |  |
| running | DataExportFailed | — | failed |  |
| queued | DataExportCanceled | — | canceled |  |
| running | DataExportCanceled | — | canceled |  |
| succeeded | DataExportExpired | duration_since(completed_at) >= duration('2592000s') | expired |  |
