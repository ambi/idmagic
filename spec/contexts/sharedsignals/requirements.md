# SharedSignals Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-SHAREDSIGNALS-001: kill-switchは既発行トークンをintrospectionで即時無効化する
- Actor: TenantAdministrator
- Given: Active な Agent "A1" に紐づく access token "AT1" が発行済みである
- Then: 管理者が KillAgent で "A1" を kill する
- Then: "A1" の AgentRevocationEpoch が現在時刻へ前進する
- Then: "RevocationEpochAdvanced" と "AgentAccessRevoked" が発行される
- Then: "AT1" の introspection は active=false を返す
- Alternative ("AT1" の issued_at が新しい epoch より後 (kill 後に再発行された token) である): introspection は active=true を返す (kill 前の token だけが対象)

### REQ-SHAREDSIGNALS-002: 所有者のオフボードは配下エージェント群を一括失効する
- Actor: TenantAdministrator
- Given: User "owner1" が Agent "A1"・"A2" を所有している
- Then: 管理者が "owner1" を DisableAdminUser する
- Then: "A1" と "A2" の AgentRevocationEpoch が同一エポックへ前進する
- Then: 両 Agent それぞれについて "AgentAccessRevoked" が発行される

### REQ-SHAREDSIGNALS-003: 署名不正のSETは反映されず拒否される
- Actor: System
- Given: direction=Receive の SsfStream "S1" が trusted_issuer "https://issuer.example" で登録されている
- Then: 署名が不正な SET を "S1" へ POST する
- Then: 要求は SecurityEventRejectedError で拒否される
- Then: "SecurityEventRejected" が verification_result=rejected_signature で発行される
- Then: AgentRevocationEpoch は変化しない

### REQ-SHAREDSIGNALS-004: 重複jtiのSETは一度だけ反映される
- Actor: System
- Given: direction=Receive の SsfStream "S1" に有効な SET (jti="J1") を送信済みである
- Then: 同一 jti "J1" の SET を再度 "S1" へ POST する
- Then: 要求は SecurityEventRejectedError で拒否される
- Then: "SecurityEventRejected" が verification_result=rejected_replay で発行される

### REQ-SHAREDSIGNALS-005: 他テナントのstreamはissuer一致でも受理されない
- Actor: System
- Given: tenant "T1" が direction=Receive の SsfStream "S1" を登録している
- Given: tenant "T2" の Agent を subject とする SET を保持している
- Then: SET を "S1" へ POST する
- Then: 要求は SecurityEventRejectedError で拒否される
- Then: "SecurityEventRejected" が verification_result=rejected_subject_unresolved で発行される

### REQ-SHAREDSIGNALS-006: 配送失敗は再試行され上限超過でdead_letterへ遷移する
- Actor: System
- Given: direction=Transmit の SsfStream "S1" に対する SecurityEventDelivery "D1" が pending である
- Given: "S1" の max_delivery_attempts は 3 である
- Then: receiver endpoint への配送が 3 回連続で失敗する
- Then: "D1" は 2 回まで failed→pending の再試行を経て、3 回目の失敗で dead_letter へ遷移する
- Then: 各失敗で "SecurityEventDeliveryFailed" が、最終失敗で "SecurityEventDeliveryDeadLettered" が発行される
- Alternative (3 回目までに配送が成功する): "D1" は delivered へ遷移し "SecurityEventTransmitted" が発行される

### REQ-SHAREDSIGNALS-007: receiver障害はローカル失効を遅らせない
- Actor: TenantAdministrator
- Given: direction=Transmit の SsfStream "S1" の receiver endpoint が到達不能である
- Then: 管理者が Agent "A1" を kill する
- Then: "A1" の AgentRevocationEpoch は即座に前進し introspection へ反映される
- Then: "S1" 向けの SecurityEventDelivery は pending のまま再試行対象になる (LocalRevocation は EcosystemPropagation の成否を待たない)

### REQ-SHAREDSIGNALS-008: 無効化したstreamは配送も受理も行わない
- Actor: TenantAdministrator
- Given: SsfStream "S1" が enabled である
- Then: 管理者が "S1" を DisableSsfStream する
- Then: "S1" が direction=Receive なら以後の SET は ssf_receiver_stream_enabled により拒否される
- Then: "S1" が direction=Transmit なら以後の RevocationEpochAdvanced から新規 SecurityEventDelivery を生成しない

### REQ-SHAREDSIGNALS-009: CheckRevocationEpoch
指定した Agent の revocation epoch を返す。OAuth2 の Introspect / 保護 API が access token の issued_at と比較し、epoch 以前に発行された token を fail-closed で無効と判定するために呼ぶ。

### REQ-SHAREDSIGNALS-010: AdvanceRevocationEpoch
KillAgent・DisableAgent・UnbindAgentCredential・所有者 (owner_user_id) の Disable/Delete・受理済み inbound SecurityEvent のいずれかを起点に、対象 Agent 群の revocation epoch を前進させる。所有者オフボードは対象テナント内で owner_user_id が一致する全 Agent を一括で前進させる。epoch は既存値より後退しない。
- Postcondition: epoch_advances_monotonically_for_all(input.agent_ids)

### REQ-SHAREDSIGNALS-011: ListSsfStreams
管理者が所属テナントの SsfStream を一覧する。

