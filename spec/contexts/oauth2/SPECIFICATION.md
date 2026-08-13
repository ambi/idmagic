---
context: oauth2
updated_at: 2026-08-14
---

# OAuth2 Specification

## Overview

OAuth 2.0 / OIDC プロトコルファミリーの全責務を所有する。Client metadata
とその Dynamic Registration、認可判断 (Authorize / Consent /
Authorization Code / PAR / Device Authorization / RP-Initiated Logout)、
トークン発行とライフサイクル (AccessToken / RefreshToken / ID Token /
Introspect / Revoke / UserInfo / proof-of-possession)、Discovery /
Authorization Server Metadata / 健全性報告
を本 bounded context に集約する。

The `oauth2` context implements an OAuth 2.0 / OIDC authorization server as a set of feature
slices — `authorization/`, `client/`, `consent/`, `device/`, `token/` — each owning its own
`domain`, `ports`, `usecases`, and `db_memory`/`db_postgres` adapters. The context-level `domain`,
`ports`, and `usecases` packages are compatibility facades over the slices, `handlers_http` is the
shared HTTP/persistence adapter, and `module.go` is the single composition root. Read this document
mechanism by mechanism: authorization/device lifecycles, PKCE/PAR, client authentication, token
formats and rotation, sender constraints, consent, authorization policy, discovery, the device
grant, lifetime/security configuration, agent principals and delegation, rich authorization
requests, and session/logout binding.

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ResourceOwner | 保護リソースに対するアクセス権限を付与できる実体。本アプリではエンドユーザー。 | resource_owner, リソースオーナー, EndUser |
| Client | ResourceOwner に代わって保護リソースへのアクセスを要求するアプリケーション。client_id で識別される。client_id が Client ID Metadata Document (CIMD) の URL 形状の場合、永続登録なしに認可リクエストごとにドキュメントを解決する。 | client, クライアント, RelyingParty |
| AuthorizationServer | ResourceOwner を認証し Client にトークンを発行するサーバー。本アプリそのもの。OIDC では IdP と同義。 | authorization_server, IdP, IdentityProvider |
| ResourceServer | AccessToken を提示する Client に対して保護リソースを返すサーバー。本アプリの外部。 | resource_server |
| McpResourceServer | MCP エコシステムでツール/データソースを提供する ResourceServer の tenant-scoped 登録。canonical resource URI と許可 scope を所有し、Protected Resource Metadata と resource indicator 検証の基準になる。 | mcp_resource_server |
| ResourceIndicator | RFC 8707 の resource パラメータ。要求 audience を表す絶対 URI で fragment を含まない。1 トークンにつき 1 個のみを許可し、複数指定・未登録・無効は fail-closed で拒否する。 | resource_indicator |
| Public | client_secret を安全に保持できない Client（SPA, ネイティブアプリ）。PKCE 必須。 | public |
| Confidential | サーバーサイドで client_secret または private_key を保持できる Client。 | confidential |
| AuthorizationCode | 短命（60秒以下）の一度限り使用される中間トークン。/authorize で発行され /token で交換される。grant_type 値「authorization_code」と同じ文字列を共有する（RFC 6749 §4.1）。 | authorization_code, 認可コード |
| RefreshToken | AccessToken の再発行に使うトークン。提示のたびにローテーションされる。絶対 TTL 30 日。grant_type 値「refresh_token」と同じ文字列を共有する（RFC 6749 §6）。 | refresh_token, リフレッシュトークン |
| ClientCredentials | Client 自身が認可主体となる machine-to-machine フロー (RFC 6749 §4.4)。confidential のみ。 | client_credentials, M2Mグラント |
| DeviceCode | 入力制約のあるデバイス向けのデバイス認可グラント (RFC 8628)。grant_type は URN。 | urn:ietf:params:oauth:grant-type:device_code, device_code, デバイスコード |
| TokenExchange | 既存トークンを別トークンに交換するトークン交換グラント (RFC 8693)。本アプリは 2 種類の subject_token を扱う: (1) 自己発行トークンの委任 (delegation) — subject_token / actor_token は本 IdP が発行し IntrospectAccessToken を通過したものに限る、(2) workload identity federation — subject_token_type が JwtSvid のとき、外部 attestation token を WorkloadIdentity の VerifyWorkloadAttestation で検証し、テナント登録済みの AgentWorkloadBinding で写した Agent の資格情報として発行する。grant_type は URN。 | urn:ietf:params:oauth:grant-type:token-exchange, token-exchange, トークン交換 |
| Ciba | Client-Initiated Backchannel Authentication (OpenID CIBA Core)。client が帯域外の authentication device 経由で ResourceOwner の判断を求める decoupled なフロー。本アプリは poll mode のみを実装し、grant_type は URN。 | urn:openid:params:grant-type:ciba, ciba, CIBA, backchannel authentication |
| ApprovalRequest | 人間の承認が成立するまで client の要求を保留する承認要求。承認対象 User・要求元 client / Agent・要求 scope / AuthorizationDetails・binding message・期限を持ち、Pending から一方向に進む。CIBA の輸送語彙に依存しない一般形で保持する。 | approval_request, 承認要求 |
| AuthReqId | /bc-authorize が発行する承認要求の bearer secret (CIBA Core §7.3)。token endpoint の ciba grant で提示する。保存は SHA-256 ハッシュのみで、画面にも監査ログにも出さない。 | auth_req_id |
| BindingMessage | 承認画面に表示する短い識別文 (CIBA Core §7.1)。別要求の取り違え承認を防ぐ補助であり、要求内容の提示を代替しない。 | binding_message |
| TokenDeliveryMode | 承認成立をどう client へ届けるかの区分 (CIBA Core §4)。本アプリは poll のみ実装し、ping / push は広告しない。 | backchannel_token_delivery_mode, poll, ping, push |
| UnknownUserId | CIBA error code unknown_user_id。login_hint / id_token_hint から承認対象 User を解決できない。存在有無を開示しないため非 active・別テナントも同じ error に畳む。 | unknown_user_id |
| Pending | 承認要求が起票され、まだ判断されていない初期状態。 | pending |
| Consumed | 承認済みの承認要求がちょうど一度トークン化された終端状態。 | consumed |
| Consume | 承認済みの承認要求をトークンへ一度きり消費する。 | consume |
| AuthorizationDetails | 構造化された細粒度の権限要求 (RFC 9396)。type で識別される JSON オブジェクトの配列として、対象・操作・上限・条件を表し、/authorize・/par・/token で要求・同意・トークン反映する。本アプリは受理する type をテナント登録スキーマに限定し fail-closed に検証する。 | authorization_details, RAR, Rich Authorization Requests, リッチ認可リクエスト |
| AccessToken | ResourceServer にアクセスする際に提示するトークン。JWT (PS256 / ES256) として発行、TTL 600秒。 | access_token, アクセストークン |
| IdToken | OIDC が定める、ResourceOwner の認証結果を表明する JWT。iss/sub/aud/exp/iat/auth_time/nonce/azp を含む。 | id_token, IDトークン |
| Pkce | Proof Key for Code Exchange (RFC 7636)。code_challenge / code_verifier で認可コード横取り攻撃を防ぐ。本アプリでは例外なく必須。 | pkce, PKCE |
| Dpop | Demonstrating Proof of Possession (RFC 9449)。JWK サムプリント (jkt) でトークンを所有鍵にバインドする。 | dpop, DPoP |
| Mtls | Mutual-TLS Client Authentication (RFC 8705)。TLS クライアント証明書で Client を認証する。 | mtls, mTLS, tls_client_auth |
| Par | Pushed Authorization Requests (RFC 9126)。/par に認可リクエストを事前 POST し /authorize は request_uri のみで参照する。FAPI 2.0 必須。 | par, PAR |
| SenderConstrainedToken | 所有証明（DPoP または mTLS）と組み合わさったトークン。所有者以外による再利用を防ぐ。 |  |
| Issuer | トークンを発行した AuthorizationServer の URL 識別子。 | iss |
| Subject | トークンの主体（通常は ResourceOwner）の仮名化された識別子。削除後も監査ログに残る。 | sub |
| Audience | トークンの想定受信者の識別子。通常は Client の client_id。 | aud |
| JwtId | JWT 一意識別子。リプレイ防止と監査の追跡に使用。 | jti |
| Nonce | Client が認可リクエストに含め、ID トークンに含めて返される値。IDトークンのリプレイ防止。 |  |
| Scope | Client が要求する権限の集合。Client メタデータで宣言された集合の部分集合でなければならない。 |  |
| Consent | ResourceOwner が Client に対して特定の Scope を付与する意思表示。tenant / subject / client / scopes / granted_at / expires_at を永続化する。管理者は参照と撤回だけが可能で、付与や scope 拡張を代行できない。 | consent, 同意 |
| AdminConsentManagement | Administrator が所属テナント内の Consent を監査目的で参照し、必要時に撤回する管理操作。同意の作成や scope 拡張は含まない。 | consent administration, 同意管理 |
| PreferredUsername | 表示用のユーザー名。可変。 | preferred_username |
| ClientSecretBasic | HTTP Basic 認証（レガシークライアント向け）。 | client_secret_basic |
| ClientSecretPost | フォームボディ内のクレデンシャル。 | client_secret_post |
| PrivateKeyJwt | RFC 7523 — client_assertion による署名済み JWT 認証。confidential 推奨。 | private_key_jwt |
| TlsClientAuth | RFC 8705 §2 — Mutual-TLS Client Authentication（PKI 結合）。Mtls 機構の一形態を token_endpoint_auth_method の値として表現したもの。 | tls_client_auth |
| None | クライアント認証なし（public クライアント、PKCE が代替防御）。 | none |
| PS256 | RSASSA-PSS using SHA-256 (RFC 7518)。 |  |
| ES256 | ECDSA using P-256 and SHA-256 (RFC 7518)。 |  |
| S256 | PKCE code_challenge_method=S256。code_challenge は BASE64URL-ENCODE(SHA256(code_verifier))。 | S256 |
| Hwk | RFC 8176 の hardware-secured key 認証メソッド。 | hwk |
| Swk | RFC 8176 の software-secured key 認証メソッド。 | swk |
| Code | response_type=code。本アプリが対応する唯一のレスポンスタイプ。 | code |
| Query | authorization response parameters を redirect_uri の query component で返す response_mode。 | query |
| FormPost | authorization response parameters を自動送信 HTML form で返す response_mode。 | form_post |
| InvalidRequest | OAuth error code invalid_request。必須パラメータ欠落、不正値、重複などのリクエスト不備。 | invalid_request |
| InvalidClient | OAuth error code invalid_client。クライアント認証失敗。 | invalid_client |
| InvalidGrant | OAuth error code invalid_grant。認可コード、refresh token、device_code などの grant が無効。 | invalid_grant |
| UnauthorizedClient | OAuth error code unauthorized_client。クライアントが当該 grant_type を使えない。 | unauthorized_client |
| UnsupportedGrantType | OAuth error code unsupported_grant_type。未対応の grant_type。 | unsupported_grant_type |
| InvalidScope | OAuth error code invalid_scope。要求 scope が不正、未知、または許可外。 | invalid_scope |
| InvalidToken | Bearer token error code invalid_token。トークンが無効、期限切れ、改ざん、または失効済み。 | invalid_token |
| InvalidDpopProof | DPoP error code invalid_dpop_proof。DPoP proof JWT の検証失敗。 | invalid_dpop_proof |
| AccessDenied | OAuth error code access_denied。ResourceOwner またはポリシーによる拒否。 | access_denied |
| ExpiredToken | Device Authorization Grant error code expired_token。device_code が期限切れ。 | expired_token |
| InsufficientScope | Bearer token error code insufficient_scope。提示 token の scope が不足。 | insufficient_scope |
| ServerError | OAuth error code server_error。AuthorizationServer 側の予期せぬ内部エラー。 | server_error |
| InvalidTarget | RFC 8707 error code invalid_target。resource パラメータが未登録・無効・複数指定、または McpResourceServer が Disabled。 | invalid_target |
| Fapi2SecurityProfile | FAPI 2.0 Security Profile。tls_client_auth または private_key_jwt + PAR 必須。 | fapi_2_security_profile |
| Received | 認可リクエストを受信した直後の初期状態。 | received |
| AuthenticationPending | ResourceOwner の認証を待っている状態。 | authentication_pending |
| Authenticated | ResourceOwner の認証が完了した状態。 | authenticated |
| ConsentPending | ResourceOwner に Scope への同意を求めている状態。 | consent_pending |
| Consented | ResourceOwner が要求 Scope に同意した状態。 | consented |
| CodeIssued | AuthorizationCode を Client に発行した状態。 | code_issued |
| Exchanged | AuthorizationCode が /token で正常に交換された終端状態。 | exchanged |
| Rejected | 認可リクエストが拒否された終端状態。 | rejected |
| Expired | 期限切れにより無効化された終端状態。 | expired |
| Validate | 認可リクエストの構文・必須パラメータ・redirect_uri を検証する。検証成功で AuthenticationPending へ遷移し、ログイン UI 提示を含む。 | validate |
| AuthenticateUser | ResourceOwner の認証を実行する。 | authenticate_user |
| RequestConsent | ResourceOwner に Scope への同意を求める（同意 UI 表示）。 | request_consent |
| GrantConsent | ResourceOwner が同意を付与する。 | grant_consent |
| IssueCode | AuthorizationCode を Client に発行する。 | issue_code |
| RedeemCode | Client が AuthorizationCode を /token で AccessToken と交換する。 | redeem_code |
| Reject | 認可リクエストを拒否する。 | reject |
| Expire | 期限切れにより認可リクエストを無効化する。 | expire |
| Issued | device_code と user_code を発行した初期状態。 | issued |
| UserCodeEntered | ユーザーが verification_uri で user_code を入力した状態。 | user_code_entered |
| AuthorizationPending | ユーザー承認待ち状態（device_code フロー）。 | authorization_pending |
| Approved | ユーザーが承認した状態。 | approved |
| Denied | ユーザーが拒否した、または user_code 誤り過多で失敗した状態。 | denied |
| EnterUserCode | ユーザーが verification_uri で user_code を入力する。 | enter_user_code |
| Approve | ユーザーが承認する。 | approve |
| Deny | ユーザーが拒否する。 | deny |
| Exchange | デバイスが /token で device_code を交換する。 | exchange |
| SlowDown | ポーリング間隔を増やすよう指示する。 | slow_down |
| Active | トークンまたは鍵が現在有効で第一線で使われている状態。RefreshToken と SigningKey で共用。 | active |
| Rotated | 子トークンに引き継がれ親が消費された状態。家族失効の対象になりうる。 | rotated |
| Revoked | 失効済み。/revoke もしくは家族失効により遷移する。RefreshToken と Consent で共用。 | revoked |
| Rotate | 親 RefreshToken を Rotated とし、新しい子トークンを発行する。 | rotate |
| RevokeToken | トークンを失効させる（/revoke または家族失効）。 | revoke_token |
| Deliver | LogoutNotification の配送 (backchannel_logout_uri への logout token POST) が成功する。 | deliver |
| Exhaust | LogoutNotification の配送が max_attempts に到達し dead-letter として確定する。 | exhaust |
| Redeemed | 認可コードが /token で正規に交換された終端状態（AuthorizationCodeRecord）。AuthorizationCodeFlow の Exchanged と並行。 | redeemed |
| Stored | /par で受領済みかつ未参照の request_uri。 | stored |
| Used | request_uri が一度参照済みの終端状態。再使用不可。 | used |
| Use | /authorize から request_uri を参照する。一度きり。 | use |
| Granted | 同意が付与された状態（Consent の初期状態）。 | granted |
| RevokeConsent | ResourceOwner が同意を撤回する（GDPR Art.7(3)）。 | revoke_consent |
| Implicit | implicit grant は RFC 9700 §2.1.2 により参照実装から除外する。 | __excluded_implicit |
| PasswordGrant | Resource Owner Password Credentials grant は RFC 9700 §2.4 により参照実装から除外する。 | __excluded_password |

