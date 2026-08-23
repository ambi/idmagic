# SharedSignals State Transitions

## SsfStreamLifecycle

登録時に `enabled` として作成する。無効化すると `disabled` に遷移し、それ以降は配送も受信も行わない。再有効化すれば `enabled` に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作であり、付随する送信側設定と受信側設定をカスケード削除する。

| State | Kind | Meaning |
|---|---|---|
| enabled | initial | 配送と受信を行う |
| disabled | — | 配送も受信も行わない。再有効化できる |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | SsfStreamDisabled | — | disabled |  |
| disabled | SsfStreamEnabled | — | enabled |  |

## SecurityEventDeliveryLifecycle

生成時に `pending` として作成する。配送に成功すると終端状態の `delivered` へ、失敗すると `failed` へ遷移して再試行を予定する。`failed` から再試行すると `pending` に戻り、`max_delivery_attempts` を使い切ると終端状態の `dead_letter` へ遷移する。

| State | Kind | Meaning |
|---|---|---|
| pending | initial | 配送待ち。生成直後と再試行の予定後がこの状態である |
| delivered | terminal | 受信側へ配送できた |
| failed | — | 配送に失敗した。再試行を予定するか、上限に達すれば `dead_letter` へ進む |
| dead_letter | terminal | `max_delivery_attempts` を使い切った。以後は配送しない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | SecurityEventTransmitted | — | delivered |  |
| pending | SecurityEventDeliveryFailed | — | failed |  |
| failed | SecurityEventDeliveryRetried | — | pending |  |
| failed | SecurityEventDeliveryDeadLettered | — | dead_letter |  |
