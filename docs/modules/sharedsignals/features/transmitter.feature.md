# Feature: SET の配送の例

## Rule: REQ-SHAREDSIGNALS-006 配送失敗は再試行し、上限を超えると dead_letter へ遷移する

### Example: EX-SHAREDSIGNALS-006-01 通常経路

- Given `direction=Transmit` の SsfStream "S1" に対する SecurityEventDelivery "D1" が `pending` である
- And "S1" の `max_delivery_attempts` は 3 である
- When 受信側エンドポイントへの配送が 3 回連続で失敗する
- Then "D1" は 2 回まで `failed` から `pending` へ戻って再試行し、3 回目の失敗で `dead_letter` へ遷移する
- Then 各失敗で "SecurityEventDeliveryFailed" が、最終失敗で "SecurityEventDeliveryDeadLettered" が発行される

### Example: EX-SHAREDSIGNALS-006-02 3 回目までに配送が成功する

- Given `direction=Transmit` の SsfStream "S1" に対する SecurityEventDelivery "D1" が `pending` である
- And "S1" の `max_delivery_attempts` は 3 である
- When 受信側エンドポイントへの配送が 3 回連続で失敗する
- But 3 回目までに配送が成功する
- Then "D1" は `delivered` へ遷移し、"SecurityEventTransmitted" が発行される

## Rule: REQ-SHAREDSIGNALS-013 管理者は配信不能の配送をやり直せる

### Example: EX-SHAREDSIGNALS-013-01 受信側の復旧の後にやり直す

- Given `direction=Transmit` の SsfStream "S1" に対する SecurityEventDelivery "D1" が `dead_letter` である
- And 受信側エンドポイントが復旧している
- When 管理者が "S1" の配送のやり直しを要求する
- Then 応答は 200 で `redelivered_count` は 1 であり、"SecurityEventDeliveryRedeliveryRequested" が発行される
- And "D1" は `pending` に戻り、試行の回数は 0 である
- When `worker` が配送を試行する
- Then 受信側は最初と同じ SET を受け取り、"D1" は `delivered` へ遷移する

### Example: EX-SHAREDSIGNALS-013-02 無効化したストリームではやり直せない

- Given `direction=Transmit` の SsfStream "S1" が `disabled` である
- And "S1" に対する SecurityEventDelivery "D1" が `dead_letter` である
- When 管理者が "S1" の配送のやり直しを要求する
- Then 応答は 409 `ssf_stream_disabled` であり、"D1" は `dead_letter` のままである
