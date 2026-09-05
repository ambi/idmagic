# Feature: SharedSignals Scenarios

## Rule: REQ-SHAREDSIGNALS-001 キルスイッチは発行済みトークンをイントロスペクションで即時無効化する

Primary actor: `TenantAdministrator`

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

## Rule: REQ-SHAREDSIGNALS-002 所有者のオフボードは配下エージェント群を一括失効する (superseded by REQ-PLATFORM-001)

引き金の `DisableAdminUser` は IdManagement の操作である。所有者の無効化がログイン、既存セッション、配下エージェントのトークンを同時に閉じることを、REQ-PLATFORM-001 が 1 つの保証として述べる。

## Rule: REQ-SHAREDSIGNALS-003 署名が不正な SET は反映せずに拒否する

Primary actor: `System`

### Example: EX-SHAREDSIGNALS-003-01 通常経路

- Given `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- When 署名が不正な SET を "S1" へ POST する
- Then リクエストは `SecurityEventRejectedError` で拒否される
- Then "SecurityEventRejected" が `verification_result=rejected_signature` で発行される
- Then AgentRevocationEpoch は変化しない

## Rule: REQ-SHAREDSIGNALS-004 同じ jti の SET は一度だけ反映する

Primary actor: `System`

### Example: EX-SHAREDSIGNALS-004-01 通常経路

- Given `direction=Receive` の SsfStream "S1" に有効な SET（`jti="J1"`）を送信済みである
- When 同一 jti "J1" の SET を再度 "S1" へ POST する
- Then リクエストは `SecurityEventRejectedError` で拒否される
- Then "SecurityEventRejected" が `verification_result=rejected_replay` で発行される

## Rule: REQ-SHAREDSIGNALS-005 発行者が一致しても他テナントのストリームでは受理しない

Primary actor: `System`

### Example: EX-SHAREDSIGNALS-005-01 通常経路

- Given テナント "T1" が `direction=Receive` の SsfStream "S1" を登録している
- And テナント "T2" の Agent を subject とする SET がある
- When SET を "S1" へ POST する
- Then リクエストは `SecurityEventRejectedError` で拒否される
- Then "SecurityEventRejected" が `verification_result=rejected_subject_unresolved` で発行される

## Rule: REQ-SHAREDSIGNALS-006 配送失敗は再試行し、上限を超えると dead_letter へ遷移する

Primary actor: `System`

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

## Rule: REQ-SHAREDSIGNALS-007 受信側の障害はローカル失効を遅らせない

Primary actor: `TenantAdministrator`

### Example: EX-SHAREDSIGNALS-007-01 通常経路

- Given `direction=Transmit` の SsfStream "S1" の受信側エンドポイントに到達できない
- When 管理者が Agent "A1" を強制終了する
- Then "A1" の AgentRevocationEpoch は即座に進み、イントロスペクションに反映される
- Then "S1" 向けの SecurityEventDelivery は `pending` のまま再試行対象になる（LocalRevocation は EcosystemPropagation の成否を待たない）

## Rule: REQ-SHAREDSIGNALS-008 無効化したストリームでは配送も受理も行わない

Primary actor: `TenantAdministrator`

### Example: EX-SHAREDSIGNALS-008-01 通常経路

- Given SsfStream "S1" が `enabled` である
- When 管理者が "S1" を DisableSsfStream する
- Then "S1" が `direction=Receive` なら、以後の SET は `ssf_receiver_stream_enabled` によって拒否される
- Then "S1" が `direction=Transmit` なら、以後の RevocationEpochAdvanced から新しい SecurityEventDelivery を生成しない

## Rule: REQ-SHAREDSIGNALS-009 SsfStream の登録は Hard Quota を超えると拒否される

Primary actor: `TenantAdministrator`

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

## Rule: REQ-SHAREDSIGNALS-010 RFC 9493 の Subject Identifier で送られた SET も主体を解決する

Primary actor: `System`

### Example: EX-SHAREDSIGNALS-010-01 通常経路

- Given `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- And テナント "T1" に Agent "A1" が存在し、OAuth2Client "C1" に束縛されている
- When `format=iss_sub`、`iss="https://issuer.example"`、`sub="A1"` の Subject Identifier を持つ SET を "S1" へ POST する
- Then "A1" の AgentRevocationEpoch が前進する
- Then "SecurityEventReceived" が発行される

### Example: EX-SHAREDSIGNALS-010-02 `sub` が "A1" ではなく束縛先の "C1" である

- Given `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- And テナント "T1" に Agent "A1" が存在し、OAuth2Client "C1" に束縛されている
- When `format=iss_sub`、`iss="https://issuer.example"`、`sub="A1"` の Subject Identifier を持つ SET を "S1" へ POST する
- But `sub` が "A1" ではなく束縛先の "C1" である
- Then 同じく "A1" として解決され、失効エポックが前進する

### Example: EX-SHAREDSIGNALS-010-03 `format=opaque`、`id="A1"` である

- Given `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- And テナント "T1" に Agent "A1" が存在し、OAuth2Client "C1" に束縛されている
- When `format=iss_sub`、`iss="https://issuer.example"`、`sub="A1"` の Subject Identifier を持つ SET を "S1" へ POST する
- But `format=opaque`、`id="A1"` である
- Then 同じく "A1" として解決される（テナントは受信ストリームが属するテナントで決まる）

### Example: EX-SHAREDSIGNALS-010-04 `iss` が "S1" の `trusted_issuer` と一致しない

- Given `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- And テナント "T1" に Agent "A1" が存在し、OAuth2Client "C1" に束縛されている
- When `format=iss_sub`、`iss="https://issuer.example"`、`sub="A1"` の Subject Identifier を持つ SET を "S1" へ POST する
- But `iss` が "S1" の `trusted_issuer` と一致しない
- Then `SecurityEventRejectedError` で拒否され、"SecurityEventRejected" が `verification_result=rejected_subject_unresolved` で発行される

### Example: EX-SHAREDSIGNALS-010-05 `format=email` など IdMagic が解釈しない形式である

- Given `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- And テナント "T1" に Agent "A1" が存在し、OAuth2Client "C1" に束縛されている
- When `format=iss_sub`、`iss="https://issuer.example"`、`sub="A1"` の Subject Identifier を持つ SET を "S1" へ POST する
- But `format=email` など IdMagic が解釈しない形式である
- Then `SecurityEventRejectedError` で拒否される

### Example: EX-SHAREDSIGNALS-010-06 `sub` がどの Agent にも束縛先クライアントにも一致しない

- Given `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- And テナント "T1" に Agent "A1" が存在し、OAuth2Client "C1" に束縛されている
- When `format=iss_sub`、`iss="https://issuer.example"`、`sub="A1"` の Subject Identifier を持つ SET を "S1" へ POST する
- But `sub` がどの Agent にも束縛先クライアントにも一致しない
- Then `SecurityEventRejectedError` で拒否される

## Rule: REQ-SHAREDSIGNALS-011 SsfStream の登録と状態変更は管理者に限られる

Primary actor: `TenantAdministrator`

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