## Standards

### The OAuth 2.0 Authorization Framework

RFC 6749 — https://www.rfc-editor.org/rfc/rfc6749.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6749-AUTHORIZATION-CODE | required | MUST | Authorization Code Grantを認可エンドポイントとトークンエンドポイントで提供し、認可要求の単一値 security parameter の重複をinvalid_requestとして拒否する。 |
| RFC6749-CLIENT-CREDENTIALS | optional | MAY | confidential clientに限りClient Credentials Grantを許可する。 |
| RFC6749-IMPLICIT | excluded | MAY | Implicit Grantを提供する。 |
| RFC6749-PASSWORD-GRANT | excluded | MAY | Resource Owner Password Credentials Grantを提供する。 |

### The OAuth 2.0 Authorization Framework Bearer Token Usage

RFC 6750 — https://www.rfc-editor.org/rfc/rfc6750.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6750-AUTHORIZATION-HEADER | required | MUST | Bearer Access TokenはAuthorizationヘッダーで受け付ける。 |
| RFC6750-QUERY-TOKEN | excluded | MAY | URI query parameterによるAccess Token提示を受け付ける。 |

### OAuth 2.0 Token Revocation

RFC 7009 — https://www.rfc-editor.org/rfc/rfc7009.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7009-REVOCATION-ENDPOINT | required | MUST | 認証済みクライアントへトークン失効エンドポイントを提供する。 |
| RFC7009-UNKNOWN-TOKEN | required | MUST | 無効または他クライアント所有のトークンに対しても成功応答を返し情報を漏らさない。 |

### JSON Web Key

RFC 7517 — https://www.rfc-editor.org/rfc/rfc7517.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7517-JWKS | required | MUST | 公開可能な検証鍵をJWK Setとして配布する。 |

### JSON Web Algorithms

RFC 7518 — https://www.rfc-editor.org/rfc/rfc7518.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7518-SIGNATURE-ALGORITHMS | required | MUST | JWT署名をPS256またはES256で行い、対称鍵署名をAccess TokenとID Tokenに使用しない。 |

### JSON Web Token

RFC 7519 — https://www.rfc-editor.org/rfc/rfc7519.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7519-REGISTERED-CLAIMS | required | MUST | JWTのissuer、subject、audience、有効期限、発行時刻、一意識別子を用途に応じて検証・発行する。 |

### JSON Web Token Profile for OAuth 2.0 Client Authentication and Authorization Grants

RFC 7523 — https://www.rfc-editor.org/rfc/rfc7523.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7523-CLIENT-ASSERTION | optional | MAY | client assertionの署名、issuer、subject、audience、有効期限、jtiを検証する。 |

### OAuth 2.0 Dynamic Client Registration Protocol

RFC 7591 — https://www.rfc-editor.org/rfc/rfc7591.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7591-REGISTER | optional | MAY | Client metadataを受け取りclient_idと登録結果を返す。 |
| RFC7591-REDIRECT-URI | required | MUST | Authorization Code Grantを利用するClientにはredirect_uri登録を要求する。 |

### OAuth Client ID Metadata Document

draft-ietf-oauth-client-id-metadata-document-00 — https://www.ietf.org/archive/id/draft-ietf-oauth-client-id-metadata-document-00.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| CIMD00-URL-SHAPE | required | MUST | client_idはhttps schemeかつ非空pathを持つURLでなければならない。 |
| CIMD00-FETCH | required | SHOULD | URL形式のclient_idを検出したらそのURLからmetadata documentをfetchする。 |
| CIMD00-CLIENT-ID-MATCH | required | MUST | fetchしたdocument内のclient_idフィールドがfetch元URLと厳密一致することを検証する。 |
| CIMD00-REDIRECT-VALIDATE | required | MUST | 認可リクエストのredirect_uriがdocumentのredirect_uris一覧に含まれることを検証する。 |
| CIMD00-STRUCTURE | required | MUST | documentが valid JSON であり client_id・client_name・redirect_uris (非空) を含むことを検証する。不正な場合はfail-closedで拒否する。 |
| CIMD00-CACHE | partial | SHOULD | HTTP cache headerに従いdocumentをcacheする。 |
| CIMD00-PRIVATE-KEY-JWT | excluded | MAY | private_key_jwtクライアント認証をinline jwksまたはjwks_uri経由でサポートしてよい。 |

### Proof Key for Code Exchange by OAuth Public Clients

RFC 7636 — https://www.rfc-editor.org/rfc/rfc7636.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7636-VERIFY | required | MUST | Token requestのcode_verifierを認可時のcode_challengeと照合し、認可要求でPKCE parameterの重複を許容しない。 |
| RFC7636-S256 | required | SHOULD | code_challenge_methodはS256だけを許可する。 |
| RFC7636-PLAIN | excluded | MAY | plain code challenge methodを許可する。 |

### OAuth 2.0 Token Introspection

RFC 7662 — https://www.rfc-editor.org/rfc/rfc7662.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7662-INTROSPECT | required | MUST | 認証済みResource Serverへactive状態と許可されたメタデータを返す。 |
| RFC7662-INACTIVE | required | MUST | 無効なトークンにはactive=falseだけを返す。 |

### Proof-of-Possession Key Semantics for JSON Web Tokens

RFC 7800 — https://www.rfc-editor.org/rfc/rfc7800.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7800-CONFIRMATION | optional | MAY | Sender-constrained tokenの確認鍵情報をcnf claimへ格納する。 |

### Authentication Method Reference Values

RFC 8176 — https://www.rfc-editor.org/rfc/rfc8176.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8176-AMR | required | MAY | 実際に成立した認証方法をamr値として記録する。 |

### OAuth 2.0 Authorization Server Metadata

RFC 8414 — https://www.rfc-editor.org/rfc/rfc8414.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8414-METADATA | required | MUST | issuerと利用可能なエンドポイント・機能をmetadata文書として公開する。 |

### OAuth 2.0 Device Authorization Grant

RFC 8628 — https://www.rfc-editor.org/rfc/rfc8628.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8628-DEVICE-AUTHORIZATION | optional | MUST | device_code、user_code、verification_uriを発行しResourceOwnerの判断を受け付ける。 |
| RFC8628-POLLING | required | MUST | authorization_pending、slow_down、expired_tokenのポーリング意味論を守る。 |

### OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens

RFC 8705 — https://www.rfc-editor.org/rfc/rfc8705.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8705-CLIENT-AUTH | optional | MAY | 登録済みSubject DNと検証済みクライアント証明書を照合してClientを認証する。 |
| RFC8705-CERT-BOUND | optional | MAY | Access Tokenをクライアント証明書thumbprintへバインドしResource access時に照合する。 |

### Resource Indicators for OAuth 2.0

RFC 8707 — https://www.rfc-editor.org/rfc/rfc8707.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8707-AUDIENCE | required | SHOULD | 発行するAccess Tokenへ非空のaudienceを設定し、意図しないResource Serverでの利用を防ぐ。 |
| RFC8707-MCP-RESOURCE-BINDING | required | MUST | resource パラメータで指定された McpResourceServer へ Access Token の audience を厳格に限定し、未登録・無効・複数指定の resource は fail-closed で拒否する。Authorize / PushAuthorizationRequest / Token(authorization_code redemption・refresh rotation・client_credentials・device_code・token-exchange) の全経路で resource 指定時に一様に適用する。resource 未指定時は従来どおり client_id を audience とする後方互換を保つ。 |

### JSON Web Token Profile for OAuth 2.0 Access Tokens

