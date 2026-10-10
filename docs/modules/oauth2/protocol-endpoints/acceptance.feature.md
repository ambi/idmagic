# Feature: プロトコルのエンドポイントの例

## Rule: REQ-OAUTH2-014 Discovery Metadata は宣言された全エンドポイントを広告する

### Example: EX-OAUTH2-014-01 通常経路

- When Discovery Metadata を取得する
- Then レスポンスに `issuer`、`authorization_endpoint`、`token_endpoint`、`userinfo_endpoint`、`jwks_uri`、`introspection_endpoint`、`revocation_endpoint`、`pushed_authorization_request_endpoint`、`device_authorization_endpoint`、`backchannel_authentication_endpoint`、`registration_endpoint` が含まれる

## Rule: REQ-OAUTH2-030 RFC 8414 メタデータ文書は OIDC Discovery と同等の内容を返す

### Example: EX-OAUTH2-030-01 通常経路

- When Authorization Server メタデータを取得する
- Then レスポンスに `issuer`、`authorization_endpoint`、`token_endpoint`、`jwks_uri`、`grant_types_supported` が含まれる

## Rule: REQ-OAUTH2-033 realm 接頭辞付きの Discovery Metadata は同じ接頭辞を持つ発行者を返す

### Example: EX-OAUTH2-033-01 通常経路

- Given tenant_id "acme" が Active で存在する
- When /realms/acme/.well-known/openid-configuration を取得する
- Then レスポンスの `issuer` は基底 URL + `/realms/acme` となる
- Then レスポンスの `authorization_endpoint` は基底 URL + `/realms/acme/authorize` となる

## Rule: REQ-OAUTH2-040 プロトコルエンドポイントは閾値を超えたリクエストをレート制限で拒否する

### Background:

- Given クライアントがある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している

### Example: EX-OAUTH2-040-01 通常経路

- When 同一 window 内で追加リクエストを送る
- Then エラー "RateLimitedError" (HTTP 429、Retry-After ヘッダ付き)

### Scenario Outline: 条件ごとの結果

- When 同一 window 内で追加リクエストを送る
- But <condition>
- Then <result>
- And エラー "RateLimitedError"

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-OAUTH2-040-02 | 対象 endpoint が /tokenである | client_id と IP の組で閾値超過している状態でトークンを要求する |
  | EX-OAUTH2-040-03 | 対象 endpoint が /authorize または /par である | IP と client_id の組で閾値超過している状態で認可リクエストを送る |
  | EX-OAUTH2-040-04 | 対象 endpoint が /device_authorization である | client_id と IP の組で閾値超過している状態でデバイス認可を開始する |
  | EX-OAUTH2-040-05 | 対象 endpoint が /bc-authorize である | client_id と IP の組で閾値超過している状態で backchannel 認可を開始する |

### Example: EX-OAUTH2-040-06 共有カウンタストアに到達できない

- When 同一 window 内で追加リクエストを送る
- But 共有カウンタストアに到達できない
- Then リクエストは fail-closed で "RateLimitedError" として拒否される
