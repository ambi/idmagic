# エージェントの状態遷移

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
