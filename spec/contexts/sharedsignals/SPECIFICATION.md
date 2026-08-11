---
context: sharedsignals
updated_at: 2026-08-11
---

# SharedSignals Specification

## Overview

Shared Signals Framework (SSF) / Continuous Access Evaluation Profile (CAEP) による
エージェントの近リアルタイム失効を所有する。Agent ごとの revocation epoch を保持し、
KillAgent・DisableAgent・UnbindAgentCredential・所有者オフボード・検証済み inbound Security Event
Token (SET, RFC 8417) のいずれかを起点に fail-closed で前進させる (LocalRevocation)。前進した epoch は
OAuth2 の Introspect が access token の issued_at と比較し即時失効へ反映する。同時に SSF stream 経由で
CAEP イベントを外部 receiver へ push し (EcosystemPropagation)、外部 transmitter からの検証済みイベントも
受理してローカル失効へ収束させる。idmagic は transmitter と receiver の双方として振る舞う。

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

## State Transitions

### SsfStreamLifecycle

登録時に enabled として作成される。無効化で disabled に遷移し、以後配送/受理を行わない (fail-closed)。再有効化で enabled に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作で、付随する Transmitter/ReceiverConfig を cascade で削除する。

Initial: `enabled`
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | SsfStreamDisabled | — | disabled |  |
| disabled | SsfStreamEnabled | — | enabled |  |

### SecurityEventDeliveryLifecycle

生成時に pending として作成される。配送成功で delivered (終端) へ、失敗で failed へ遷移し再試行をスケジュールする。failed からは再試行成功で pending に戻り、max_delivery_attempts を使い切ると dead_letter (終端) へ遷移する。

Initial: `pending`
Terminal: `delivered`, `dead_letter`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | SecurityEventTransmitted | — | delivered |  |
| pending | SecurityEventDeliveryFailed | — | failed |  |
| failed | SecurityEventDeliveryRetried | — | pending |  |
| failed | SecurityEventDeliveryDeadLettered | — | dead_letter |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Internal Interfaces

#### CheckRevocationEpoch
指定した Agent の revocation epoch を返す。OAuth2 の Introspect / 保護 API が access token の issued_at と比較し、epoch 以前に発行された token を fail-closed で無効と判定するために呼ぶ。

#### AdvanceRevocationEpoch
KillAgent・DisableAgent・UnbindAgentCredential・所有者 (owner_user_id) の Disable/Delete・受理済み inbound SecurityEvent のいずれかを起点に、対象 Agent 群の revocation epoch を前進させる。所有者オフボードは対象テナント内で owner_user_id が一致する全 Agent を一括で前進させる。epoch は既存値より後退しない。
- Result invariant: epoch_advances_monotonically_for_all(input.agent_ids)

## Scenarios

### REQ-SHAREDSIGNALS-001: kill-switchは既発行トークンをintrospectionで即時無効化する
- ACTOR TenantAdministrator
- GIVEN Active な Agent "A1" に紐づく access token "AT1" が発行済みである
- WHEN 管理者が KillAgent で "A1" を kill する
- THEN "A1" の AgentRevocationEpoch が現在時刻へ前進する
- THEN "RevocationEpochAdvanced" と "AgentAccessRevoked" が発行される
- THEN "AT1" の introspection は active=false を返す
  - ALT token の issued_at が新しい epoch より後 (kill 後に再発行された token) である → introspection は active=true を返す (kill 前の token だけが対象)

### REQ-SHAREDSIGNALS-002: 所有者のオフボードは配下エージェント群を一括失効する
- ACTOR TenantAdministrator
- GIVEN User "owner1" が Agent "A1"・"A2" を所有している
- WHEN 管理者が "owner1" を DisableAdminUser する
- THEN "A1" と "A2" の AgentRevocationEpoch が同一エポックへ前進する
- THEN 両 Agent それぞれについて "AgentAccessRevoked" が発行される

### REQ-SHAREDSIGNALS-003: 署名不正のSETは反映されず拒否される
- ACTOR System
- GIVEN direction=Receive の SsfStream "S1" が trusted_issuer "https://issuer.example" で登録されている
- WHEN 署名が不正な SET を "S1" へ POST する
- THEN 要求は SecurityEventRejectedError で拒否される
- THEN "SecurityEventRejected" が verification_result=rejected_signature で発行される
- THEN AgentRevocationEpoch は変化しない

### REQ-SHAREDSIGNALS-004: 重複jtiのSETは一度だけ反映される
- ACTOR System
- GIVEN direction=Receive の SsfStream "S1" に有効な SET (jti="J1") を送信済みである
- WHEN 同一 jti "J1" の SET を再度 "S1" へ POST する
- THEN 要求は SecurityEventRejectedError で拒否される
- THEN "SecurityEventRejected" が verification_result=rejected_replay で発行される

### REQ-SHAREDSIGNALS-005: 他テナントのstreamはissuer一致でも受理されない
- ACTOR System
- GIVEN tenant "T1" が direction=Receive の SsfStream "S1" を登録している
- GIVEN tenant "T2" の Agent を subject とする SET を保持している
- WHEN SET を "S1" へ POST する
- THEN 要求は SecurityEventRejectedError で拒否される
- THEN "SecurityEventRejected" が verification_result=rejected_subject_unresolved で発行される

### REQ-SHAREDSIGNALS-006: 配送失敗は再試行され上限超過でdead_letterへ遷移する
- ACTOR System
- GIVEN direction=Transmit の SsfStream "S1" に対する SecurityEventDelivery "D1" が pending である
- GIVEN "S1" の max_delivery_attempts は 3 である
- WHEN receiver endpoint への配送が 3 回連続で失敗する
  - ALT 3 回目までに配送が成功する → "D1" は delivered へ遷移し "SecurityEventTransmitted" が発行される
- THEN "D1" は 2 回まで failed→pending の再試行を経て、3 回目の失敗で dead_letter へ遷移する
- THEN 各失敗で "SecurityEventDeliveryFailed" が、最終失敗で "SecurityEventDeliveryDeadLettered" が発行される

### REQ-SHAREDSIGNALS-007: receiver障害はローカル失効を遅らせない
- ACTOR TenantAdministrator
- GIVEN direction=Transmit の SsfStream "S1" の receiver endpoint が到達不能である
- WHEN 管理者が Agent "A1" を kill する
- THEN "A1" の AgentRevocationEpoch は即座に前進し introspection へ反映される
- THEN "S1" 向けの SecurityEventDelivery は pending のまま再試行対象になる (LocalRevocation は EcosystemPropagation の成否を待たない)

### REQ-SHAREDSIGNALS-008: 無効化したstreamは配送も受理も行わない
- ACTOR TenantAdministrator
- GIVEN SsfStream "S1" が enabled である
- WHEN 管理者が "S1" を DisableSsfStream する
- THEN "S1" が direction=Receive なら以後の SET は ssf_receiver_stream_enabled により拒否される
- THEN "S1" が direction=Transmit なら以後の RevocationEpochAdvanced から新規 SecurityEventDelivery を生成しない