RFC 9068 — https://www.rfc-editor.org/rfc/rfc9068.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9068-CLAIMS | required | MUST | JWT Access Tokenへiss、sub、aud、exp、iat、jti、client_idを含める。 |
| RFC9068-ASYMMETRIC-SIGNATURE | required | MUST | Access Tokenを非対称アルゴリズムで署名し公開鍵で検証可能にする。 |

### OAuth 2.0 Pushed Authorization Requests

RFC 9126 — https://www.rfc-editor.org/rfc/rfc9126.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9126-PAR | optional | MAY | Client認証済みPARを保存し短命なrequest_uriを返す。 |
| RFC9126-SINGLE-USE | required | SHOULD | request_uriは短命かつ一度だけ使用可能とする。 |

### OAuth 2.0 Authorization Server Issuer Identification

RFC 9207 — https://www.rfc-editor.org/rfc/rfc9207.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9207-ISS | required | MUST | Authorization responseおよび安全に確定済みのredirect_uriへ返すauthorization errorへissuer識別子を含める。 |

### OAuth 2.0 Demonstrating Proof of Possession

RFC 9449 — https://www.rfc-editor.org/rfc/rfc9449.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9449-PROOF | optional | MAY | DPoP proofの署名、htm、htu、iat、jtiを検証する。 |
| RFC9449-TOKEN-BINDING | optional | MUST | DPoP利用時はAccess Tokenへjktを含め、提示proofの鍵と照合する。 |

### Best Current Practice for OAuth 2.0 Security

RFC 9700 / BCP 240 — https://www.rfc-editor.org/rfc/rfc9700.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9700-REDIRECT-MATCH | required | MUST | redirect_uriは登録値と完全一致させ、未検証URIへリダイレクトしない。 |
| RFC9700-AUTHORIZATION-CODE | required | MUST | redirect-based flowはAuthorization Code GrantとPKCEで保護する。 |
| RFC9700-SENDER-CONSTRAINT | optional | SHOULD | 高セキュリティClientではAccess TokenをDPoPまたはmTLSでsender-constrainedにする。 |
| RFC9700-REFRESH-REPLAY | required | MUST | Refresh Tokenをローテーションし再利用検知時に関連トークンを失効する。 |

### OAuth 2.0 Protected Resource Metadata

RFC 9728 — https://www.rfc-editor.org/rfc/rfc9728.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9728-METADATA | required | MUST | 登録済み McpResourceServer ごとに resource・対応する authorization_servers・サポート scope を含む metadata文書を配信する。 |
| RFC9728-WELL-KNOWN | required | MUST | /.well-known/oauth-protected-resource で resource を指定した metadata取得を提供する。 |
| RFC9728-IDMAGIC-API | required | MUST | resource 未指定時は realm の IdMagic API metadata と account / management / SCIM scope、header / DPoP bearer method を公開する。 |

### OpenID Connect Core 1.0 incorporating errata set 1

Final — https://openid.net/specs/openid-connect-core-1_0-18.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-CORE-CODE-FLOW | required | MUST | code response typeによるOpenID Connect Authenticationを提供し、promptを空白区切りの一意なtoken集合として検証する。loginとconsentをそれぞれ適用し、noneは他tokenと併用せずUIを発生させない。 |
| OIDC-CORE-ID-TOKEN | required | MUST | ID Tokenへiss、sub、aud、exp、iatと認証コンテキストを含める。 |
| OIDC-CORE-USERINFO | required | SHOULD | openid scopeのAccess Tokenに対してsubを含むUserInfoを返す。 |
| OIDC-CORE-HYBRID-IMPLICIT | excluded | MAY | Implicit FlowおよびHybrid Flowを提供する。 |

### OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0

Final — https://openid.net/specs/openid-client-initiated-backchannel-authentication-core-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| CIBA-CORE-BACKCHANNEL-REQUEST | optional | MUST | client認証済みのbackchannel authentication requestを受け付ける。scope は必須で openid を含み、login_hint または id_token_hint のちょうど一方から承認対象Userを解決し、auth_req_id・expires_in・intervalを返す。解決できない場合はunknown_user_idで拒否する。 |
| CIBA-CORE-POLL-MODE | optional | MUST | token endpointのciba grantでauthorization_pending、slow_down、access_denied、expired_tokenのポーリング意味論を守り、承認成立後の auth_req_id をちょうど一度だけトークン化する。 |
| CIBA-CORE-BINDING-MESSAGE | optional | SHOULD | binding_messageを承認画面に表示し、client・要求scope・authorization_detailsと併せて何を承認するのかを示す。 |
| CIBA-CORE-PING-PUSH | excluded | MAY | ping および push の token delivery mode を提供する。 |
| CIBA-CORE-USER-CODE | excluded | MAY | user_code パラメータによる authentication device 側の本人確認補助を受け付ける。 |
| CIBA-CORE-SIGNED-REQUEST | excluded | MAY | 署名済み JWT による backchannel authentication request を受け付ける。 |

### OpenID Connect Discovery 1.0 incorporating errata set 1

Final — https://openid.net/specs/openid-connect-discovery-1_0-21.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-DISCOVERY-CONFIGURATION | required | MUST | well-known configurationからissuer、endpoints、supported metadataを公開する。 |

### OpenID Connect RP-Initiated Logout 1.0

Final — https://openid.net/specs/openid-connect-rpinitiated-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-LOGOUT-ENDPOINT | required | MUST | end_session_endpointを公開しRPからのlogout要求を受け付ける。 |
| OIDC-LOGOUT-REDIRECT | required | MUST | post_logout_redirect_uriはClientに登録済みの値だけを許可する。 |
| OIDC-LOGOUT-ID-TOKEN-HINT | required | SHOULD | id_token_hintが与えられた場合、署名・issuer・audience・subject・sidを検証してlogout対象のsessionとclientを解決し、client_idパラメータと矛盾するhintを拒否する。 |

### OpenID Connect Front-Channel Logout 1.0

Final — https://openid.net/specs/openid-connect-frontchannel-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-FRONTCHANNEL-IFRAME | required | MUST | end_session応答にfrontchannel_logout_uriを登録した各clientへのiframeを含め、frontchannel_logout_session_required=trueのclientにはiss/sidクエリパラメータを付与する。 |
| OIDC-FRONTCHANNEL-BEST-EFFORT | required | MUST | iframeの到達失敗を許容し、ローカルsession revokeの成否に影響させない。 |

### OpenID Connect Back-Channel Logout 1.0

Final — https://openid.net/specs/openid-connect-backchannel-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-BACKCHANNEL-LOGOUT-TOKEN | required | MUST | logout tokenにiss、sub、aud、iat、jti、events (http://schemas.openid.net/event/backchannel-logout)、対象がbrowser session由来ならsidを含めて署名する。 |
| OIDC-BACKCHANNEL-DELIVERY-RETRY | required | MUST | backchannel_logout_uriへの配送失敗を再試行可能なjobとして扱い、ローカルsession/refresh tokenの失効を配送成否に依存させない。 |
| OIDC-BACKCHANNEL-REPLAY | required | MUST | logout tokenのjtiによりRP側のreplay検出を可能にする一意な値を発行する。 |

### OpenID Connect Session Management 1.0

Draft 28 — https://openid.net/specs/openid-connect-session-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-SESSION-MGMT-CHECK-IFRAME | optional | MAY | check_session_iframeを公開し、RPからのpostMessageに対しOP session状態を応答する。 |

### FAPI 2.0 Security Profile

Final — https://openid.net/specs/fapi-security-profile-2_0-final.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| FAPI2-PROFILE-SELECTION | optional | MUST | Fapi2SecurityProfileを選択したClientにだけ本プロファイルの追加制約を適用する。 |
| FAPI2-PAR-PKCE | optional | MUST | FAPI ClientはPARとS256 PKCEを使用する。 |
| FAPI2-CLIENT-AUTH | optional | MUST | FAPI Clientはprivate_key_jwtまたはmTLSで認証する。 |
| FAPI2-SENDER-CONSTRAINT | optional | MUST | FAPI Access TokenをDPoPまたはmTLSでsender-constrainedにする。 |

## State Transitions

### ClientSecretCredentialLifecycle

client secret credential は発行時に Active となり、期限到達で Expired、管理者の個別失効で Revoked となる。Revoked は期限より優先して表示する。

Initial: `Active`
Terminal: `Expired`, `Revoked`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Expire | now() >= expires_at | Expired |  |
| Active | ClientSecretRevoked | — | Revoked |  |

### AuthorizationCodeFlow

/authorize から /token に至る authorization request のライフサイクル。

Initial: `Received`
Terminal: `Exchanged`, `Rejected`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Received | Validate | — | AuthenticationPending |  |
| Received | Reject | — | Rejected |  |
| AuthenticationPending | AuthenticateUser | — | Authenticated |  |
| AuthenticationPending | Reject | — | Rejected |  |
| AuthenticationPending | Expire | — | Expired |  |
| Authenticated | RequestConsent | — | ConsentPending |  |
| Authenticated | IssueCode | — | CodeIssued |  |
| Authenticated | Reject | — | Rejected |  |
| ConsentPending | GrantConsent | — | Consented |  |
| ConsentPending | Reject | — | Rejected |  |
| ConsentPending | Expire | — | Expired |  |
| Consented | IssueCode | — | CodeIssued |  |
| Consented | Reject | — | Rejected |  |
| CodeIssued | RedeemCode | — | Exchanged |  |
| CodeIssued | Expire | — | Expired |  |

### DeviceCodeFlow

RFC 8628 デバイス認可グラントのライフサイクル。device_code と user_code がペアで進む。

Initial: `Issued`
Terminal: `Exchanged`, `Denied`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Issued | EnterUserCode | — | UserCodeEntered |  |
| Issued | Expire | — | Expired |  |
| UserCodeEntered | Approve | — | Approved |  |
| UserCodeEntered | Deny | — | Denied |  |
| UserCodeEntered | Expire | — | Expired |  |
| Approved | Exchange | — | Exchanged |  |
| Approved | Expire | — | Expired |  |

### ApprovalRequestLifecycle

人間の承認を待つ ApprovalRequest のライフサイクル。Pending から Approved / Denied / Expired へ
一方向に進み、Consumed へ到達できるのは Approved からだけである。Consume は保存層の CAS で
ちょうど一度だけ成立し、並行するポーリングが二重にトークンを得ることはない。

Initial: `Pending`
Terminal: `Denied`, `Expired`, `Consumed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Pending | Approve | now() < expires_at | Approved |  |
| Pending | Deny | — | Denied |  |
| Pending | Expire | — | Expired |  |
| Approved | Consume | now() < expires_at | Consumed |  |
| Approved | Expire | — | Expired |  |

### RefreshTokenLifecycle

RefreshToken のライフサイクル。Rotate で子トークンに引き継がれ、Revoke で失効、Expire で期限切れ。Rotated 後も家族失効により Revoked へ遷移しうる（RFC 9700 §4.14）。

Initial: `Active`
Terminal: `Revoked`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Rotate | now() < absolute_expires_at | Rotated |  |
| Active | RevokeToken | — | Revoked |  |
| Active | Expire | — | Expired |  |
| Rotated | RevokeToken | — | Revoked |  |
| Rotated | Expire | — | Expired |  |

### LogoutNotificationLifecycle

LogoutNotification のライフサイクル。Deliver で成功確定、Exhaust で
max_attempts 到達による最終失敗確定 (dead-letter)。Jobs 側の Retry は Pending のまま
attempts のみ増やす (状態遷移ではない)。

Initial: `Pending`
Terminal: `Delivered`, `Failed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Pending | Deliver | — | Delivered |  |
| Pending | Exhaust | — | Failed |  |

### AuthorizationCodeRecordLifecycle

発行された AuthorizationCode 本体のライフサイクル。AuthorizationCodeFlow（AuthorizationRequest 側）の Exchanged に対応するのが Redeemed。

Initial: `Issued`
Terminal: `Redeemed`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Issued | RedeemCode | now() < expires_at | Redeemed |  |
| Issued | Expire | — | Expired |  |

