# Feature: SET の受信の例

## Rule: REQ-SHAREDSIGNALS-003 署名が不正な SET は反映せずに拒否する

### Example: EX-SHAREDSIGNALS-003-01 通常経路

- Given `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- When 署名が不正な SET を "S1" へ POST する
- Then リクエストは `SecurityEventRejectedError` で拒否される
- Then "SecurityEventRejected" が `verification_result=rejected_signature` で発行される
- Then AgentRevocationEpoch は変化しない

## Rule: REQ-SHAREDSIGNALS-004 同じ jti の SET は一度だけ反映する

### Example: EX-SHAREDSIGNALS-004-01 通常経路

- Given `direction=Receive` の SsfStream "S1" に有効な SET（`jti="J1"`）を送信済みである
- When 同一 jti "J1" の SET を再度 "S1" へ POST する
- Then リクエストは `SecurityEventRejectedError` で拒否される
- Then "SecurityEventRejected" が `verification_result=rejected_replay` で発行される

## Rule: REQ-SHAREDSIGNALS-005 発行者が一致しても他テナントのストリームでは受理しない

### Example: EX-SHAREDSIGNALS-005-01 通常経路

- Given テナント "T1" が `direction=Receive` の SsfStream "S1" を登録している
- And テナント "T2" の Agent を subject とする SET がある
- When SET を "S1" へ POST する
- Then リクエストは `SecurityEventRejectedError` で拒否される
- Then "SecurityEventRejected" が `verification_result=rejected_subject_unresolved` で発行される

## Rule: REQ-SHAREDSIGNALS-010 RFC 9493 の Subject Identifier で送られた SET も主体を解決する

### Background:

- Given `direction=Receive` の SsfStream "S1" に `trusted_issuer="https://issuer.example"` が登録されている
- And テナント "T1" に Agent "A1" が存在し、OAuth2Client "C1" に束縛されている

### Example: EX-SHAREDSIGNALS-010-01 通常経路

- When `format=iss_sub`、`iss="https://issuer.example"`、`sub="A1"` の Subject Identifier を持つ SET を "S1" へ POST する
- Then "A1" の AgentRevocationEpoch が前進する
- Then "SecurityEventReceived" が発行される

### Scenario Outline: 条件ごとの結果

- When `format=iss_sub`、`iss="https://issuer.example"`、`sub="A1"` の Subject Identifier を持つ SET を "S1" へ POST する
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-SHAREDSIGNALS-010-02 | `sub` が "A1" ではなく束縛先の "C1" である | 同じく "A1" として解決され、失効エポックが前進する |
  | EX-SHAREDSIGNALS-010-03 | `format=opaque`、`id="A1"` である | 同じく "A1" として解決される（テナントは受信ストリームが属するテナントで決まる） |
  | EX-SHAREDSIGNALS-010-04 | `iss` が "S1" の `trusted_issuer` と一致しない | `SecurityEventRejectedError` で拒否され、"SecurityEventRejected" が `verification_result=rejected_subject_unresolved` で発行される |
  | EX-SHAREDSIGNALS-010-05 | `format=email` など IdMagic が解釈しない形式である | `SecurityEventRejectedError` で拒否される |
  | EX-SHAREDSIGNALS-010-06 | `sub` がどの Agent にも束縛先クライアントにも一致しない | `SecurityEventRejectedError` で拒否される |