### REQ-SHAREDSIGNALS-012: GetSsfStream
管理者が個々の SsfStream を参照する。

### REQ-SHAREDSIGNALS-013: RegisterSsfTransmitterStream
管理者が direction=Transmit の SsfStream と配送先 (SsfTransmitterConfig) を登録する。

### REQ-SHAREDSIGNALS-014: RegisterSsfReceiverStream
管理者が direction=Receive の SsfStream と外部 transmitter の信頼設定 (SsfReceiverConfig) を登録する。

### REQ-SHAREDSIGNALS-015: UpdateSsfStream
管理者が SsfStream の event_types 等のメタデータを更新する。

### REQ-SHAREDSIGNALS-016: DisableSsfStream
管理者が SsfStream を無効化する。以後配送/受理を行わない。

### REQ-SHAREDSIGNALS-017: EnableSsfStream
管理者が無効化済みの SsfStream を再有効化する。

### REQ-SHAREDSIGNALS-018: DeleteSsfStream
管理者が SsfStream を削除する。付随する Transmitter/ReceiverConfig は cascade で削除する。

### REQ-SHAREDSIGNALS-019: ListSecurityEventDeliveries
管理者が direction=Transmit の SsfStream の配送状況 (pending/delivered/failed/dead_letter) を一覧する。運用の delivery health 確認に使う。

### REQ-SHAREDSIGNALS-020: ReceiveSecurityEvent
外部 transmitter からの SET を受理する。署名検証・登録済み issuer 照合・jti 重複検知・subject 解決のいずれかに失敗した場合は反映せず拒否する (fail-closed)。検証を通過したイベントはローカル revocation (AdvanceRevocationEpoch) として反映する。
- Precondition: ssf_receiver_stream_enabled(context.tenant_id, input.stream_id)
- Precondition: security_event_signature_and_claims_valid(context.tenant_id, input.stream_id, input.token)
- Precondition: !security_event_replayed(context.tenant_id, input.stream_id, input.token)
- Precondition: security_event_subject_resolves_to_tenant_local_principal(context.tenant_id, input.token)

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| SSF | OpenID Shared Signals Framework。Security Event Token (SET) を transport にセキュリティイベントを push/受信する標準。idmagic は transmitter と receiver の双方として振る舞う。 |  |
| CAEP | OpenID Continuous Access Evaluation Profile。SSF 上で session-revoked / token-claims-change / credential-change / assurance-level-change を表現するイベント種別の規約。 |  |
| RevocationEpoch | Agent ごとに保持する単調増加のタイムスタンプ。これより前に発行された access token / session は無効とみなす (fail-closed)。KillAgent・DisableAgent・UnbindAgentCredential・所有者 (owner_user_id) の Disable/Delete・受理した inbound SecurityEvent のいずれでも前進する。所有者オフボードは配下の全 Agent の epoch を一括で前進させる。 |  |
| LocalRevocation | idmagic 自身の Introspect / 保護 API が revocation epoch と access token の issued_at を比較して行う、当該 IdP 内で完結する失効判定。CAEP/SSF イベント配送より常に先行して確定させる。 |  |
| EcosystemPropagation | LocalRevocation で確定した失効を、SSF stream 経由で外部 receiver (別 IdP・resource server・ガバナンス基盤) へ CAEP イベントとして伝える層。receiver 障害・遅延は LocalRevocation を遅らせない。 |  |
| FailClosed | SET の署名検証失敗・鍵不明・改竄検知・issuer 未登録・重複 (replay) 検知・revocation epoch 判定不能のいずれかに該当する場合、常に「反映しない/無効とみなす」側へ倒す方針。 |  |
| System | SharedSignals の revocation 反映・SET 送受信 usecase そのものを指す、人間の操作者を伴わない技術的な主体。KillAgent 等のドメインイベントを起点に revocation epoch を前進させ、CAEP イベントを生成する。 |  |

## Standards

### Security Event Token (SET)

RFC 8417 — https://www.rfc-editor.org/rfc/rfc8417.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8417-SET-SIGNED | required | MUST | SET は jti/iat/iss/aud/events を含む署名済み JWT として発行する。 |
| RFC8417-SET-VERIFY | required | MUST | 受信側は署名検証を通過した SET のみ反映し、検証失敗・鍵不明・改竄検知は反映せず拒否・監査する (fail-closed)。 |

## State machines

### SsfStreamLifecycle

登録時に enabled として作成される。無効化で disabled に遷移し、以後配送/受理を行わない (fail-closed)。再有効化で enabled に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作で、付随する Transmitter/ReceiverConfig を cascade で削除する。

Initial: `enabled`  
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | SsfStreamDisabled | "" | disabled |  |
| disabled | SsfStreamEnabled | "" | enabled |  |

### SecurityEventDeliveryLifecycle

生成時に pending として作成される。配送成功で delivered (終端) へ、失敗で failed へ遷移し再試行をスケジュールする。failed からは再試行成功で pending に戻り、max_delivery_attempts を使い切ると dead_letter (終端) へ遷移する。

Initial: `pending`  
Terminal: `delivered`, `dead_letter`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | SecurityEventTransmitted | "" | delivered |  |
| pending | SecurityEventDeliveryFailed | "" | failed |  |
| failed | SecurityEventDeliveryRetried | "" | pending |  |
| failed | SecurityEventDeliveryDeadLettered | "" | dead_letter |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