### ConsentLifecycle

同意レコードのライフサイクル。GDPR Art.7(3) により Granted → Revoked が可能。

Initial: `Granted`
Terminal: `Revoked`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Granted | RevokeConsent | — | Revoked |  |
| Granted | Expire | — | Expired |  |

### PARRecordLifecycle

PAR で発行された request_uri のライフサイクル。/authorize から一度だけ参照可能（RFC 9126）。

Initial: `Stored`
Terminal: `Used`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Stored | Use | now() < expires_at | Used |  |
| Stored | Expire | — | Expired |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Internal Interfaces

#### FrontChannelLogout
対象 sid に ClientSession を持つ RP のうち frontchannel_logout_uri を登録した
ものについて、EndSession 応答へ埋め込む iframe target 一覧を算出する
(OpenID Connect Front-Channel Logout 1.0)。frontchannel_logout_session_required=true
の client には iss / sid クエリパラメータを付与する。到達失敗 (RP 側 iframe load
エラー等) は許容し、ローカル revoke の成否に影響しない。

#### BackChannelLogout
1 件の LogoutNotification を配送する job handler (OpenID Connect Back-Channel
Logout 1.0)。署名済み logout token (iss, sub, aud, iat, jti, events, sid) を
target_uri へ POST し、2xx を成功、それ以外・timeout・接続失敗を再試行対象とする。
Jobs context の kind=backchannel_logout_delivery job から呼ばれ、max_attempts 到達で
LogoutNotification は Failed (dead-letter) に確定する。ローカル revoke は本 interface の
成否に関わらず既に確定済みである。

### Authorization and device lifecycles as declarative state machines

The `AuthorizationRequest` and device-code lifecycles are expressed as declarative
state-transition tables in `spec/flows/` (states, events, transitions) instead of being scattered
across `if`/`switch` logic, so that regenerating the adapter layer cannot silently drift the set of
transitions a client is allowed to make. Refresh token families are
deliberately excluded from this treatment: their state space is effectively just
`{active, revoked, rotated}`, and the parent/child rotation graph matters more than transition
legality, so they are expressed as record fields and revocation rules instead (see Refresh token
rotation below). `authorization/usecases` and `device/usecases` consume these tables directly
rather than re-implementing transition logic.

### PKCE and Pushed Authorization Requests

PKCE requirement is per-client (`require_pkce` client metadata), defaulting to required for public
and FAPI 2.0 clients and opt-in for legacy confidential clients, so RFC 6749-era confidential
deployments are not forced to migrate while public/FAPI clients keep the strongest default. Only
`S256` is supported as
`code_challenge_method`; `plain` is rejected because it lets an interceptor recover the verifier
from logs. Authorization codes stay single-use and short-lived (≤60s) independent of PKCE, since
both reuse detection and replay-window minimization depend on that.

Pushed Authorization Requests (`/par`) let a client submit authorization parameters over an
authenticated back channel and reference them from `/authorize` via `request_uri`, closing off URL
tampering, open-redirect abuse, and unauthenticated request forgery at `/authorize`. FAPI 2.0 clients
(`require_pushed_authorization_requests`) must use PAR; other clients may use either path.
`request_uri` values are single-use with a ≤600s TTL, and once `/authorize` resolves a
`request_uri`, any additional query parameters on that request are ignored in favor of the pushed
ones (RFC 9126 §4) — otherwise an attacker could reattach parameters to a legitimate pushed
request.

### Client authentication

Five `token_endpoint_auth_method`s are supported — `private_key_jwt`, `tls_client_auth`, `none`,
`client_secret_post`, `client_secret_basic` — spanning FAPI-grade asymmetric authentication down to
legacy shared-secret methods, so new clients can be steered toward the strongest options while
existing deployments keep migrating. `client_secret_jwt` (HMAC) is deliberately not implemented: once `private_key_jwt` is available, a
symmetric-key alternative only adds risk without adding capability. A failed client authentication
always returns `401 invalid_client` without revealing whether the `client_id` is registered, and an
unregistered `client_id` still pays the same verification round-trip cost, to avoid a timing oracle.

`private_key_jwt` verification in `handlers_http` is pinned to a fixed rule set so that what
Discovery advertises is what the server actually checks: signature algorithm restricted to
`PS256`/`ES256` (never `none` or HMAC), `iss == sub == client_id`, audience matching this server's
issuer or endpoint URLs, signing keys resolved from the client's registered inline `jwks` or
`jwks_uri`, a bounded assertion lifetime, and single-use `jti` replay protection kept in its own
store separate from DPoP's replay store because the two have different TTLs and audit semantics.
Combining `client_assertion`
with Basic/secret authentication on the same request is rejected as `invalid_request` per RFC 6749
§2.3.

### Client ID Metadata Documents (CIMD): registry-less client resolution

Alongside RFC 7591 Dynamic Client Registration, `client_id` values shaped as an `https` URL with a
path resolve live from a client-hosted Client ID Metadata Document instead of the `OAuth2Client`
repository. Resolution is
never persisted — the document is fetched (and cached 5 minutes) each time a `client_id` isn't
found in the repository, then mapped into the same `OAuth2Client` shape every other code path
already understands, so redirect_uri matching, consent rendering, PKCE, and scope handling need no
CIMD-specific branches. The integration point is `client/cimd_http.ClientRepositoryWithCIMD`, a
decorator that embeds `OAuth2ClientRepository` and overrides only `FindByID`: a repository hit
short-circuits before any fetch, and every other method (`Save`, `Delete`, `FindAll`, credential
listing) passes straight through untouched. It is wired once at the composition root
(`cmd/internal/bootstrap`), so `authorize.go`, `push_authorization_request.go`, and
`client_auth.go` require no changes.

The fetch itself goes through `shared/security/safehttp`, the same SSRF-hardened dialer
`tokens_jose.JWKResolver` uses for `jwks_uri` (https-only, DNS-resolved-then-public-IP-only,
validated-IP direct dialing, no environment proxy, capped redirects, short timeouts, and a body
size cap). Direct dialing is required because a proxy would resolve and connect to the final target
outside the checked dial path, bypassing the transport's SSRF boundary. The shared package keeps
both fetchers behind one hardened implementation rather than two. MVP only accepts documents that
omit `token_endpoint_auth_method` or declare `none`; anything else is rejected fail-closed, and a
document's `client_id` field must match the URL it was fetched from exactly. A resolved client's
`scope` is whatever the document self-declares (default `openid`) — the same self-declared trust
model RFC 7591 DCR already uses, not a new admin-curated catalog. A CIMD-resolved client is never
linked to an `Application`, matching the existing behavior for self-registered DCR clients: the
`ApplicationGate` already treats "no Application record" as allowed, not fail-closed.

### Token formats: JWT access tokens, opaque refresh tokens

Access tokens are issued as self-contained JWTs (RFC 9068) by default, refresh tokens as opaque,
database-backed references. The asymmetry is deliberate: with many resource servers, an `/introspect` round-trip per request
doesn't scale, so a JWT a resource server can verify from JWKS alone is the better default, with a
short (600s) TTL bounding the exposure of not being able to revoke it instantly. Refresh tokens need
rotation and family-wide revocation (see below), which is naturally a database-record operation, so
opacity avoids keeping a JWT and a revocation record in sync for no benefit; refresh tokens are
stored as SHA-256 hashes, never plaintext. `/introspect` is still exposed for resource servers that
want to confirm sender-constraint (`cnf`) or real-time revocation state, but it is not the default
verification path for JWT access tokens.

### Refresh token rotation and reuse detection

Every refresh token use rotates it: the presented token is marked `rotated`, a new one is issued
carrying `parent_id`, and every token descending from one authorization code exchange shares a
`family_id`. Presenting an already-rotated token is treated as reuse: the request is rejected, every token in that `family_id`
is revoked, and a `RefreshTokenReuseDetected` audit event fires. This applies uniformly to public
and confidential clients, to keep the operational and audit story simple. Genuinely concurrent
legitimate use (e.g., two open tabs) is not distinguished from replay — one succeeds and the other
is treated as reuse — trading an occasional forced re-login for avoiding the complexity and
external-state cost of a grace-period window. `absolute_expires_at` is fixed at issuance (30 days)
and rotation never extends it.

### Sender-constrained tokens: DPoP and mTLS

DPoP (RFC 9449) is the default sender-constraint mechanism because it works identically for web
apps, SPAs, and native clients without requiring any change to a TLS-terminating proxy; mTLS
(RFC 8705) is offered as an option for organizations that already run client PKI, particularly
FAPI/banking clients. Clients declaring the FAPI 2.0 profile must use at least one of the two;
general-profile clients opt
in via `dpop_bound_access_tokens`. DPoP proof validation checks the `jwk`-header signature plus
`htm`/`htu`/`iat`/`jti`, with a bounded clock skew and a replay window on `jti`, and issued tokens
carry the JWK thumbprint in `cnf.jkt`. mTLS validation trusts a TLS-terminating proxy to pass a
verified client certificate, matches the registered `tls_client_auth_subject_dn`, and binds issued
tokens via `cnf.x5t#S256`; `/userinfo` requires the presented certificate's thumbprint to match the
token's `cnf` before accepting it. Both access and refresh tokens carry the sender constraint —
refresh tokens via a `sender_constraint` field on the store record — so proof-of-possession survives
rotation, and `/introspect` responses include `cnf` so resource servers can re-verify DPoP proofs at
the request level.

### Consent

Consent is persisted per `(subject, client_id)` as a set of granted scopes — not per-client and not
per-interaction. A per-client grant would
silently extend to newly requested high-privilege scopes added later, which conflicts with
purpose-specific consent; asking on every interaction would create consent fatigue and pressure
toward ad hoc "remember me" shortcuts. The consent UI is skipped only when every requested scope is
already covered by an unexpired, unrevoked grant; new scopes trigger a UI that highlights only the
delta. Grants expire 365 days after being granted, aligned with periodic re-consent expectations,
and `prompt=consent` forces the UI regardless of an existing grant. Revoking consent affects future
authorizations only; revoking a client's refresh tokens at the same time is an explicit, separate
action that reuses the family-revocation mechanism from refresh token rotation, while already-issued
short-lived access tokens are left to expire naturally.

### Authorization policy (AuthZEN)

Every authorization decision in this context — client/redirect_uri checks, grant-type entitlement,
refresh token validity, sender-constraint/proof matching, `/userinfo` scope checks, `/introspect`
caller authentication — is declared in `spec/policy/client-authorization.json` and evaluated through
an AuthZEN-style `authorize({subject, action, resource, context})` interface, rather than scattered
as inline conditionals across adapters where a check could silently drop on regeneration. The
current evaluator is a local
pure-function adapter over that same policy document; swapping in an external AuthZEN service, OPA,
or Cedar later only touches the adapter, not the usecases that call `authorize()`. Every `rules[].id`
declared in the policy JSON must have a matching implementation, which an invariants test enforces
so a newly declared rule cannot silently ship unimplemented.

### Discovery

The OAuth 2.0 Authorization Server Metadata / OIDC Discovery document is a derived artifact, not a
hand-maintained one: a template in `spec/discovery.json` is read at runtime with the issuer
placeholder substituted in, and its content (supported grants, auth methods, signing algorithms,
response types, PKCE methods) is cross-checked against the other specification-core files
authoritative for those facts, such as `grants/grant-types.json`, the token schemas, and the
configured PKCE method requirements. This avoids both failure
modes of the alternatives: a hand-maintained document drifting from the implementation, and a
build-time-generated document creating a second copy that goes stale if the build step is skipped.

### Device Authorization Grant

