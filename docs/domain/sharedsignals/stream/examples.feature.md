# Feature: SSF ストリームの管理の例

## Rule: REQ-SHAREDSIGNALS-008 無効化したストリームでは配送も受理も行わない

### Example: EX-SHAREDSIGNALS-008-01 通常経路

- Given SsfStream "S1" が `enabled` である
- When 管理者が "S1" を DisableSsfStream する
- Then "S1" が `direction=Receive` なら、以後の SET は `ssf_receiver_stream_enabled` によって拒否される
- Then "S1" が `direction=Transmit` なら、以後の RevocationEpochAdvanced から新しい SecurityEventDelivery を生成しない

## Rule: REQ-SHAREDSIGNALS-009 SsfStream の登録は Hard Quota を超えると拒否される

### Example: EX-SHAREDSIGNALS-009-01 通常経路

- Given 対象テナントの `ssf_streams` 上限が 20、利用量が 20 である
- When 管理者が RegisterSsfTransmitterStream で新しいストリームを登録しようとする
- Then QuotaExceededError で拒否され、SsfStream も付随する設定も作成されない
- Then "QuotaExceeded" が `resource="ssf_streams"` で発行される

### Example: EX-SHAREDSIGNALS-009-02 RegisterSsfReceiverStream で登録しようとする

- Given 対象テナントの `ssf_streams` 上限が 20、利用量が 20 である
- When 管理者が RegisterSsfTransmitterStream で新しいストリームを登録しようとする
- But RegisterSsfReceiverStream で登録しようとする
- Then 同じく QuotaExceededError で拒否される（送信側と受信側は同一の上限を共有する）

### Example: EX-SHAREDSIGNALS-009-03 管理者が先に既存のストリームを DeleteSsfStream する

- Given 対象テナントの `ssf_streams` 上限が 20、利用量が 20 である
- When 管理者が RegisterSsfTransmitterStream で新しいストリームを登録しようとする
- But 管理者が先に既存のストリームを DeleteSsfStream する
- Then 利用量が 19 に戻り、次の登録は成功する

## Rule: REQ-SHAREDSIGNALS-011 SsfStream の登録と状態変更は管理者に限られる

### Example: EX-SHAREDSIGNALS-011-01 通常経路

- Given "alice" は認証済みだが `admin` ロールを持たない
- When "alice" が送信側または受信側の SsfStream を登録する
- Then AccessDeniedError で拒否される
- Then SsfStream は作成されず、既存のストリームの状態も変わらない

### Example: EX-SHAREDSIGNALS-011-02 "alice" がストリームの無効化、有効化、更新、削除を要求する

- Given "alice" は認証済みだが `admin` ロールを持たない
- When "alice" が送信側または受信側の SsfStream を登録する
- But "alice" がストリームの無効化、有効化、更新、削除を要求する
- Then AccessDeniedError で拒否される
