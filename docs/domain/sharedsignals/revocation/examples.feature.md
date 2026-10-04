# Feature: 失効エポックの例

## Rule: REQ-SHAREDSIGNALS-001 キルスイッチは発行済みトークンをイントロスペクションで即時無効化する

### Example: EX-SHAREDSIGNALS-001-01 通常経路

- Given `Active` の Agent "A1" に紐づくアクセストークン "AT1" が発行済みである
- When 管理者が KillAgent で "A1" を強制終了する
- Then "A1" の AgentRevocationEpoch が現在時刻へ前進する
- Then "RevocationEpochAdvanced" と "AgentAccessRevoked" が発行される
- Then "AT1" のイントロスペクションは `active=false` を返す

### Example: EX-SHAREDSIGNALS-001-02 トークンの `issued_at` が新しい失効エポックより後である

- Given `Active` の Agent "A1" に紐づくアクセストークン "AT1" が発行済みである
- When 管理者が KillAgent で "A1" を強制終了する
- Then "A1" の AgentRevocationEpoch が現在時刻へ前進する
- Then "RevocationEpochAdvanced" と "AgentAccessRevoked" が発行される
- Then トークンの `issued_at` が新しい失効エポックより後である
- Then イントロスペクションは `active=true` を返す（強制終了後に再発行されたトークンは失効対象にしない）

## Rule: REQ-SHAREDSIGNALS-007 受信側の障害はローカル失効を遅らせない

### Example: EX-SHAREDSIGNALS-007-01 通常経路

- Given `direction=Transmit` の SsfStream "S1" の受信側エンドポイントに到達できない
- When 管理者が Agent "A1" を強制終了する
- Then "A1" の AgentRevocationEpoch は即座に進み、イントロスペクションに反映される
- Then "S1" 向けの SecurityEventDelivery は `pending` のまま再試行対象になる（LocalRevocation は EcosystemPropagation の成否を待たない）