The device flow (RFC 8628) — `POST /device_authorization`, the `/device` verification UI, and the
`device_code` grant at `/token` — consumes the state-transition table already declared in
`spec/flows/device-code-flow.json` rather than reimplementing approve/deny/exchange transitions ad
hoc. `device_code` is a 32-byte
random value stored only as a SHA-256 hash (bearer secret); `user_code` uses a reduced, unambiguous
20-character alphabet (excludes vowels and visually confusable characters) rendered in
`WDJB-MJHT`-style groups. Polling honors `authorization_pending`/`slow_down`/`access_denied`/
`expired_token` against the spec-core-owned interval and backoff increment, and an approved code
moves `approved → exchanged` before token issuance to prevent double issuance.

### Lifetime, security, and retention configuration

Protocol timing and security parameters — authorization code TTL (60s, single-use), PAR
`request_uri` TTL (600s, single-use), access token TTL (600s), ID token TTL (3600s), refresh token
TTL (14 days sliding / 30 days absolute), device and user code TTL (600s), default polling interval
(5s, +5s per `slow_down`), client-authentication and code-redemption rate limits, DPoP clock skew and
replay window, and consent record retention (7 years) — are recorded together in one place rather
than as product objectives, because they are protocol/security/operational settings, not availability
or latency SLOs with error-budget semantics. Values a single model, state, or interface can
naturally enforce are expressed there as constraints/guards/contracts; values that don't belong to
one element — a rate limit spanning multiple requests, a retention window spanning a lifecycle —
stay authoritative in that shared configuration record instead.

### Agent principals and token-exchange delegation

`Agent` is a first-class principal distinct from `User` and `OAuth2Client`: it owns identity,
ownership, purpose, and lifecycle (including a kill-switch), but deliberately holds no credential
primitives of its own — it binds to one or more existing `OAuth2Client` registrations instead, so
agent governance doesn't require a second, redundant set of credential/crypto machinery. Every
agent has a
required owner (a `User` or group), so offboarding an owner can cascade to the agent's access;
`status` (`active`/`disabled`/`killed`) is checked fail-closed on every token-issuance path, meaning
an unresolved status blocks issuance rather than allowing it. Access token claims carry an optional
principal-type marker so resource servers and the AuthZEN policy layer can distinguish agent-issued
tokens without breaking existing token consumers.

Acting on a user's behalf is implemented as OAuth 2.0 Token Exchange (RFC 8693) at `/token`. The
default outcome is delegation, not impersonation: the exchanged token keeps the original user as `sub` and
records the agent as current actor in the `act` claim, nesting prior actors inward per RFC 8693
§4.1 so a chain of sub-agent delegation stays traceable; impersonation (`act` dropped, `sub`
replaced) is available only where a client/agent is explicitly permitted, and any unresolved case
defaults to delegation because that is the side that preserves the audit trail. `may_act` and the
AuthZEN policy jointly gate which actor/audience/depth combinations are permitted, exchanges must
specify a `resource` narrowing the result to a single audience (RFC 8707), a configurable maximum
delegation depth bounds `act`-chain length, and exchanged tokens are short-lived with no refresh
token issued — continuation means re-exchanging, which keeps revocation effective. Sender
constraints on the subject token carry through to the exchanged token so proof-of-possession is not
lost in the exchange.

### Rich Authorization Requests for agent-scoped permissions

Coarse OAuth scopes cannot express "transfer up to $100 from account X," so `/authorize`, `/par`,
and `/token` (including the token-exchange grant above) accept RFC 9396 `authorization_details`,
letting a request declare structured, bounded permissions instead of a broad scope. Only `type`s
pre-registered per tenant are accepted, and each detail is schema-validated fail-closed — an
unregistered type or a schema mismatch is rejected outright rather than partially accepted. Issued
and exchanged tokens may carry only a subset of what the user consented to, and — composing with
token-exchange delegation above — a subsequent exchange may only narrow that subset further, never
widen it; the partial order used for that check is defined by the registered schema itself
(containment of targets, monotonic decrease of limits). The consent UI renders each detail from a
schema-linked, human-readable template rather than raw JSON, and resource servers treat the
IdP-issued/introspected details as the sole trust boundary — they must not reinterpret or expand
what was granted. Where a `type` and a coarse `scope` overlap the same area, the structured detail's
bound wins; a request that would let `scope` re-widen an area already bounded by
`authorization_details` is rejected.

### Backchannel human approval for agent actions (CIBA)

An autonomous agent that is about to do something consequential — move money, delete data, publish
externally — needs a human to say yes first, and that human is not sitting in the agent's request.
`POST /bc-authorize` (OpenID CIBA Core) lets an authenticated client raise an approval request out of
band, and the `urn:openid:params:grant-type:ciba` grant at `/token` holds the request open until a
person decides. This is what finally gives `AgentKind.Supervised` a runtime meaning: before this
grant existed, an agent could be *declared* supervised without any path by which supervision was
actually exercised. Which agents are *obliged* to use it is deliberately not decided here — the
default kind for every agent created so far is `supervised`, so enforcing "supervised implies
approval on every grant" now would revoke access from existing deployments; that threshold judgment
belongs to the governance layer.

CIBA is modeled as an approval capability layered onto OAuth 2.0, not as a separate authentication
method, and it does not displace consent or step-up. Consent stays what it is — a long-lived
`(subject, client_id)` scope grant; an approval request is a short-lived judgment about one action.
Step-up protects the approval *action* (the account portal demands reauthentication before a decision
is recorded) rather than being replaced by the backchannel flow.

The record that holds the decision is deliberately not shaped like CIBA. OpenID AuthZEN's Access
Request and Approval Profile abstracts "authorization cannot be decided *yet* — a precondition must
be satisfied first" into request → track → satisfy → re-evaluate, and CIBA is its most widely
deployed instance. So the aggregate is `ApprovalRequest`, keyed by a UUID, and CIBA transport
bookkeeping is explicitly separated from decision semantics: `auth_req_id` is a 32-byte bearer secret
stored only as a SHA-256 lookup digest, while `interval_seconds` and `last_polled_at` are labeled as
transport fields. They remain on the same persisted record so a decision and a concurrent poll are
serialized by one store boundary instead of being split across inconsistent records.
The account portal addresses requests by UUID, so a bearer secret never reaches a human-facing
surface. Moving to an AARP profile later should touch the adapter, not the aggregate.

`Pending → Approved | Denied | Expired` and `Approved → Consumed` is one-way, and issuance goes
through a store-level compare-and-set that flips exactly the rows still in `approved` — the same
shape as the device grant's `Exchange` — so two concurrent polls cannot both mint a token. Every
non-approved state is fail-closed at `/token`: pending polls get `authorization_pending`, polls
faster than the interval get `slow_down` (and pay a +5s interval increase), denial gets
`access_denied`, expiry gets `expired_token`, and a second exchange of a consumed request gets
`invalid_grant`.

Only `poll` delivery is implemented and advertised; `ping` is left as a seam rather than wired,
and `push` is out of the question while it would mean shipping a notification fabric. `user_code`
is advertised as unsupported: the approval screen already sits behind the user's authenticated
session plus step-up, and adding a second, weaker shared secret in front of it buys nothing.
Requests default to a 300-second lifetime; a positive `requested_expiry` up to 600 seconds is
accepted and larger or non-positive values are rejected. The polling interval reuses the device grant's 5s/+5s configuration
rather than introducing a second polling convention. Out-of-band notification reuses the existing
tenant-overridable notification catalog with one added template key, so an approval request reaches
the human the same way password resets and security alerts already do.

The approving user is resolved at `/bc-authorize` from exactly one of `login_hint` (username or email)
or a verified `id_token_hint`; missing or multiple hints are `invalid_request`. An unresolvable hint,
a non-active user, and a user in another tenant all
collapse to the same `unknown_user_id` so the endpoint cannot be used to probe which accounts exist.
The resulting request is bound to that user, and decisions require the account session, step-up, and
CSRF verification, with another user's request invisible in the list and rejected when addressed
directly by id. The approval screen renders the agent, the client, the requested scopes, and the
`authorization_details` (reusing the RAR human-readable rendering), because a `binding_message`
alone tells a person which request this is, not what it does.

Approval requests are stored the way every other volatile OAuth2 record already is — an `UNLOGGED`
Postgres table plus an in-memory adapter, swept by the shared ephemeral sweep — rather than in a
separate short-lived store, since a second storage technology for one record type would buy
nothing that the existing convention does not already provide.

### OIDC session binding and logout propagation

The `sid` claim is `LoginSession.id` itself — one value shared across every relying party for a
given browser session, not a per-RP value — because OIDC's `sid` semantics describe the OP session,
and a per-RP `sid` would make it impossible to walk from a single session revoke to every affected
RP. `sid` propagates once, at `authenticate_user` completion, into `AuthorizationRequest`, then straight
through `AuthorizationCodeRecord` → `RefreshTokenRecord` → `IdTokenClaims`; Authentication's
`LoginSession` stays the single source of truth and none of its attributes are duplicated into
OAuth2. `ClientSession` exists purely as a `(sid, client_id)` delivery index for logout
notification, not a second copy of session state. Because `RefreshTokenRecord.sid` survives
rotation, revoking "this browser session" can revoke every refresh token across every client/family
tied to that `sid` in one operation, rather than requiring a family-by-family walk.

`id_token_hint` on `/end_session` is verified fail-closed — signature, `iss`, `aud` (must agree with
an explicit `client_id` parameter rather than being silently ignored), `sub`, and `sid` — with `exp`
deliberately not checked, since an expired ID Token at logout time is the normal case for RPs.
Without a hint, resolution falls back to `client_id` plus the browser cookie. Back-channel logout
delivery is handed to the Jobs context as a durable, idempotent job rather than a bespoke queue, and
local session/refresh-token revocation is never rolled back if delivery fails; front-channel logout
is a same-request computed iframe-target list with no delivery guarantee, since RP-side iframe
failures are an accepted, unrecoverable-by-design condition. Access token revocation is explicitly
out of scope here: access tokens stay self-contained JWTs verified by signature alone, and immediate
revocation would require making every resource-server verification a store lookup — the 600-second
residual exposure is accepted instead, on top of immediate refresh-token-family revocation and RP
notification. `check_session_iframe` (OIDC Session Management 1.0) is implemented minimally —
discovery advertisement plus a static "is the browser cookie still a valid session" check — because
the underlying spec never reached Final status and major IdPs implement it inconsistently.

### Conventions

Protocol-critical behavior that has an unambiguous, enumerable shape is declared once in `spec/`
and consumed by usecases/adapters rather than re-implemented: state transitions, authorization
rules, discovery metadata, and device-flow transitions all follow this shape, so regenerating the
adapter layer cannot silently diverge from the specification. Feature slices
(`authorization/`, `client/`, `consent/`, `device/`, `token/`) each own their `domain`/`ports`/
`usecases`/`db_memory`/`db_postgres` layers; the context-level `domain`/`ports`/`usecases` packages
are compatibility facades over the slices, and `module.go` is the sole composition root.

### Design Decisions

- Authorization request and device-code lifecycles are expressed as declarative state-transition
  tables in `spec/flows/` rather than ad hoc conditional logic, so regeneration cannot silently
  drift the transitions a client is allowed to make.
- PKCE requirement is staged per client type — required by default for public and FAPI 2.0 clients,
  opt-in for legacy confidential clients — rather than mandated uniformly.
- Pushed Authorization Requests are mandatory for FAPI 2.0 clients and optional for everyone else,
  closing off URL tampering and unauthenticated request forgery at `/authorize` for the clients that
  need the strongest guarantee.
- Five client authentication methods are supported, spanning FAPI-grade asymmetric authentication
  down to legacy shared-secret methods, with `client_secret_jwt` deliberately excluded.
- `private_key_jwt` verification is pinned to a fixed rule set — algorithm allowlist, issuer/
  subject/audience checks, bounded assertion lifetime, replay protection — so what Discovery
  advertises is what the server actually enforces.
