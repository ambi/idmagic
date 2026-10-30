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

## Rule: REQ-SHAREDSIGNALS-014 送信する SET は最上位に sub_id を持つ

### Example: EX-SHAREDSIGNALS-014-01 失効の SET

- Given Agent "A1" の失効を購読する送信側のストリームがある
- When `AgentAccessRevoked` を配送する
- Then SET の最上位の `sub_id` は `format` が `opaque`、`id` が "A1" である

## Rule: REQ-SHAREDSIGNALS-015 監査の記録と配送を同時に確定する

### Example: EX-SHAREDSIGNALS-015-01 追記が巻き戻ると配送も残らない

- Given `audit_event_types` が `*` の送信側のストリームがある
- When 監査の追記のトランザクションが確定せず巻き戻る
- Then 配送は残らない

### Example: EX-SHAREDSIGNALS-015-02 配送の結果のイベントは配送を生まない

- Given `audit_event_types` が `*` の送信側のストリームがある
- When "SecurityEventDeliveryFailed" の記録を追記する
- Then 配送は作られない

## Rule: REQ-SHAREDSIGNALS-016 監査の SET は許可した識別子だけを持つ

### Example: EX-SHAREDSIGNALS-016-01 個人情報を含めない

- Given 監査の記録の本文に `userId`、`targetUserId`、`username`、`email` がある
- When 配送を送る
- Then SET の本文には `userId` と `targetUserId` の識別子だけが含まれる

## Rule: REQ-SHAREDSIGNALS-017 配送先は公開の HTTPS に限り、接続のたびに検査する

### Example: EX-SHAREDSIGNALS-017-01 メタデータの宛先を登録できない

- When 配送先 "https://169.254.169.254/" で送信側のストリームを登録する
- Then 422 と `ssf_transmitter_delivery_endpoint_invalid` で拒否される

### Example: EX-SHAREDSIGNALS-017-02 登録後に非公開へ解決が変わった宛先へ接続しない

- Given 配送先が今は "10.0.0.1" へ解決する
- When 配送を試行する
- Then 接続せず、試行を失敗として記録する

## Rule: REQ-SHAREDSIGNALS-018 連続失敗でストリームを無効化し、配送を隔離する

### Example: EX-SHAREDSIGNALS-018-01 閾値で無効化する

- Given ストリームの連続の失敗が 19 回で、最初の失敗から 1 時間が過ぎている
- When 配送が失敗する
- Then ストリームは `disabled` になり、"SsfStreamAutoDisabled" が発行される
