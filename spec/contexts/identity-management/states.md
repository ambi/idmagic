# IdManagement State Transitions

## UserLifecycle

User Aggregate のライフサイクル。`Active` は通常稼働、Disable は復元可能な無効化を表す。SoftDelete で削除予約 (`PendingDeletion`) に入り、猶予期間内は Restore で `Active` に戻せる。Purge では Tombstone 化し、匿名化をカスケードする。`Deleted` は終端状態であり復元できない。`PendingDeletion` から `Deleted` への遷移ガードは、猶予期間（業界で一般的な 7〜30 日に合わせたデフォルト 30 日）が経過したこと、または管理者が明示的に `purge=true` を指定したことのいずれかを要求する。猶予期間の経過前に `purge=false` で PendingDeletion のユーザーに対して呼び出した場合、`DeleteAdminUser.requires` が InvalidRequestError で拒否し、この遷移は発生しない。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 通常稼働。認証を許可するのはこの状態だけである |
| Disabled | — | 復元可能な無効化 |
| PendingDeletion | — | 削除予約。猶予期間内は復元できる |
| Deleted | terminal | Tombstone 化して匿名化した。復元できない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | UserDisabled | — | Disabled |  |
| Disabled | UserEnabled | — | Active |  |
| Active | UserSoftDeleted | — | PendingDeletion |  |
| Disabled | UserSoftDeleted | — | PendingDeletion |  |
| PendingDeletion | UserRestored | — | Active |  |
| PendingDeletion | UserDeleted | input.purge == true \|\| duration_since(status_changed_at) >= duration('2592000s') | Deleted | UserDeleted |
| Active | UserDeleted | — | Deleted |  |
| Disabled | UserDeleted | — | Deleted |  |

## DynamicMembershipEvaluationLifecycle

全件再評価は queued から running を経て succeeded または failed へ終端する。

| State | Kind | Meaning |
|---|---|---|
| queued | initial | 全件再評価を受理した。実行を待つ |
| running | — | 全件再評価を実行している |
| succeeded | terminal | 全件再評価が完了した |
| failed | terminal | 全件再評価が失敗した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | DynamicMembershipEvaluationStarted | — | running |  |
| running | DynamicMembershipEvaluated | — | succeeded |  |
| running | DynamicMembershipEvaluationFailed | — | failed |  |

## AgentLifecycle

Agent Aggregate のライフサイクル。`Active` は通常稼働、Disable は復元可能な運用停止、Kill は一方向の終端となる緊急停止を表す。`Killed` は終端状態で復元できず、`Active` 以外には新しいトークンを発行しない（フェイルクローズ）。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 通常稼働。新しいトークンを発行できる唯一の状態である |
| Disabled | — | 復元可能な運用停止 |
| Killed | terminal | 一方向の緊急停止。復元できない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | AgentDisabled | — | Disabled |  |
| Disabled | AgentEnabled | — | Active |  |
| Active | AgentKilled | — | Killed |  |
| Disabled | AgentKilled | — | Killed |  |

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
