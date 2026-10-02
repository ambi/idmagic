# 動的グループの状態遷移

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
