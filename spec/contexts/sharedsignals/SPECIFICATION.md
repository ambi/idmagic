---
context: sharedsignals
updated_at: 2026-08-15
---

# SharedSignals Specification

## Overview

Shared Signals Framework (SSF) と Continuous Access Evaluation Profile (CAEP) による、エージェントのほぼリアルタイムな失効を所有する。IdMagic は SSF の送信側と受信側の両方として振る舞う。

中心となるのは `Agent` ごとの失効エポックである。`KillAgent`、`DisableAgent`、`UnbindAgentCredential`、所有者のオフボーディング、検証済みの受信 Security Event Token (SET、RFC 8417) のいずれかを契機に単調に前進する。OAuth2 の `Introspect` はこの値をアクセストークンの `issued_at` と比較し、即時失効へ反映する (`LocalRevocation`)。

確定した失効は、SSF ストリームを通じて CAEP イベントとして外部の受信側へ伝える (`EcosystemPropagation`)。伝播はローカル失効の後に行うため、受信側の障害や遅延がローカル失効を妨げることはない。外部の送信側から受け取った検証済みイベントも、同じ失効エポックへ反映する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| SSF | OpenID Shared Signals Framework。Security Event Token (SET) を搬送形式として、セキュリティイベントをプッシュまたは受信する標準。IdMagic は送信側と受信側の両方として振る舞う。 |  |
| CAEP | OpenID Continuous Access Evaluation Profile。SSF 上で `session-revoked`、`token-claims-change`、`credential-change`、`assurance-level-change` を表現するイベント種別の規約。 |  |
| RevocationEpoch | `Agent` ごとに保持する単調増加のタイムスタンプ。これより前に発行されたアクセストークンやセッションは、フェイルクローズで無効とみなす。`KillAgent`、`DisableAgent`、`UnbindAgentCredential`、所有者 (`owner_user_id`) の無効化または削除、受信した `SecurityEvent` のいずれでも前進する。所有者のオフボーディングでは、配下の全 `Agent` のエポックを一括して前進させる。 |  |
| LocalRevocation | IdMagic 自身の `Introspect` または保護 API が、失効エポックとアクセストークンの `issued_at` を比較して行う、当該 IdP 内で完結する失効判定。CAEP / SSF イベントの配送より常に先に確定する。 |  |
| EcosystemPropagation | `LocalRevocation` で確定した失効を、SSF ストリームを通じて外部の受信側 (別の IdP、リソースサーバー、ガバナンス基盤) へ CAEP イベントとして伝える層。受信側の障害や遅延によって `LocalRevocation` を遅らせない。 |  |
| FailClosed | SET の署名検証失敗、未知の鍵、改ざんの検知、未登録の発行者、重複の検知、失効エポックを判定できない場合のいずれでも、常に「反映しない」または「無効とみなす」側に倒す方針。 |  |
| SubjectIdentifier | SET が指す主体の表現。RFC 9493 は `format` メンバーで種別を区別する標準形式を定める。IdMagic は自身の送信側が使う独自形式に加えて、外部の送信側との相互運用のために RFC 9493 の `iss_sub` と `opaque` を解釈する。 | Subject Identifier, サブジェクト識別子 |
| System | `SharedSignals` の失効反映と SET 送受信のユースケースそのものを指す、人間の操作者を伴わない技術的な主体。`KillAgent` などの Domain Event を契機に失効エポックを進め、CAEP イベントを生成する。 |  |

## Standards

### Security Event Token (SET)

RFC 8417 — https://www.rfc-editor.org/rfc/rfc8417.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8417-SET-SIGNED | required | MUST | SET は jti/iat/iss/aud/events を含む署名済み JWT として発行する。 |
| RFC8417-SET-VERIFY | required | MUST | 受信側は署名検証を通過した SET だけを反映する。検証失敗、未知の鍵、改ざんを検知した場合は反映せず、フェイルクローズで拒否して監査する。 |

### Subject Identifiers for Security Event Tokens

