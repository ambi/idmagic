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
