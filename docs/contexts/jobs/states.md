# Jobs State Transitions

## JobLifecycle

`worker` が取得すると `queued` から `running` へ遷移する。実行に失敗した場合、`attempts` が `max_attempts` 未満ならバックオフを伴って `queued` に戻り、上限に達していれば配信不能を表す `failed` へ遷移する。`succeeded`、`failed`、`canceled` は不可逆の終端状態である。

| State | Kind | Meaning |
|---|---|---|
| queued | initial | 取得待ち。`run_at` に達すると自分のレーンの `worker` が取得できる |
| running | — | `worker` がリースを持ってハンドラーを実行している |
| succeeded | terminal | ハンドラーが正常終了した |
| failed | terminal | 試行上限に達し、配信不能として確定した |
| canceled | terminal | 終端に達する前に取り消した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | JobStarted | — | running |  |
| running | JobSucceeded | — | succeeded |  |
| running | JobFailed | attempts >= max_attempts | failed |  |
| running | JobRetried | attempts < max_attempts | queued |  |
| queued | JobCanceled | — | canceled |  |
| running | JobCanceled | — | canceled |  |