RFC 9493 — https://www.rfc-editor.org/rfc/rfc9493.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9493-SUBID-FORMAT | partial | MUST | `format` メンバーを持つ Subject Identifier を受け取った場合、`format` の値でのみ種別を判定する。IdMagic が解釈するのは `iss_sub` と `opaque` であり、それ以外の `format` は主体を解決できないものとしてフェイルクローズで拒否する。 |
| RFC9493-SUBID-ISS-SUB | partial | MUST | `format=iss_sub` の `iss` は、受信ストリームに登録された `trusted_issuer` と完全一致しなければならない。一致しない `iss` は、SET の署名が正しくても主体を解決できないものとして拒否する。 |

## State Transitions

### SsfStreamLifecycle

登録時に `enabled` として作成する。無効化すると `disabled` に遷移し、それ以降は配送も受信も行わない。再有効化すれば `enabled` に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作であり、付随する送信側設定と受信側設定をカスケード削除する。

Initial: `enabled` Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | SsfStreamDisabled | — | disabled |  |
| disabled | SsfStreamEnabled | — | enabled |  |

### SecurityEventDeliveryLifecycle

生成時に `pending` として作成する。配送に成功すると終端状態の `delivered` へ、失敗すると `failed` へ遷移して再試行を予定する。`failed` から再試行すると `pending` に戻り、`max_delivery_attempts` を使い切ると終端状態の `dead_letter` へ遷移する。

Initial: `pending` Terminal: `delivered`, `dead_letter`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| pending | SecurityEventTransmitted | — | delivered |  |
| pending | SecurityEventDeliveryFailed | — | failed |  |
| failed | SecurityEventDeliveryRetried | — | pending |  |
| failed | SecurityEventDeliveryDeadLettered | — | dead_letter |  |

## Authorization Boundary

`SsfStream` の登録、更新、有効化、無効化、削除、および配送状況の参照は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが所属テナントに対して行える。API アクセストークンでは、ロールに加えて `shared-signals:read` がストリームと配送状況の参照だけを、`shared-signals:write` がストリームの登録と変更を許可する。

受信エンドポイント (`/ssf/streams/{stream_id}/events`) はブラウザーセッションを持たない外部の送信側が呼ぶため、管理 API の認証経路には載せない。代わりに、SET の署名が受信ストリームに登録済みの `trusted_issuer` の鍵で検証できること、`jti` が未使用であること、主体がそのストリームのテナント内で解決できることをすべて満たしたときにだけ受理する。1 つでも満たさなければ失効を反映せず拒否する。受理した SET が変更できるのは対象 Agent の失効エポックだけであり、これを進める以外の副作用は持たない。

失効エポックの前進 (`AdvanceRevocationEpoch`) と参照 (`CheckRevocationEpoch`) は HTTP に公開せず、Domain Event と OAuth2 の `Introspect` からの内部呼び出しに限る。エポックを巻き戻す操作は、どの権限にも存在しない。

## Design

### Internal Interfaces

#### CheckRevocationEpoch
指定した Agent の失効エポックを返す。OAuth2 の Introspect または保護 API がアクセストークンの `issued_at` と比較し、失効エポック以前に発行されたトークンをフェイルクローズで無効と判定するために呼び出す。

#### ResolveSecurityEventSubject
受理した SET の `events` クレームから、失効を反映する対象のテナントローカルなプリンシパルを決める。IdMagic 自身の送信側が生成する独自形式 (`subject_type` / `tenant_id` / `principal_id`) と、RFC 9493 の Subject Identifier (`format=iss_sub` と `format=opaque`) の両方を解釈する。判別は `format` メンバーの有無で行い、両形式を同時に満たす表現は受け付けない。RFC 9493 形式は自身のテナントを名乗らないため、テナントは受信ストリームが属するテナントで決まる。識別子は Agent の識別子として解決し、一致しなければ Agent に束縛済みの `OAuth2Client` の識別子として解決する。どちらでも解決できない場合はフェイルクローズで拒否する。
- Result invariant: subject_resolves_within_receiving_stream_tenant(input.stream_id)

#### AdvanceRevocationEpoch
KillAgent、DisableAgent、UnbindAgentCredential、所有者（`owner_user_id`）の無効化または削除、受理済みの SecurityEvent のいずれかを契機として、対象 Agent 群の失効エポックを進める。所有者のオフボーディングでは、対象テナント内で `owner_user_id` が一致するすべての Agent を一括して進める。失効エポックを既存値より前の時刻に戻すことはない。
- Result invariant: epoch_advances_monotonically_for_all(input.agent_ids)

## Scenarios

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
