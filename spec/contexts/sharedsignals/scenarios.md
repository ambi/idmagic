# SharedSignals Scenarios

### REQ-SHAREDSIGNALS-001: キルスイッチは発行済みトークンをイントロスペクションで即時無効化する
- ACTOR TenantAdministrator
- GIVEN `Active` の Agent "A1" に紐づくアクセストークン "AT1" が発行済みである
- WHEN 管理者が KillAgent で "A1" を強制終了する
- THEN "A1" の AgentRevocationEpoch が現在時刻へ前進する
- THEN "RevocationEpochAdvanced" と "AgentAccessRevoked" が発行される
- THEN "AT1" のイントロスペクションは `active=false` を返す
  - ALT トークンの `issued_at` が新しい失効エポックより後である → イントロスペクションは `active=true` を返す（強制終了後に再発行されたトークンは失効対象にしない）

### REQ-SHAREDSIGNALS-002: 所有者のオフボードは配下エージェント群を一括失効する
- ACTOR TenantAdministrator
- GIVEN User "owner1" が Agent "A1"・"A2" を所有している
- WHEN 管理者が "owner1" を DisableAdminUser する
- THEN "A1" と "A2" の AgentRevocationEpoch が同一エポックへ前進する
- THEN 両 Agent それぞれについて "AgentAccessRevoked" が発行される

### REQ-SHAREDSIGNALS-003: 署名が不正な SET は反映せずに拒否する
- ACTOR System
- GIVEN `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- WHEN 署名が不正な SET を "S1" へ POST する
- THEN リクエストは `SecurityEventRejectedError` で拒否される
- THEN "SecurityEventRejected" が `verification_result=rejected_signature` で発行される
- THEN AgentRevocationEpoch は変化しない

### REQ-SHAREDSIGNALS-004: 同じ jti の SET は一度だけ反映する
- ACTOR System
- GIVEN `direction=Receive` の SsfStream "S1" に有効な SET（`jti="J1"`）を送信済みである
- WHEN 同一 jti "J1" の SET を再度 "S1" へ POST する
- THEN リクエストは `SecurityEventRejectedError` で拒否される
- THEN "SecurityEventRejected" が `verification_result=rejected_replay` で発行される

### REQ-SHAREDSIGNALS-005: 発行者が一致しても他テナントのストリームでは受理しない
- ACTOR System
- GIVEN テナント "T1" が `direction=Receive` の SsfStream "S1" を登録している
- GIVEN テナント "T2" の Agent を subject とする SET がある
- WHEN SET を "S1" へ POST する
- THEN リクエストは `SecurityEventRejectedError` で拒否される
- THEN "SecurityEventRejected" が `verification_result=rejected_subject_unresolved` で発行される

### REQ-SHAREDSIGNALS-006: 配送失敗は再試行し、上限を超えると dead_letter へ遷移する
- ACTOR System
- GIVEN `direction=Transmit` の SsfStream "S1" に対する SecurityEventDelivery "D1" が `pending` である
- GIVEN "S1" の `max_delivery_attempts` は 3 である
- WHEN 受信側エンドポイントへの配送が 3 回連続で失敗する
  - ALT 3 回目までに配送が成功する → "D1" は `delivered` へ遷移し、"SecurityEventTransmitted" が発行される
- THEN "D1" は 2 回まで `failed` から `pending` へ戻って再試行し、3 回目の失敗で `dead_letter` へ遷移する
- THEN 各失敗で "SecurityEventDeliveryFailed" が、最終失敗で "SecurityEventDeliveryDeadLettered" が発行される

### REQ-SHAREDSIGNALS-007: 受信側の障害はローカル失効を遅らせない
- ACTOR TenantAdministrator
- GIVEN `direction=Transmit` の SsfStream "S1" の受信側エンドポイントに到達できない
- WHEN 管理者が Agent "A1" を強制終了する
- THEN "A1" の AgentRevocationEpoch は即座に進み、イントロスペクションに反映される
- THEN "S1" 向けの SecurityEventDelivery は `pending` のまま再試行対象になる（LocalRevocation は EcosystemPropagation の成否を待たない）

### REQ-SHAREDSIGNALS-008: 無効化したストリームでは配送も受理も行わない
- ACTOR TenantAdministrator
- GIVEN SsfStream "S1" が `enabled` である
- WHEN 管理者が "S1" を DisableSsfStream する
- THEN "S1" が `direction=Receive` なら、以後の SET は `ssf_receiver_stream_enabled` によって拒否される
- THEN "S1" が `direction=Transmit` なら、以後の RevocationEpochAdvanced から新しい SecurityEventDelivery を生成しない

### REQ-SHAREDSIGNALS-009: SsfStream の登録は Hard Quota を超えると拒否される
- ACTOR TenantAdministrator
- GIVEN 対象テナントの `ssf_streams` 上限が 20、利用量が 20 である
- WHEN 管理者が RegisterSsfTransmitterStream で新しいストリームを登録しようとする
  - ALT RegisterSsfReceiverStream で登録しようとする → 同じく QuotaExceededError で拒否される（送信側と受信側は同一の上限を共有する）
  - ALT 管理者が先に既存のストリームを DeleteSsfStream する → 利用量が 19 に戻り、次の登録は成功する
- THEN QuotaExceededError で拒否され、SsfStream も付随する設定も作成されない
- THEN "QuotaExceeded" が `resource="ssf_streams"` で発行される

### REQ-SHAREDSIGNALS-010: RFC 9493 の Subject Identifier で送られた SET も主体を解決する
- ACTOR System
- GIVEN `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- GIVEN テナント "T1" に Agent "A1" が存在し、OAuth2Client "C1" に束縛されている
- WHEN `format=iss_sub`、`iss="https://issuer.example"`、`sub="A1"` の Subject Identifier を持つ SET を "S1" へ POST する
  - ALT `sub` が "A1" ではなく束縛先の "C1" である → 同じく "A1" として解決され、失効エポックが前進する
  - ALT `format=opaque`、`id="A1"` である → 同じく "A1" として解決される（テナントは受信ストリームが属するテナントで決まる）
  - ALT `iss` が "S1" の `trusted_issuer` と一致しない → `SecurityEventRejectedError` で拒否され、"SecurityEventRejected" が `verification_result=rejected_subject_unresolved` で発行される
  - ALT `format=email` など IdMagic が解釈しない形式である → `SecurityEventRejectedError` で拒否される
  - ALT `sub` がどの Agent にも束縛先クライアントにも一致しない → `SecurityEventRejectedError` で拒否される
- THEN "A1" の AgentRevocationEpoch が前進する
- THEN "SecurityEventReceived" が発行される

### REQ-SHAREDSIGNALS-011: SsfStream の登録と状態変更は管理者に限られる
- ACTOR TenantAdministrator
- GIVEN "alice" は認証済みだが `admin` ロールを持たない
- WHEN "alice" が送信側または受信側の SsfStream を登録する
  - ALT "alice" がストリームの無効化、有効化、更新、削除を要求する → AccessDeniedError で拒否される
- THEN AccessDeniedError で拒否される
- THEN SsfStream は作成されず、既存のストリームの状態も変わらない