- Client ID Metadata Documents are supported as a non-persisted, registry-less alternative to
  Dynamic Client Registration for resolving `client_id`s shaped as HTTPS URLs.
- Access tokens are issued as self-contained JWTs by default while refresh tokens stay opaque,
  database-backed references, since the two need different revocation and verification-scaling
  properties.
- Refresh tokens rotate on every use, and presenting an already-rotated token revokes the entire
  token family, uniformly for public and confidential clients.
- DPoP is the default sender-constraint mechanism, with mTLS offered as an option for clients that
  already run client PKI.
- Consent is persisted per `(subject, client_id)` as a set of granted scopes, not per-client and not
  per-interaction, to avoid silent scope creep and consent fatigue.
- Authorization decisions are declared as policy in `spec/policy/client-authorization.json` and
  evaluated through an AuthZEN-style `authorize()` interface rather than scattered as inline
  conditionals.
- The Discovery document is generated at runtime from `spec/discovery.json` rather than
  hand-maintained or build-time generated, so it cannot drift from the implementation.
- The Device Authorization Grant reuses the state-transition table already declared in
  `spec/flows/device-code-flow.json` rather than reimplementing approve/deny/exchange transitions
  ad hoc.
- Protocol timing and security parameters (token/code/PAR TTLs, rate limits, DPoP replay window,
  consent retention) are kept together in one place rather than modeled as product objectives, since
  they are protocol/security settings, not availability SLOs.
- `Agent` is a first-class principal that owns identity and lifecycle but holds no credentials of
  its own, binding instead to existing `OAuth2Client` registrations.
- Acting on a user's behalf is implemented as OAuth 2.0 Token Exchange, defaulting to delegation
  (original `sub`, agent recorded in `act`) rather than impersonation.
- Agent-scoped permissions are expressed as RFC 9396 `authorization_details` rather than coarse
  scopes, so bounds like a transfer limit can be declared and only ever narrowed, never widened, on
  a subsequent token exchange.
- The `sid` claim is `LoginSession.id` itself, shared across every relying party for a browser
  session, so a single session revoke can be walked to every affected RP.
- Human approval of an agent action is implemented as CIBA layered onto OAuth 2.0 as an approval
  capability, displacing neither consent nor step-up.
- The record holding an approval decision is a transport-neutral `ApprovalRequest` keyed by a UUID;
  CIBA lookup and polling fields are explicitly transport bookkeeping but stay co-located so the
  store can serialize decision and poll updates atomically.
- An approved request becomes a token through a store-level compare-and-set, so concurrent polls
  cannot double-issue; every other state is fail-closed at `/token`.
- Only CIBA `poll` delivery is implemented, and `user_code` is advertised as unsupported, since the
  approval screen already sits behind the user's session and step-up.
- Whether a `supervised` agent is *obliged* to obtain approval is left to the governance layer, since
  `supervised` is the default kind and enforcing it here would revoke access from existing agents.

## Scenarios

### REQ-OAUTH2-001: user-bound OAuth grantはaccount scopeのaccess tokenを発行できる
- ACTOR RegisteredClient
- GIVEN client は account:read と account:write を許可 scope として登録している
- GIVEN active User が Authorization Code + PKCE または Device Authorization で account:read に同意している
- WHEN client が user-bound grant を /token で交換する
  - ALT client_credentials または User subject のない token exchange で account scope を要求する → token request は InvalidScopeError で拒否される
  - ALT client の許可 scope または User consent に account scope が含まれない → account scope は発行されない
- THEN access token の sub は同意した User、audience は realm の IdMagic API、scope は account:read になる
- THEN account resource server は token の subject 本人に限って read 操作を許可する

### REQ-OAUTH2-002: API token発行者はaccount consent scope内で自身のconsentだけを操作できる
- ACTOR SelfApiClient
- GIVEN client は対象 tenant の active User に固定された有効な API access token を提示している
- WHEN client が自身の active consent の参照または撤回を要求する
  - ALT account:read だけで consent revoke を要求する → 操作は AccessDeniedError で拒否される
  - ALT token の tenant または user_id が操作対象と一致しない → 操作は AccessDeniedError で拒否される
- THEN account:read scope は自身の active consent の参照だけを許可する
- THEN account:consents:write scope は自身の consent の撤回だけを許可する

### REQ-OAUTH2-003: management API clientはOAuth resourceごとのscope内だけを操作できる
- ACTOR ManagementApiClient
- GIVEN client は対象 tenant の有効な API access token を提示している
- WHEN client が OAuth2 client、authorization detail type、または MCP resource server の操作を要求する
  - ALT oauth-clients:read だけで OAuth2 client の変更を要求する → 操作は AccessDeniedError で拒否される
  - ALT 別 resource の scope で操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT token の tenant と request tenant が一致しない → 操作は AccessDeniedError で拒否される
- THEN oauth-clients:read scope は OAuth2 client の参照だけを許可する
- THEN authorization-detail-types:write scope は authorization detail type の変更だけを許可する
- THEN mcp-resource-servers:read scope は MCP resource server の参照だけを許可する

### REQ-OAUTH2-004: 管理者は自身に可視なrole policyを確認できる
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] の管理者が認証済みである
- WHEN 管理者が role policy 一覧を取得する
  - ALT principal が admin または system_admin ではない → role policy 一覧は AccessDeniedError で拒否される
- THEN 応答には可視な role、permission、対応 HTTP interface が含まれる

### REQ-OAUTH2-005: 認可コードフローでアクセストークンと ID トークンを取得できる
- ACTOR RegisteredClient
- GIVEN "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- GIVEN ユーザー "alice" は "web-app" に scope "openid profile" を同意済みである
- WHEN "web-app" として scope "openid profile" で認可リクエストを送る
  - ALT 認可リクエストの redirect_uri が未登録である → リダイレクトは行われず IdP がエラーページを表示する → エラー "InvalidRequestError"
  - ALT 単一値認可parameterが重複する、promptに重複または未対応tokenがある、またはnoneが他のprompt tokenと併用される → 認可コードは発行されない → 安全に確定済みの登録redirect_uriがあればstateとissuer識別子を含むinvalid_requestを返す → それ以外はリダイレクトせず IdP がエラーページを表示する
  - ALT request_uriと許容されないfront-channel authorization parameterが混在する → 認可コードは発行されない → エラー "InvalidRequestError"
  - ALT promptがnoneで既存セッションまたは必要な同意がない → UIおよびログインリダイレクトは発生しない → 既存セッションがなければstateとissuer識別子を含むlogin_requiredを登録redirect_uriへ返す → 同意がなければstateとissuer識別子を含むconsent_requiredを登録redirect_uriへ返す
- WHEN client が発行された認可コードを正しい PKCE verifier で交換する
  - ALT PKCE verifier が一致しない → 認可コードを誤った code_verifier で交換する → エラー "InvalidGrantError" → トークンは発行されない
  - ALT 同じ認可コードを 2 回交換する → 1 回目の応答には access_token が含まれる → 2 回目はエラー "InvalidGrantError" → 発行ファミリーのトークンがすべて失効する → "RefreshTokenReuseDetected" が発行される → "TokenRevoked" が発行される
  - ALT 認可コードが発行から 60 秒を超えている → 認可コードの交換はエラー "InvalidGrantError" → 認可コードの状態は Expired になる
- THEN 応答に access_token・id_token・refresh_token が含まれ token_type は Bearer
- THEN "UserAuthenticated" が発行される
- THEN "AuthorizationCodeIssued" が発行される
- THEN "AuthorizationCodeRedeemed" が発行される
- THEN "AccessTokenIssued" が発行される
- THEN "RefreshTokenIssued" が発行される

### REQ-OAUTH2-006: リフレッシュトークンをローテーションして新しいトークンを得る
- ACTOR RegisteredClient
- GIVEN 有効な refresh token "RT1" が存在する
- WHEN リフレッシュトークン "RT1" を交換する
  - ALT ローテーション済みの旧 refresh token を再使用する → family_id "F1" の refresh token "RT1" をローテーション後に再使用する → 再使用はエラー "InvalidGrantError" → "RT1" の状態は "Revoked" → family_id "F1" のトークンがすべて失効する → "RefreshTokenReuseDetected" が発行される → "TokenRevoked" が発行される
- THEN 応答に新しい access_token と refresh_token が含まれる
- THEN "RT1" の状態は "Rotated"
- THEN "RefreshTokenRotated" が発行される
- THEN "AccessTokenIssued" が発行される
- THEN "RefreshTokenIssued" が発行される

### REQ-OAUTH2-007: 不正なクライアント認証は invalid_client で一律拒否される
- ACTOR RegisteredClient
- WHEN 既知のクライアントを誤った client_secret で認可コードを交換する
  - ALT 未知の client_id で交換する → 未知の client_id で認可コードを交換する → エラー "InvalidClientError"
- THEN エラー "InvalidClientError"

### REQ-OAUTH2-008: 既存同意の有無に応じて同意画面を出し分ける
- ACTOR ResourceOwner
- GIVEN ユーザー "alice" が "web-app" に scope "openid profile" を同意済みである
- WHEN "web-app" として scope "openid profile" で認可リクエストを送る
  - ALT prompt=consent で再同意を要求する → "web-app" として prompt "consent" で認可リクエストを送る → 認可リクエストの状態は ConsentPending
- THEN 認可リクエストの状態は Consented
- THEN 同意 UI は表示されない

### REQ-OAUTH2-009: PAR で送信した認可リクエストを request_uri 経由で実行する
- ACTOR RegisteredClient
- GIVEN クライアント "web-app" が存在する
- WHEN "web-app" として認可リクエストを事前送信する
  - ALT PAR 必須の FAPI クライアントが PAR なしで直接送信する → PAR 必須の FAPI クライアント "fapi-app" として scope "openid" で直接認可リクエストを送る → エラー "InvalidRequestError"
- THEN PAR 応答に request_uri が含まれ expires_in は 600 以下
- WHEN client が request_uri "<返された値>" で認可リクエストを送る
- THEN その PAR レコードの状態は "Used"
- THEN "PARStored" が発行される
- THEN "AuthorizationCodeIssued" が発行される

### REQ-OAUTH2-010: DPoP 証明付き要求はセンダー制約付きトークンを発行する
- ACTOR RegisteredClient
- WHEN 有効な DPoP 証明を付けて認可コードを交換する
  - ALT iat が 60 秒以上古い DPoP 証明である → iat "2026-01-01T00:00:00Z" の DPoP 証明を時刻 "2026-01-01T00:01:30Z" で付けて認可コードを交換する → エラー "InvalidDpopProofError"
  - ALT 同一 DPoP jti を再使用する → jti "ABC" の DPoP 証明を付けて認可コードを交換する → 同じ jti "ABC" の DPoP 証明を付けて認可コードを交換する → 1 回目の応答には access_token が含まれる → 2 回目はエラー "InvalidDpopProofError"
- THEN 発行された access token は DPoP 鍵サムプリントに cnf でバインドされる
- THEN 発行された refresh token のセンダー制約は Dpop

### REQ-OAUTH2-011: 失効済みトークンの introspection は active=false のみ返す
- ACTOR ResourceServer
- GIVEN 失効済み access token "AT1" が存在する
- WHEN トークン "AT1" を検査する
- THEN 応答は active=false のみで他のフィールドを含まない

### REQ-OAUTH2-012: kill-switch後のAgentトークンはintrospectionでactive=falseになる
- ACTOR ResourceServer
- GIVEN Agent "A1" に issued_at が古い access token "AT1" が発行済みである
- GIVEN "A1" は kill-switch により revocation epoch が "AT1" の issued_at より後へ前進している
- WHEN トークン "AT1" を検査する
  - ALT "AT1" が revocation epoch より後に発行された (kill 後に再発行された) token である → 応答は通常どおり active=true と claim を返す
- THEN 応答は active=false のみで他のフィールドを含まない

