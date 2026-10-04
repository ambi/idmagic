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

## 操作

### worker による配送

#### REQ-SHAREDSIGNALS-006 配送失敗は再試行し、上限を超えると dead_letter へ遷移する
