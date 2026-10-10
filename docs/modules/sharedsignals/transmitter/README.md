# SET の配送

## 概要

この文書は、確定した失効を、送信側のストリームを通じて外部の受信側へ CAEP のイベントとして配送する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `AgentAccessRevoked` からの配送の作成、配送の試行、再試行、配信不能への遷移 |
| 行為者 | System（失効への反応と、`worker` の配送のループ） |
| 扱わないもの | ローカルの失効の確定は[失効エポック](../revocation/README.md)が扱う |

## 状態遷移

### SecurityEventDeliveryLifecycle

生成時に `pending` として作成する。配送の試行に成功すると終端状態の `delivered` へ、失敗すると `failed` へ遷移して再試行を予定する。再試行の時刻が来ると `pending` に戻って試行し、`max_delivery_attempts` 回目の試行に失敗すると終端状態の `dead_letter` へ遷移する。

| State | Kind | Meaning |
|---|---|---|
| pending | initial | 配送待ち。生成直後と再試行の時刻の到来後がこの状態である |
| delivered | terminal | 受信側へ配送できた |
| failed | — | 配送に失敗し、再試行を予定している |
| dead_letter | terminal | `max_delivery_attempts` を使い切った。以後は配送しない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | SecurityEventTransmitted | — | delivered |  |
| pending | SecurityEventDeliveryFailed | `attempt_count < max_delivery_attempts` | failed |  |
| pending | SecurityEventDeliveryDeadLettered | `attempt_count = max_delivery_attempts` | dead_letter |  |
| failed | SecurityEventDeliveryRetried | — | pending |  |

| State | 配送の試行 | 再試行の時刻の到来 |
|---|---|---|
| pending | → delivered（受信側が受理）<br>→ failed（失敗し、試行の回数が上限未満）<br>→ dead_letter（失敗し、試行の回数が上限） | 何もしない |
| delivered | 何もしない | 何もしない |
| failed | 何もしない | → pending |
| dead_letter | 何もしない | 何もしない |

## 操作

### worker による配送

#### REQ-SHAREDSIGNALS-006 配送失敗は再試行し、上限を超えると dead_letter へ遷移する

- `AgentAccessRevoked` を発行したとき、SharedSignals は、`enabled` で `session-revoked` を購読する送信側のストリームごとに、`pending` の配送を作る。
- `worker` が配送の時刻が来た `pending` の配送を試行し、受信側が受理したとき、SharedSignals は、配送を `delivered` にし、`SecurityEventTransmitted` を発行する。
- `worker` が配送を試行して失敗し、試行の回数が `max_delivery_attempts`（デフォルト 8 回）未満の場合、SharedSignals は、配送を `failed` にし、30 秒から倍々に増えて 30 分を上限とする待ち時間の後に次の試行を予定し、`SecurityEventDeliveryFailed` を発行する。
- `worker` が配送を試行して失敗し、試行の回数が `max_delivery_attempts` に達した場合、SharedSignals は、配送を `dead_letter` にし、`SecurityEventDeliveryFailed` と `SecurityEventDeliveryDeadLettered` を発行し、以後は配送しない。
- `failed` の配送の再試行の時刻が来たとき、SharedSignals は、配送を `pending` に戻し、`SecurityEventDeliveryRetried` を発行してから試行する。
- 送信側の設定を削除したストリームの配送を試行する場合、SharedSignals は、配送を失敗として扱う。
- **例**：EX-SHAREDSIGNALS-006-01、EX-SHAREDSIGNALS-006-02