### REQ-OAUTH2-013: UserInfo は openid スコープのトークンに sub を返す
- ACTOR RegisteredClient
- GIVEN scope "openid profile" の access token "AT1" が存在する
- WHEN トークン "AT1" でユーザー情報を取得する
  - ALT openid スコープを持たないトークンで取得する → scope "profile" のみの access token "AT1" でユーザー情報を取得する → エラー "InsufficientScopeError"
- THEN 応答に sub・name・preferred_username が含まれる

### REQ-OAUTH2-014: Discovery 文書は宣言された全エンドポイントを広告する
- ACTOR RegisteredClient
- WHEN Discovery 文書を取得する
- THEN 応答に issuer・authorization_endpoint・token_endpoint・userinfo_endpoint・jwks_uri・introspection_endpoint・revocation_endpoint・pushed_authorization_request_endpoint・device_authorization_endpoint・backchannel_authentication_endpoint・registration_endpoint が含まれる

### REQ-OAUTH2-015: 認可コードの並行交換はちょうど一方だけ成功する
- ACTOR RegisteredClient
- GIVEN 発行済み認可コード "AC1"（family_id "F1"）が存在する
- WHEN 認可コード "AC1" を verifier "v" で並行に 2 回交換する
- THEN ちょうど一方が成功し、もう一方はエラー "InvalidGrantError"
- THEN family_id "F1" のトークンがすべて失効する
- THEN "AuthorizationCodeRedeemed" が発行される
- THEN "RefreshTokenReuseDetected" が発行される
- THEN "TokenRevoked" が発行される

### REQ-OAUTH2-016: 動的クライアント登録は client_id を採番して返す
- ACTOR Client
- WHEN confidential クライアント "web-app" を redirect_uri "https://app.example.com/callback" で登録する
  - ALT redirect_uri を持たない登録要求である → confidential クライアント "web-app" を redirect_uri "" で登録する → エラー "InvalidRequestError"
- THEN 応答に client_id と client_secret が含まれる
- THEN "ClientRegistered" が発行される

### REQ-OAUTH2-017: Client metadata fetchは公開IPへ直接接続する
- ACTOR RegisteredClient
- GIVEN client_id は public IP に解決される client 所有の HTTPS metadata URL である
- GIVEN Authorization Server の環境に HTTPS proxy が設定されている
- WHEN Authorization Server が client metadata URL を取得する
  - ALT metadata host が private、loopback、link-local、または CGNAT 100.64.0.0/10 の IP に解決される → Authorization Server は対象 IP へ接続しない → client metadata の解決を fail-closed で拒否する
- THEN 環境 proxy を使用せず、DNS 検査済みの public IP へ直接接続する
- THEN metadata document の取得と検証に成功する

### REQ-OAUTH2-018: absolute_expires_at を超えた refresh token はローテーション不可
- ACTOR RegisteredClient
- GIVEN absolute_expires_at "2026-01-01T00:00:00Z" の refresh token "RT1" が存在する
- GIVEN 現在時刻は "2026-01-02T00:00:00Z" である
- WHEN client がリフレッシュトークン "RT1" を交換する
- THEN エラー "InvalidGrantError"

### REQ-OAUTH2-019: クライアントは自分のトークンを失効できる
- ACTOR RegisteredClient
- GIVEN 有効な refresh token "RT1" が存在する
- WHEN トークン "RT1" を失効させる
  - ALT 所有者でないクライアントが失効を要求する → クライアント "client-A" が所有する refresh token "RT1" に対し "client-B" として失効を要求する → 盗難検知防止のため 200 OK のみ返り "RT1" の状態は "Active" のまま
- THEN "RT1" の状態は "Revoked"
- THEN "TokenRevoked" が発行される

### REQ-OAUTH2-020: 失効した access_token でユーザー情報取得は invalid_token で拒否される
- ACTOR RegisteredClient
- GIVEN 有効な access token "AT1" が存在する
- WHEN トークン "AT1" を失効させる
- THEN トークン "AT1" は失効状態になる
- WHEN client がトークン "AT1" でユーザー情報を取得する
- THEN エラー "InvalidTokenError"

### REQ-OAUTH2-021: refresh_token は offline_access スコープ付与時のみ発行される
- ACTOR RegisteredClient
- GIVEN confidential クライアント "web-app" が grant_types に "authorization_code"・"refresh_token" を含めて登録済みである
- WHEN "web-app" として scope "openid offline_access" で認可リクエストを送る
  - ALT offline_access を要求しない → "web-app" として scope "openid profile" で認可リクエストを送る → 発行された認可コードを verifier "v" で交換する → 応答に refresh_token は含まれない
- WHEN client が発行された認可コードを verifier "v" で交換する
- THEN 応答に refresh_token が含まれる
- THEN "RefreshTokenIssued" が発行される

### REQ-OAUTH2-022: 認可リクエストの nonce は ID トークンに伝播する
- ACTOR RegisteredClient
- WHEN "web-app" として scope "openid"、nonce "n-12345" で認可リクエストを送る
- WHEN client が発行された認可コードを verifier "v" で交換する
- THEN 応答の id_token の nonce クレームは "n-12345"

### REQ-OAUTH2-023: RP-Initiated Logout は登録済み post_logout_redirect_uri にだけ戻す
- ACTOR ResourceOwner
- GIVEN confidential クライアント "web-app" が redirect_uri "https://app.example.com/cb" で登録済みである
- WHEN "web-app" として post_logout_redirect_uri "https://app.example.com/cb" でログアウトする
  - ALT 未登録の post_logout_redirect_uri を指定する → "web-app" として post_logout_redirect_uri "https://evil.example.com/cb" でログアウトする → エラー "InvalidRequestError"
- THEN state が post_logout_redirect_uri に伝播する

### REQ-OAUTH2-024: RP-Initiated Logout はid_token_hintからsessionとclientを解決する
- ACTOR ResourceOwner
- GIVEN ユーザー "alice" が "web-app" として認可コードを交換し sid 付きの ID Token を持つ
- WHEN "alice" が発行済み ID Token を id_token_hint として /end_session を呼ぶ
  - ALT id_token_hint の aud が指定 client_id と一致しない → client_id "other-app" と "web-app" 発行の ID Token を id_token_hint に付けて /end_session を呼ぶ → エラー "InvalidRequestError"
  - ALT id_token_hint の署名がidmagicの署名鍵で検証できない → 他 issuer が署名した JWT を id_token_hint に付けて /end_session を呼ぶ → エラー "InvalidRequestError"
  - ALT id_token_hint が期限切れ (exp 経過) である → 期限切れの発行済み ID Token を id_token_hint として /end_session を呼ぶ → exp 切れのみを理由にした拒否はされず sid による session 解決が成功する
- THEN id_token_hint の sid が示す LoginSession が失効する
- THEN 同じ sid を持つ全 client の RefreshTokenRecord が Revoked へ遷移する

### REQ-OAUTH2-025: session revokeはbackchannel_logout_uri登録済みRPへlogout tokenを配送する
- ACTOR ResourceOwner
- GIVEN "web-app" が backchannel_logout_uri "https://app.example.com/backchannel_logout" を登録済みである
- GIVEN ユーザー "alice" が "web-app" とのブラウザ session を持つ
- WHEN "alice" が /end_session でログアウトする
- THEN 対象 sid と "web-app" の LogoutNotification が作成される
- THEN 署名済み logout token が backchannel_logout_uri へ配送され Delivered になる
  - ALT 配送が一時的に失敗する (5xx / timeout) → LogoutNotification は Pending のまま attempts が増え再試行され、ローカルの session/refresh token 失効は取り消されない
  - ALT max_attempts まで再試行しても配送が成功しない → LogoutNotification は Failed (dead-letter) に確定し、ローカルの session/refresh token 失効は取り消されない

### REQ-OAUTH2-026: client_credentials グラントで M2M トークンが発行される
- ACTOR RegisteredClient
- GIVEN confidential クライアント "backend" が grant_types に "client_credentials" を含めて登録済みである
- WHEN "backend" として client_credentials で scope "api:read" のトークンを取得する
- THEN 応答に access_token が含まれ refresh_token は含まれない
- THEN 発行された access_token の sub は client_id と一致する
- THEN "AccessTokenIssued" が発行される
- WHEN public クライアント "spa-app" を grant_types に "client_credentials" を含めて登録する
- THEN client_credentials は confidential 限定であるため InvalidRequestError で拒否される

### REQ-OAUTH2-027: デバイス認可フローでアクセストークンを取得できる
- ACTOR RegisteredClient
- GIVEN confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- WHEN "tv-app" として scope "openid profile" でデバイス認可を開始する
- THEN 応答に device_code・user_code・verification_uri・interval が含まれる
- WHEN ユーザー "alice" が verification_uri で user_code を入力し承認する
- THEN device authorization は承認済みになる
- WHEN client が device_code "DC1" を交換する
  - ALT ユーザー承認前にポーリングする → Issued 状態の device_code "DC1" を交換する → エラー "AuthorizationPendingError"
  - ALT ポーリング間隔より短い再試行をする → interval 5 秒の device_code "DC1" を Issued 状態で用意する → device_code "DC1" を交換し "2s" 経過後に再度交換する → 2 回目はエラー "SlowDownError"
  - ALT device_code が expires_in を超えている → issued_at "2026-01-01T00:00:00Z"・expires_at "2026-01-01T00:10:00Z" の device_code "DC1" を時刻 "2026-01-01T00:11:00Z" で交換する → エラー "ExpiredTokenError"
- THEN 応答に access_token と id_token が含まれる
- THEN "DeviceAuthorizationRequested" が発行される
- THEN "DeviceAuthorizationApproved" が発行される
- THEN "AccessTokenIssued" が発行される

### REQ-OAUTH2-028: 改ざんされた client_assertion は invalid_client で拒否される
- ACTOR RegisteredClient
- GIVEN confidential クライアント "fapi-app" が token_endpoint_auth_method "private_key_jwt"・jwks 登録で存在する
- WHEN 認可コード "AC1" を verifier "v"・改ざんされた client_assertion で交換する
- THEN エラー "InvalidClientError"

### REQ-OAUTH2-029: mTLS バインド AT は同じ証明書のリクエストでのみ受理される
- ACTOR RegisteredClient
- GIVEN confidential クライアント "mtls-app" が token_endpoint_auth_method "tls_client_auth" で存在する
- WHEN client が mTLS 証明書を提示して access_token を要求する
- THEN access_token は提示された証明書にバインドされる
- WHEN client が証明書にバインドされた access_token で userinfo を取得する
  - ALT 別の証明書を提示する → invalid_token で拒否される
- THEN 同じ証明書を提示した要求は 200 を返す

### REQ-OAUTH2-030: RFC 8414 メタデータ文書は OIDC Discovery と同等の内容を返す
- ACTOR RegisteredClient
- WHEN Authorization Server メタデータを取得する
- THEN 応答に issuer・authorization_endpoint・token_endpoint・jwks_uri・grant_types_supported が含まれる

### REQ-OAUTH2-031: 管理者は所属テナントの同意を参照・撤回できるが付与は代行できない
- ACTOR TenantAdministrator
- GIVEN tenant_id "acme" の roles=["admin"] のユーザー "operator" が認証済みである
- GIVEN tenant_id "acme" のユーザー "alice" と client "portal" の Consent が Granted で存在する
- WHEN 管理者 "operator" が Consent 一覧と単一 Consent を取得する
- THEN 所属テナントの Consent だけが返る
- WHEN 管理者 "operator" がユーザー "alice" と client "portal" の Consent を撤回する
- THEN Consent の state は Revoked で revoked_at が記録される
- THEN "ConsentRevoked" が actorUserId "operator" で発行される
- THEN 管理者が Consent を作成または scope 拡張する interface は存在しない

### REQ-OAUTH2-032: ユーザーは接続済みアプリの同意を自分で撤回できる
- ACTOR ResourceOwner
- GIVEN ユーザー "alice" が client "web-app" に scope "openid profile" を同意済みである
- GIVEN ユーザー "alice" が認証済みで接続済みアプリ画面を開いている
- WHEN ユーザー "alice" が接続済みアプリ一覧を取得する
- THEN 一覧に "web-app" が表示される
- WHEN ユーザー "alice" が "web-app" の同意を撤回する
- THEN Consent の state は Revoked になり一覧から消える

### REQ-OAUTH2-033: realm prefix 付き Discovery は同 prefix の issuer を返す
- ACTOR RegisteredClient
- GIVEN tenant_id "acme" が Active で存在する
- WHEN /realms/acme/.well-known/openid-configuration を取得する
- THEN 応答の issuer は base URL + /realms/acme
- THEN 応答の authorization_endpoint は base URL + /realms/acme/authorize

### REQ-OAUTH2-034: トークンエンドポイントはテナント境界を越えた資格情報を受理しない
- ACTOR RegisteredClient
- WHEN tenant_id "acme" で発行した認可コード "AC1" を "/realms/default/token" で交換する
  - ALT 他テナントの client_id を使う → tenant_id "acme" に登録した client_id "web-app" で "/realms/default/token" に交換を要求する → エラー "InvalidClientError"
  - ALT 他テナントのリフレッシュトークンを再発行する → tenant_id "acme" で発行した refresh token "RT1" を "/realms/default/token" で再発行する → エラー "InvalidGrantError"
  - ALT 他テナントの device_code を交換する → tenant_id "acme" で発行し承認した device_code "DC1" を "/realms/default/token" で交換する → エラー "InvalidGrantError"
- THEN エラー "InvalidGrantError"
- WHEN 保存層に tenant_id "acme" の refresh token と tenant_id "default" の client_id または sub を書き込む
- THEN 永続化層が参照整合性エラーで拒否する

### REQ-OAUTH2-035: 管理者は所属テナントのclientを作成・更新・削除できる
- ACTOR TenantAdministrator
- GIVEN tenant_id "acme" の admin "operator" が認証済みである
- WHEN "operator" が client "portal" を作成する
- THEN client_secret が一度だけ返る
- WHEN "operator" が client "portal" を取得する
  - ALT 別テナントの管理者が同じ client_id を指定する → InvalidRequestError で拒否される
- THEN 所属テナントの client だけが返る
- WHEN "operator" が client "portal" の redirect_uris を更新する
- THEN redirect_uris が保存される
- WHEN "operator" が client "portal" を削除する
- THEN "AdminOAuth2ClientCreated"、"AdminOAuth2ClientUpdated"、"AdminOAuth2ClientDeleted" が発行される

### REQ-OAUTH2-036: 管理者はApplicationから期限付きclient secretを追加発行して個別失効できる
- ACTOR TenantAdministrator
- GIVEN tenant_id "acme" の Application "billing" は client_secret_basic の confidential OIDC client を binding として持つ
- GIVEN 期限なし legacy secret "S1" が Active である
- WHEN 管理者が expires_in_days 90 で新 secret を追加発行する
  - ALT expires_in_days が1..730の範囲外である → エラー "InvalidRequestError"
  - ALT Active credential が既に2件存在する → 追加発行はエラー "ClientSecretLimitExceededError" で拒否され、既存 credential は変更されない
  - ALT client が private_key_jwt、mTLS、または public client である → エラー "InvalidRequestError"
- THEN 応答で新 secret を一度だけ受け取り、metadata は90日後の expires_at と Active 状態を持つ
- THEN 追加発行は既存 secret の期限と状態を変更しない
- THEN 新 secret と旧 secret の両方が token endpoint 認証へ成功する
- WHEN 管理者が旧 credential だけを個別失効する
  - ALT 別 client の credential_id または存在しない credential_id を失効する → エラー "InvalidRequestError"
  - ALT 既に Revoked の credential を再び失効する → 冪等に成功し、ClientSecretRevoked は重複発行されない
- THEN 旧 secret は InvalidClientError で拒否され、新 secret は引き続き認証へ成功する
- THEN ClientSecretIssued と ClientSecretRevoked が actor、client、credential、expiry の非機密 metadata だけを含んで発行される

### REQ-OAUTH2-037: 管理者は互換interfaceでclient secretを無停止rotationできる
- ACTOR TenantAdministrator
- GIVEN tenant_id "acme" の Application "billing" は client_secret_basic の confidential OIDC client を binding として持つ
- GIVEN 旧 secret "S1" が有効である
- WHEN 管理者が grace_days 7 で secret を rotation する
  - ALT grace_days が 0 以外で 1..30 の範囲外である → エラー "InvalidRequestError"
  - ALT client が private_key_jwt、mTLS、または public client である → エラー "InvalidRequestError"
- THEN 応答で新 secret を一度だけ受け取る
- THEN 新 secret と旧 secret は grace_until より前に token endpoint 認証へ成功する
- THEN grace_until より後は旧 secret が InvalidClientError で拒否される
- THEN ClientSecretRotated が actor、client、grace_until だけを含んで発行される

### REQ-OAUTH2-038: 管理consent APIは別テナントの同意を公開しない
- ACTOR TenantAdministrator
- GIVEN tenant_id "acme" の user と client の Consent が存在する
- WHEN tenant_id "default" の管理者が同じ user_id と client_id の Consent を取得する
- THEN エラー "InvalidRequestError"

### REQ-OAUTH2-039: KeyProvider障害時は新規token発行を拒否する
- ACTOR RegisteredClient
- GIVEN tenant_id "acme" の KeyProvider が到達不能である
- WHEN tenant_id "acme" の client が token 発行を要求する
- THEN 新規署名は行われずエラー "ServerError" で拒否される

### REQ-OAUTH2-040: protocol endpointは閾値超過リクエストをrate limitで拒否する
- ACTOR RegisteredClient
- GIVEN client がある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している
- WHEN 同一 window 内で追加リクエストを送る
  - ALT 対象 endpoint が /token である → client_id と IP の組で閾値超過している状態で token を要求する → エラー "RateLimitedError"
  - ALT 対象 endpoint が /authorize または /par である → IP と client_id の組で閾値超過している状態で認可リクエストを送る → エラー "RateLimitedError"
  - ALT 対象 endpoint が /device_authorization である → client_id と IP の組で閾値超過している状態でデバイス認可を開始する → エラー "RateLimitedError"
  - ALT 対象 endpoint が /bc-authorize である → client_id と IP の組で閾値超過している状態で backchannel 認可を開始する → エラー "RateLimitedError"
  - ALT 共有カウンタストアに到達できない → リクエストは fail-closed で "RateLimitedError" として拒否される
- THEN エラー "RateLimitedError" (HTTP 429、Retry-After ヘッダ付き)

### REQ-OAUTH2-041: backchannel 認可要求は人間の承認が成立してからトークンを発行する
- ACTOR RegisteredClient
- GIVEN confidential クライアント "agent-app" が grant_types に "urn:openid:params:grant-type:ciba" を含めて登録済みである
- GIVEN active User "alice" が存在する
- WHEN "agent-app" として login_hint "alice"・scope "openid"・binding_message "W-123" で backchannel 認可を開始する
  - ALT scope が未指定または openid を含まない → エラー "InvalidScopeError"
  - ALT login_hint と id_token_hint が両方指定される、または両方未指定である → エラー "InvalidRequestError"
  - ALT requested_expiry が非正または 600 秒を超える → エラー "InvalidRequestError"
  - ALT binding_message が 64 文字を超える、または制御文字を含む → エラー "InvalidBindingMessageError"
  - ALT login_hint が User を解決できない → 未知の login_hint "nobody" で backchannel 認可を開始する → エラー "UnknownUserIdError"
  - ALT login_hint の User が別テナントまたは非 active である → エラー "UnknownUserIdError"
  - ALT client の許可 scope に含まれない scope を要求する → エラー "InvalidScopeError"
- THEN 応答に auth_req_id・expires_in・interval が含まれる
- THEN 承認要求の状態は Pending になる
- THEN "BackchannelAuthRequested" が発行される
- WHEN "agent-app" が auth_req_id "AR1" を交換する
  - ALT ユーザー判断前にポーリングする → Pending 状態の auth_req_id "AR1" を交換する → エラー "AuthorizationPendingError"
  - ALT interval より短い間隔で再試行する → interval 5 秒の auth_req_id "AR1" を交換し "2s" 経過後に再度交換する → 2 回目はエラー "SlowDownError"
- WHEN ユーザー "alice" が承認要求 "AR1" を承認する
- THEN 承認要求 "AR1" の状態は Approved になる
- THEN "BackchannelAuthApproved" が発行される
- WHEN "agent-app" が auth_req_id "AR1" を交換する
- THEN 応答に access_token と id_token が含まれ sub は "alice"、scope は要求した "openid" になる
- THEN 承認要求 "AR1" の状態は Consumed になる
- THEN "AccessTokenIssued" が発行される

### REQ-OAUTH2-042: 承認が成立していない承認要求はトークンを発行しない
- ACTOR RegisteredClient
- GIVEN "agent-app" が起票した承認要求 "AR1" が存在する
- WHEN "agent-app" が auth_req_id "AR1" を交換する
  - ALT ユーザーが "AR1" を拒否済みである → エラー "AccessDeniedError" → 承認要求 "AR1" の状態は Denied のままになる
  - ALT "AR1" が expires_at を過ぎている → requested_at "2026-01-01T00:00:00Z"・expires_at "2026-01-01T00:05:00Z" の "AR1" を時刻 "2026-01-01T00:06:00Z" で交換する → エラー "ExpiredTokenError" → 承認要求 "AR1" の状態は Expired になる
  - ALT 承認済みの "AR1" を 2 回交換する → 1 回目の応答には access_token が含まれる → 2 回目はエラー "InvalidGrantError" → 承認要求 "AR1" の状態は Consumed のままになる
  - ALT 承認済みの "AR1" を並行に 2 回交換する → ちょうど一方が成功し、もう一方はエラー "InvalidGrantError"
  - ALT 起票元でない client "other-app" が "AR1" を交換する → エラー "InvalidGrantError"
  - ALT 別テナントの token endpoint で "AR1" を交換する → エラー "InvalidGrantError"
  - ALT "AR1" の承認後に Agent が kill-switch で停止されている → エラー "InvalidGrantError" → トークンは発行されない
- THEN 承認要求が Approved のときだけ応答に access_token が含まれる

### REQ-OAUTH2-043: 承認要求を判断できるのは対象ユーザー本人の step-up 済みセッションだけである
- ACTOR ResourceOwner
- GIVEN ユーザー "alice" 宛の承認要求 "AR1" が Pending で存在する
- GIVEN ユーザー "bob" 宛の承認要求 "AR2" が Pending で存在する
- GIVEN "alice" が認証済みで step-up の recency 窓を満たしている
- WHEN "alice" が保留中の承認要求一覧を取得する
- THEN 一覧には "AR1" と要求元 client の表示名・Agent 名・要求 scope・authorization_details・binding_message が含まれる
- THEN 一覧に "AR2" と期限切れの承認要求は含まれない
- WHEN "alice" が承認要求 "AR1" を承認する
  - ALT step-up の recency 窓を外れている → 操作は AccessDeniedError で拒否され承認要求 "AR1" の状態は Pending のままになる
  - ALT CSRF token が一致しない → 操作は拒否され承認要求 "AR1" の状態は Pending のままになる
  - ALT "alice" が他人宛の承認要求 "AR2" を判断する → 操作は AccessDeniedError で拒否される
  - ALT 既に終端状態の承認要求を判断する → 操作は InvalidRequestError で拒否され、記録済みの判断は上書きされない
- THEN 承認要求 "AR1" の状態は Approved になる
- THEN "BackchannelAuthApproved" が発行される
