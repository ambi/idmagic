---
context: oauth2
updated_at: 2026-08-15
---

# OAuth2 Specification

## Overview

OAuth 2.0 / OIDC プロトコル群の全責務を所有する。クライアントメタデータと Dynamic Client Registration、認可判断（認可、同意、認可コード、PAR、Device Authorization、RP-Initiated Logout）、トークンの発行とライフサイクル（アクセストークン、リフレッシュトークン、ID トークン、イントロスペクション、失効、UserInfo、Proof of Possession）、Discovery Metadata、Authorization Server Metadata、健全性報告をこの Bounded Context に集約する。

認可サーバーは `authorization/`、`client/`、`consent/`、`device/`、`token/` の機能単位で実装する。本書は仕組みごとに、認可とデバイスのライフサイクル、PKCE と PAR、クライアント認証、トークン形式とローテーション、送信者制約、同意、認可ポリシー、Discovery Metadata、デバイスグラント、有効期間とセキュリティ設定、Agent プリンシパルと委譲、Rich Authorization Requests、セッションとログアウトのバインディングの順に読む。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| ResourceOwner | 保護リソースに対するアクセス権限を付与できる主体。本アプリではエンドユーザー。 | resource_owner, リソースオーナー, EndUser |
| Client | ResourceOwner に代わって保護リソースへのアクセスを要求するアプリケーション。`client_id` で識別される。`client_id` が Client ID Metadata Document (CIMD) の URL 形式である場合は、永続的な登録を使わず、認可リクエストごとに文書を取得して解決する。 | クライアント, RelyingParty |
| AuthorizationServer | ResourceOwner を認証し、クライアントへトークンを発行するサーバー。本アプリそのもの。OIDC では IdP と同義。 | authorization_server, IdP, IdentityProvider |
| ResourceServer | アクセストークンを提示したクライアントへ保護リソースを返す、本アプリ外部のサーバー。 | resource_server |
| McpResourceServer | MCP エコシステムでツールやデータソースを提供するリソースサーバーのテナント単位の登録。正規リソース URI と許可するスコープを持ち、Protected Resource Metadata とリソース指示子の検証基準になる。 | mcp_resource_server |
| ResourceIndicator | RFC 8707 の `resource` パラメーター。要求する `audience` を表す、フラグメントを含まない絶対 URI。1 トークンにつき 1 個だけを許可し、複数指定、未登録、無効のいずれかであれば fail-closed で拒否する。 | resource_indicator |
| Public | `client_secret` を安全に保持できないクライアント（SPA、ネイティブアプリ）。PKCE を必須とする。 | public |
| Confidential | サーバー側で `client_secret` または秘密鍵を保持できるクライアント。 | confidential |
| AuthorizationCode | 短命（60秒以下）の一度限り使用される中間トークン。/authorize で発行され /tokenで交換される。grant_type 値「authorization_code」と同じ文字列を共有する（RFC 6749 §4.1）。 | authorization_code, 認可コード |
| RefreshToken | AccessToken の再発行に使うトークン。提示のたびにローテーションされる。絶対 TTL 30 日。grant_type 値「refresh_token」と同じ文字列を共有する（RFC 6749 §6）。 | refresh_token, リフレッシュトークン |
| ClientCredentials | クライアント自身が認可主体となる M2M フロー（RFC 6749 §4.4）。`confidential` クライアントだけが利用できる。 | client_credentials, M2M グラント |
| DeviceCode | 入力制約のあるデバイス向けのデバイス認可グラント (RFC 8628)。grant_type は URN。 | urn:ietf:params:oauth:grant-type:device_code, device_code, デバイスコード |
| TokenExchange | 既存のトークンを別のトークンへ交換するグラント（RFC 8693）。本アプリは 2 種類の `subject_token` を扱う。（1）自己発行トークンの委任では、`subject_token` と `actor_token` を本 IdP が発行し、`IntrospectAccessToken` を通過したものに限定する。（2）ワークロードアイデンティティ連携では、`subject_token_type` が `JwtSvid` のときに外部のアテステーショントークンを WorkloadIdentity の `VerifyWorkloadAttestation` で検証し、テナントに登録された `AgentWorkloadBinding` に対応する Agent の資格情報として発行する。`grant_type` は URN。 | urn:ietf:params:oauth:grant-type:token-exchange, token-exchange, トークン交換 |
| Ciba | Client-Initiated Backchannel Authentication（OpenID CIBA Core）。クライアントが帯域外の認証デバイスを介して ResourceOwner の判断を求める、分離型のフロー。本アプリはポーリングモードだけを実装し、`grant_type` には URN を使う。 | urn:openid:params:grant-type:ciba, ciba, CIBA, backchannel authentication |
| ApprovalRequest | 人間が承認するまでクライアントの要求を保留する承認要求。承認対象の User、要求元のクライアントまたは Agent、要求するスコープと `AuthorizationDetails`、バインディングメッセージ、期限を持ち、`Pending` から一方向に遷移する。CIBA の転送方式に固有の語彙に依存しない一般形で保持する。 | approval_request, 承認要求 |
| AuthReqId | `/bc-authorize` が発行する承認要求のベアラーシークレット（CIBA Core §7.3）。トークンエンドポイントの CIBA グラントで提示する。保存するのは SHA-256 ハッシュだけとし、画面にも監査ログにも出さない。 | auth_req_id |
| BindingMessage | 承認画面に表示する短い識別文 (CIBA Core §7.1)。別要求の取り違え承認を防ぐ補助であり、要求内容の提示を代替しない。 | binding_message |
| TokenDeliveryMode | 承認成立をどうクライアントへ届けるかの区分 (CIBA Core §4)。本アプリは poll のみ実装し、ping / push は広告しない。 | backchannel_token_delivery_mode, poll, ping, push |
| UnknownUserId | CIBA エラーコード `unknown_user_id`。`login_hint` または `id_token_hint` から承認対象の User を解決できないことを表す。存在の有無を開示しないため、非アクティブなユーザーや別テナントのユーザーも同じエラーとして扱う。 | unknown_user_id |
| Pending | 承認要求が起票され、まだ判断されていない初期状態。 | pending |
| Consumed | 承認済みの承認要求がちょうど一度トークン化された終端状態。 | consumed |
| Consume | 承認済みの承認要求をトークンへ一度きり消費する。 | consume |
| AuthorizationDetails | 構造化された細粒度の権限要求 (RFC 9396)。type で識別される JSON オブジェクトの配列として、対象・操作・上限・条件を表し、/authorize・/par・/tokenで要求・同意・トークン反映する。本アプリは受理する type をテナント登録スキーマに限定し fail-closed に検証する。 | authorization_details, RAR, Rich Authorization Requests, リッチ認可リクエスト |
| AccessToken | ResourceServer にアクセスする際に提示するトークン。JWT (PS256 / ES256) として発行、TTL 600秒。 | access_token, アクセストークン |
| IdToken | OIDC が定める、ResourceOwner の認証結果を表明する JWT。iss/sub/aud/exp/iat/auth_time/nonce/azp を含む。 | id_token, IDトークン |
| Pkce | Proof Key for Code Exchange（RFC 7636）。`code_challenge` と `code_verifier` で認可コード横取り攻撃を防ぐ。本アプリでは例外なく必須。 | pkce, PKCE |
| Dpop | Demonstrating Proof of Possession (RFC 9449)。JWK サムプリント (jkt) でトークンを所有鍵にバインドする。 | dpop, DPoP |
| Mtls | Mutual-TLS Client Authentication（RFC 8705）。TLS クライアント証明書でクライアントを認証する。 | mtls, mTLS, tls_client_auth |
| Par | Pushed Authorization Requests (RFC 9126)。/par に認可リクエストを事前 POST し /authorize は request_uri のみで参照する。FAPI 2.0 必須。 | par, PAR |
| SenderConstrainedToken | 所有証明（DPoP または mTLS）と組み合わさったトークン。所有者以外による再利用を防ぐ。 |  |
| Issuer | トークンを発行した認可サーバーの URL 識別子。 | iss |
| Subject | トークンの主体（通常は ResourceOwner）の仮名化された識別子。削除後も監査ログに残る。 | sub |
| Audience | トークンの想定受信者を表す `audience`。通常はクライアントの `client_id`。 | aud |
| JwtId | JWT 一意識別子。リプレイ防止と監査の追跡に使用。 | jti |
| Nonce | クライアントが認可リクエストに含め、ID トークンに含めて返される値。ID トークンのリプレイを防ぐ。 |  |
| Scope | クライアントが要求する権限の集合。クライアントメタデータで宣言した集合の部分集合でなければならない。 |  |
| Consent | ResourceOwner がクライアントへ特定のスコープを付与する意思表示。テナント、`subject`、クライアント、`scopes`、`granted_at`、`expires_at` を永続化する。管理者は参照と撤回だけが可能であり、付与やスコープの拡張を代行できない。 | 同意 |
| AdminConsentManagement | 管理者が所属テナント内の Consent を監査目的で参照し、必要に応じて撤回する管理操作。同意の作成やスコープの拡張は含まない。 | 同意管理 |
| PreferredUsername | 表示用のユーザー名。可変。 | preferred_username |
| ClientSecretBasic | HTTP Basic 認証（レガシークライアント向け）。 | client_secret_basic |
| ClientSecretPost | フォームボディ内のクレデンシャル。 | client_secret_post |
| PrivateKeyJwt | RFC 7523 — client_assertion による署名済み JWT 認証。confidential 推奨。 | private_key_jwt |
| TlsClientAuth | RFC 8705 §2 の Mutual-TLS Client Authentication（PKI 結合）。mTLS 機構の一形態を `token_endpoint_auth_method` の値として表現する。 | tls_client_auth |
| None | クライアント認証なし（public クライアント、PKCE が代替防御）。 | none |
| PS256 | RSASSA-PSS using SHA-256 (RFC 7518)。 |  |
| ES256 | ECDSA using P-256 and SHA-256 (RFC 7518)。 |  |
| S256 | PKCE code_challenge_method=S256。code_challenge は BASE64URL-ENCODE(SHA256(code_verifier))。 | S256 |
| Hwk | RFC 8176 の hardware-secured key 認証メソッド。 | hwk |
| Swk | RFC 8176 の software-secured key 認証メソッド。 | swk |
| Code | `response_type=code`。本アプリが対応する唯一のレスポンスタイプ。 | code |
| Query | 認可レスポンスのパラメーターを `redirect_uri` のクエリ部分で返す `response_mode`。 | query |
| FormPost | 認可レスポンスのパラメーターを自動送信する HTML フォームで返す `response_mode`。 | form_post |
| InvalidRequest | OAuth エラーコード `invalid_request`。必須パラメーターの欠落、不正値、重複などのリクエスト不備。 | invalid_request |
| InvalidClient | OAuth エラーコード `invalid_client`。クライアント認証の失敗。 | invalid_client |
| InvalidGrant | OAuth エラーコード `invalid_grant`。認可コード、リフレッシュトークン、`device_code` などのグラントが無効。 | invalid_grant |
| UnauthorizedClient | OAuth エラーコード `unauthorized_client`。クライアントが当該 `grant_type` を使用できない。 | unauthorized_client |
| UnsupportedGrantType | OAuth エラーコード `unsupported_grant_type`。未対応の `grant_type`。 | unsupported_grant_type |
| InvalidScope | OAuth エラーコード `invalid_scope`。リクエストされたスコープが不正、未知、または許可外。 | invalid_scope |
| InvalidToken | Bearer トークンのエラーコード `invalid_token`。トークンが無効、期限切れ、改ざん、または失効済み。 | invalid_token |
| InvalidDpopProof | DPoP エラーコード `invalid_dpop_proof`。DPoP proof JWT の検証失敗。 | invalid_dpop_proof |
| AccessDenied | OAuth エラーコード `access_denied`。ResourceOwner またはポリシーによる拒否。 | access_denied |
| ExpiredToken | Device Authorization Grant のエラーコード `expired_token`。`device_code` が期限切れ。 | expired_token |
| InsufficientScope | Bearer トークンのエラーコード `insufficient_scope`。提示されたトークンのスコープが不足。 | insufficient_scope |
| ServerError | OAuth エラーコード `server_error`。認可サーバー側の予期しない内部エラー。 | server_error |
| InvalidTarget | RFC 8707 のエラーコード `invalid_target`。`resource` パラメーターが未登録、無効、複数指定のいずれかである場合、または `McpResourceServer` が `Disabled` である場合に返す。 | invalid_target |
| Fapi2SecurityProfile | FAPI 2.0 Security Profiletls_client_auth または private_key_jwt + PAR 必須。 | fapi_2_security_profile |
| Received | 認可リクエストを受信した直後の初期状態。 | received |
| AuthenticationPending | ResourceOwner の認証を待っている状態。 | authentication_pending |
| Authenticated | ResourceOwner の認証が完了した状態。 | authenticated |
| ConsentPending | ResourceOwner に Scope への同意を求めている状態。 | consent_pending |
| Consented | ResourceOwner が要求 Scope に同意した状態。 | consented |
| CodeIssued | `AuthorizationCode` をクライアントへ発行した状態。 | code_issued |
| Exchanged | AuthorizationCode が /tokenで正常に交換された終端状態。 | exchanged |
| Rejected | 認可リクエストが拒否された終端状態。 | rejected |
| Expired | 期限切れにより無効化された終端状態。 | expired |
| Validate | 認可リクエストの構文・必須パラメータ・redirect_uri を検証する。検証成功で AuthenticationPending へ遷移し、ログイン UI 提示を含む。 | validate |
| AuthenticateUser | ResourceOwner の認証を実行する。 | authenticate_user |
| RequestConsent | ResourceOwner にスコープへの同意を求める（同意 UI の表示）。 | request_consent |
| GrantConsent | ResourceOwner が同意を付与する。 | grant_consent |
| IssueCode | `AuthorizationCode` をクライアントへ発行する。 | issue_code |
| RedeemCode | クライアントが `AuthorizationCode` を `/token` でアクセストークンと交換する。 | redeem_code |
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
| Exchange | デバイスが /tokenで device_code を交換する。 | exchange |
| SlowDown | ポーリング間隔を増やすよう指示する。 | slow_down |
| Active | トークンまたは鍵が現在有効で第一線で使われている状態。RefreshToken と SigningKey で共用。 | active |
| Rotated | 子トークンに引き継がれ親が消費された状態。家族失効の対象になりうる。 | rotated |
| Revoked | 失効済み。/revoke もしくは家族失効により遷移する。RefreshToken と Consent で共用。 | revoked |
| Rotate | 親 RefreshToken を Rotated とし、新しい子トークンを発行する。 | rotate |
| RevokeToken | トークンを失効させる（/revoke または家族失効）。 | revoke_token |
| Deliver | `LogoutNotification` の配信（`backchannel_logout_uri` へのログアウトトークンの POST）が成功する。 | deliver |
| Exhaust | LogoutNotification の配送が max_attempts に到達し dead-letter として確定する。 | exhaust |
| Redeemed | 認可コードが /tokenで正規に交換された終端状態（AuthorizationCodeRecord）。AuthorizationCodeFlow の Exchanged と並行。 | redeemed |
| Stored | `/par` で受領済みかつ未参照の `request_uri`。 | stored |
| Used | `request_uri` が一度参照済みの終端状態。再使用できない。 | used |
| Use | `/authorize` から `request_uri` を一度だけ参照する。 | use |
| Granted | 同意が付与された状態（Consent の初期状態）。 | granted |
| RevokeConsent | ResourceOwner が同意を撤回する（GDPR Art.7(3)）。 | revoke_consent |
| Implicit | implicit grant は RFC 9700 §2.1.2 により参照実装から除外する。 | __excluded_implicit |
| PasswordGrant | Resource Owner Password Credentials grant は RFC 9700 §2.4 により参照実装から除外する。 | __excluded_password |

## Standards

### The OAuth 2.0 Authorization Framework

RFC 6749 — https://www.rfc-editor.org/rfc/rfc6749.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6749-AUTHORIZATION-CODE | required | MUST | Authorization Code Grant を認可エンドポイントとトークンエンドポイントで提供し、単一値であるべきセキュリティパラメーターが認可リクエスト内で重複していれば `invalid_request` として拒否する。 |
| RFC6749-CLIENT-CREDENTIALS | optional | MAY | Client Credentials Grant は `confidential` クライアントに限って許可する。 |
| RFC6749-IMPLICIT | excluded | MAY | Implicit Grant を提供する。 |
| RFC6749-PASSWORD-GRANT | excluded | MAY | Resource Owner Password Credentials Grant を提供する。 |

### The OAuth 2.0 Authorization Framework Bearer Token Usage

RFC 6750 — https://www.rfc-editor.org/rfc/rfc6750.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6750-AUTHORIZATION-HEADER | required | MUST | ベアラーアクセストークンは `Authorization` ヘッダーで受け付ける。 |
| RFC6750-QUERY-TOKEN | excluded | MAY | URI のクエリパラメーターによるアクセストークンの提示を受け付ける。 |

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
| RFC7517-JWKS | required | MUST | 公開可能な検証鍵を JWK Set として配布する。 |

### JSON Web Algorithms

RFC 7518 — https://www.rfc-editor.org/rfc/rfc7518.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7518-SIGNATURE-ALGORITHMS | required | MUST | JWT の署名には PS256 または ES256 を使い、アクセストークンと ID トークンには対称鍵署名を使わない。 |

### JSON Web Token

RFC 7519 — https://www.rfc-editor.org/rfc/rfc7519.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7519-REGISTERED-CLAIMS | required | MUST | JWT の発行者、subject、audience、有効期限、発行時刻、一意識別子を用途に応じて検証または発行する。 |

### JSON Web Token Profile for OAuth 2.0 Client Authentication and Authorization Grants

RFC 7523 — https://www.rfc-editor.org/rfc/rfc7523.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7523-CLIENT-ASSERTION | optional | MAY | クライアントアサーションの署名、発行者、subject、audience、有効期限、`jti` を検証する。 |

### OAuth 2.0 Dynamic Client Registration Protocol

RFC 7591 — https://www.rfc-editor.org/rfc/rfc7591.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7591-REGISTER | optional | MAY | クライアントメタデータを受け取り、`client_id` と登録結果を返す。 |
| RFC7591-REDIRECT-URI | required | MUST | Authorization Code Grant を利用するクライアントには `redirect_uri` の登録を要求する。 |

### OAuth Client ID Metadata Document

draft-ietf-oauth-client-id-metadata-document-00 — https://www.ietf.org/archive/id/draft-ietf-oauth-client-id-metadata-document-00.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| CIMD00-URL-SHAPE | required | MUST | `client_id` は `https` スキームと空でないパスを持つ URL でなければならない。 |
| CIMD00-FETCH | required | SHOULD | URL 形式の `client_id` を検出したら、その URL から Client ID Metadata Document を取得する。 |
| CIMD00-CLIENT-ID-MATCH | required | MUST | 取得した文書内の `client_id` フィールドが取得元 URL と厳密に一致することを検証する。 |
| CIMD00-REDIRECT-VALIDATE | required | MUST | 認可リクエストの `redirect_uri` が文書の `redirect_uris` 一覧に含まれることを検証する。 |
| CIMD00-STRUCTURE | required | MUST | 文書が正しい JSON であり、`client_id`、`client_name`、空でない `redirect_uris` を含むことを検証する。不正な場合は fail-closed で拒否する。 |
| CIMD00-CACHE | partial | SHOULD | HTTP キャッシュヘッダーに従って文書をキャッシュする。 |
| CIMD00-PRIVATE-KEY-JWT | excluded | MAY | `private_key_jwt` クライアント認証をインラインの `jwks` または `jwks_uri` 経由で提供してよい。 |

### Proof Key for Code Exchange by OAuth Public Clients

RFC 7636 — https://www.rfc-editor.org/rfc/rfc7636.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7636-VERIFY | required | MUST | トークンリクエストの `code_verifier` を認可時の `code_challenge` と照合し、認可リクエスト内で PKCE パラメーターが重複していれば拒否する。 |
| RFC7636-S256 | required | SHOULD | `code_challenge_method` は `S256` だけを許可する。 |
| RFC7636-PLAIN | excluded | MAY | `plain` 方式のコードチャレンジを許可する。 |

### OAuth 2.0 Token Introspection

RFC 7662 — https://www.rfc-editor.org/rfc/rfc7662.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7662-INTROSPECT | required | MUST | 認証済みのリソースサーバーへ `active` ステータスと許可されたメタデータを返す。 |
| RFC7662-INACTIVE | required | MUST | 無効なトークンには `active=false` だけを返す。 |

### Proof-of-Possession Key Semantics for JSON Web Tokens

RFC 7800 — https://www.rfc-editor.org/rfc/rfc7800.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7800-CONFIRMATION | optional | MAY | 送信者制約付きトークンの確認鍵情報を `cnf` クレームに格納する。 |

### Authentication Method Reference Values

RFC 8176 — https://www.rfc-editor.org/rfc/rfc8176.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8176-AMR | required | MAY | 実際に成立した認証方法を `amr` 値として記録する。 |

### OAuth 2.0 Authorization Server Metadata

RFC 8414 — https://www.rfc-editor.org/rfc/rfc8414.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8414-METADATA | required | MUST | 発行者と利用可能なエンドポイントおよび機能を Authorization Server Metadata として公開する。 |

### OAuth 2.0 Device Authorization Grant

RFC 8628 — https://www.rfc-editor.org/rfc/rfc8628.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8628-DEVICE-AUTHORIZATION | optional | MUST | `device_code`、`user_code`、`verification_uri` を発行し、ResourceOwner の判断を受け付ける。 |
| RFC8628-POLLING | required | MUST | `authorization_pending`、`slow_down`、`expired_token` のポーリングセマンティクスを守る。 |

### OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens

RFC 8705 — https://www.rfc-editor.org/rfc/rfc8705.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8705-CLIENT-AUTH | optional | MAY | 登録済みの Subject DN と検証済みクライアント証明書を照合してクライアントを認証する。 |
| RFC8705-CERT-BOUND | optional | MAY | アクセストークンをクライアント証明書のサムプリントへバインドし、リソースへのアクセス時に照合する。 |

### Resource Indicators for OAuth 2.0

RFC 8707 — https://www.rfc-editor.org/rfc/rfc8707.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8707-AUDIENCE | required | SHOULD | 発行するアクセストークンに空でない `audience` を設定し、意図しないリソースサーバーでの利用を防ぐ。 |
| RFC8707-MCP-RESOURCE-BINDING | required | MUST | `resource` パラメーターで指定された `McpResourceServer` にアクセストークンの `audience` を厳格に限定し、未登録、無効、複数指定の `resource` は fail-closed で拒否する。認可、Pushed Authorization Requests、トークン発行（認可コードの交換、リフレッシュトークンのローテーション、`client_credentials`、`device_code`、トークン交換）の全経路へ一様に適用する。`resource` が未指定であれば `client_id` を `audience` とする。 |

### OAuth 2.0 Token Exchange

RFC 8693 — https://www.rfc-editor.org/rfc/rfc8693.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8693-DELEGATION-DEFAULT | required | MUST | 交換の既定は委譲とする。発行するトークンは元のユーザーを `sub` に保ち、現在の行為者を `act` に記録し、以前の行為者を §4.1 に従って内側へ入れ子にする。 |
| RFC8693-IMPERSONATION | optional | MAY | なりすまし (`act` を落として `sub` を置き換える形) は、クライアントまたは Agent へ明示的に許可した場合だけ受け付ける。 |
| RFC8693-SUBJECT-TOKEN | required | MUST | 受け付ける `subject_token` は、自身が発行しイントロスペクションを通過したトークンか、`subject_token_type` が JWT-SVID の登録済み外部アテステーションに限る。 |
| RFC8693-DELEGATION-DEPTH | required | MUST | `act` チェーンの長さをテナントの実効委譲深さで制限する。テナントはシステム既定を下げられるが上げられず、ポリシーを解決できない場合は交換を拒否する。 |

### OAuth 2.0 Rich Authorization Requests

RFC 9396 — https://www.rfc-editor.org/rfc/rfc9396.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9396-REGISTERED-TYPES | required | MUST | `authorization_details` は、テナントが事前登録した `type` とそのスキーマに対して検証する。未登録の型やスキーマの不一致は部分的に受理せず拒否する。 |
| RFC9396-MONOTONIC-NARROWING | required | MUST | 発行または交換するトークンが持てるのは、同意した権限の部分集合に限る。後続の交換は権限を狭めることだけを許し、広げる要求は拒否する。 |
| RFC9396-SCOPE-PRECEDENCE | required | MUST | 同じ領域で `type` と粗い `scope` が重なる場合は構造化された詳細の上限を優先し、`authorization_details` で制限した領域を `scope` が再び広げる要求は拒否する。 |

### JSON Web Token Profile for OAuth 2.0 Access Tokens

RFC 9068 — https://www.rfc-editor.org/rfc/rfc9068.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9068-CLAIMS | required | MUST | JWT アクセストークンに `iss`、`sub`、`aud`、`exp`、`iat`、`jti`、`client_id` を含める。 |
| RFC9068-ASYMMETRIC-SIGNATURE | required | MUST | アクセストークンを非対称アルゴリズムで署名し、公開鍵で検証可能にする。 |

### OAuth 2.0 Pushed Authorization Requests

RFC 9126 — https://www.rfc-editor.org/rfc/rfc9126.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9126-PAR | optional | MAY | クライアント認証済みの PAR を保存し、短命な `request_uri` を返す。 |
| RFC9126-SINGLE-USE | required | SHOULD | `request_uri` は短命かつ一度だけ使用可能とする。 |

### OAuth 2.0 Authorization Server Issuer Identification

RFC 9207 — https://www.rfc-editor.org/rfc/rfc9207.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9207-ISS | required | MUST | 認可レスポンス、および安全に確定した `redirect_uri` へ返す認可エラーに発行者の識別子を含める。 |

### OAuth 2.0 Demonstrating Proof of Possession

RFC 9449 — https://www.rfc-editor.org/rfc/rfc9449.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9449-PROOF | optional | MAY | DPoP proof の署名、`htm`、`htu`、`iat`、`jti` を検証する。 |
| RFC9449-TOKEN-BINDING | optional | MUST | DPoP 利用時はアクセストークンに `jkt` を含め、提示された proof の鍵と照合する。 |
| RFC9449-ATH | optional | MUST | 保護リソースへ提示する DPoP proof は `ath` を含み、`base64url(SHA-256(access_token))` と一致しなければ拒否する。 |

### Best Current Practice for OAuth 2.0 Security

RFC 9700 / BCP 240 — https://www.rfc-editor.org/rfc/rfc9700.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9700-REDIRECT-MATCH | required | MUST | `redirect_uri` は登録値と完全に一致させ、未検証の URI へリダイレクトしない。 |
| RFC9700-AUTHORIZATION-CODE | required | MUST | リダイレクトを使うフローは Authorization Code Grant と PKCE で保護する。 |
| RFC9700-SENDER-CONSTRAINT | optional | SHOULD | 高いセキュリティを必要とするクライアントでは、アクセストークンに DPoP または mTLS による送信者制約を付ける。 |
| RFC9700-REFRESH-REPLAY | required | MUST | リフレッシュトークンをローテーションし、再利用を検知したら関連トークンを失効させる。 |

### OAuth 2.0 Protected Resource Metadata

RFC 9728 — https://www.rfc-editor.org/rfc/rfc9728.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9728-METADATA | required | MUST | 登録済みの `McpResourceServer` ごとに、対象リソースに対応する `authorization_servers` と対応スコープを含む Protected Resource Metadata を配信する。 |
| RFC9728-WELL-KNOWN | required | MUST | `/.well-known/oauth-protected-resource` で `resource` を指定して Protected Resource Metadata を取得できるようにする。 |
| RFC9728-IDMAGIC-API | required | MUST | `resource` が未指定であれば、realm の IdMagic API に対する Protected Resource Metadata と `account`、`management`、SCIM の各スコープ、対応する `bearer_methods_supported` を公開する。 |
| RFC9728-CHALLENGE | required | MUST | ベアラー保護リソースの `401 invalid_token` と `403 insufficient_scope` レスポンスでは、当該 realm の Protected Resource Metadata URL を `resource_metadata` 認証パラメーターで提示する。 |

### OpenID Connect Core 1.0 incorporating errata set 1

Final — https://openid.net/specs/openid-connect-core-1_0-18.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-CORE-CODE-FLOW | required | MUST | `code` レスポンスタイプによる OpenID Connect Authentication を提供し、`prompt` を空白区切りの重複しないトークン集合として検証する。`login` と `consent` をそれぞれ適用し、`none` はほかのトークンと併用せず UI を表示しない。 |
| OIDC-CORE-ID-TOKEN | required | MUST | ID トークンに `iss`、`sub`、`aud`、`exp`、`iat` と認証コンテキストを含める。 |
| OIDC-CORE-USERINFO | required | SHOULD | `openid` スコープのアクセストークンに対して、`sub` を含む UserInfo を返す。 |
| OIDC-CORE-HYBRID-IMPLICIT | excluded | MAY | Implicit Flow および Hybrid Flow を提供する。 |

### OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0

Final — https://openid.net/specs/openid-client-initiated-backchannel-authentication-core-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| CIBA-CORE-BACKCHANNEL-REQUEST | optional | MUST | クライアント認証済みのバックチャネル認証リクエストを受け付ける。`scope` は必須で `openid` を含み、`login_hint` または `id_token_hint` のちょうど一方から承認対象の User を解決し、`auth_req_id`、`expires_in`、`interval` を返す。解決できなければ `unknown_user_id` で拒否する。 |
| CIBA-CORE-POLL-MODE | optional | MUST | トークンエンドポイントの CIBA グラントで `authorization_pending`、`slow_down`、`access_denied`、`expired_token` のポーリングセマンティクスを守り、承認成立後の `auth_req_id` をちょうど一度だけトークン化する。 |
| CIBA-CORE-BINDING-MESSAGE | optional | SHOULD | `binding_message` を承認画面に表示し、クライアント、要求スコープ、`authorization_details` と併せて承認内容を示す。 |
| CIBA-CORE-PING-PUSH | excluded | MAY | `ping` および `push` のトークン配信モードを提供する。 |
| CIBA-CORE-USER-CODE | excluded | MAY | `user_code` パラメーターによる認証デバイス側の本人確認補助を受け付ける。 |
| CIBA-CORE-SIGNED-REQUEST | excluded | MAY | 署名済み JWT によるバックチャネル認証リクエストを受け付ける。 |

### OpenID Connect Discovery 1.0 incorporating errata set 1

Final — https://openid.net/specs/openid-connect-discovery-1_0-21.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-DISCOVERY-CONFIGURATION | required | MUST | well-known 設定から発行者、エンドポイント、対応機能を Discovery Metadata として公開する。 |

### OpenID Connect RP-Initiated Logout 1.0

Final — https://openid.net/specs/openid-connect-rpinitiated-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-LOGOUT-ENDPOINT | required | MUST | `end_session_endpoint` を公開し、RP からのログアウトリクエストを受け付ける。 |
| OIDC-LOGOUT-REDIRECT | required | MUST | `post_logout_redirect_uri` はクライアントに登録済みの値だけを許可する。 |
| OIDC-LOGOUT-ID-TOKEN-HINT | required | SHOULD | `id_token_hint` が与えられた場合は、署名、発行者、audience、subject、`sid` を検証してログアウト対象のセッションとクライアントを解決し、`client_id` パラメーターと矛盾するヒントを拒否する。 |

### OpenID Connect Front-Channel Logout 1.0

Final — https://openid.net/specs/openid-connect-frontchannel-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-FRONTCHANNEL-IFRAME | required | MUST | `end_session` レスポンスに、`frontchannel_logout_uri` を登録した各クライアントへの `iframe` を含める。`frontchannel_logout_session_required=true` のクライアントには `iss` と `sid` のクエリパラメーターを付与する。 |
| OIDC-FRONTCHANNEL-BEST-EFFORT | required | MUST | `iframe` の到達失敗を許容し、ローカルセッションの失効結果に影響させない。 |

### OpenID Connect Back-Channel Logout 1.0

Final — https://openid.net/specs/openid-connect-backchannel-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-BACKCHANNEL-LOGOUT-TOKEN | required | MUST | ログアウトトークンに `iss`、`sub`、`aud`、`iat`、`jti`、イベント（`http://schemas.openid.net/event/backchannel-logout`）を含め、対象がブラウザーセッションに由来する場合は `sid` も含めて署名する。 |
| OIDC-BACKCHANNEL-DELIVERY-RETRY | required | MUST | `backchannel_logout_uri` への配信失敗を再試行可能なジョブとして扱い、ローカルセッションやリフレッシュトークンの失効を配信結果に依存させない。 |
| OIDC-BACKCHANNEL-REPLAY | required | MUST | ログアウトトークンの `jti` により RP 側でリプレイを検出できる一意な値を発行する。 |

### OpenID Connect Session Management 1.0

Draft 28 — https://openid.net/specs/openid-connect-session-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-SESSION-MGMT-CHECK-IFRAME | optional | MAY | `check_session_iframe` を公開し、RP からの `postMessage` に対して OP セッションの状態を返す。 |

### FAPI 2.0 Security Profile

Final — https://openid.net/specs/fapi-security-profile-2_0-final.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| FAPI2-PROFILE-SELECTION | optional | MUST | `Fapi2SecurityProfile` を選択したクライアントだけに本プロファイルの追加制約を適用する。 |
| FAPI2-PAR-PKCE | optional | MUST | FAPI クライアントは PAR と S256 PKCE を使用する。 |
| FAPI2-CLIENT-AUTH | optional | MUST | FAPI クライアントは `private_key_jwt` または mTLS で認証する。 |
| FAPI2-SENDER-CONSTRAINT | optional | MUST | FAPI アクセストークンに DPoP または mTLS による送信者制約を付ける。 |

## State Transitions

### ClientSecretCredentialLifecycle

クライアントシークレット資格情報は発行時に `Active` となり、期限到達で `Expired`、管理者による個別の失効で `Revoked` となる。`Revoked` は期限切れより優先して表示する。

Initial: `Active` Terminal: `Expired`, `Revoked`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Expire | now() >= expires_at | Expired |  |
| Active | ClientSecretRevoked | — | Revoked |  |

### AuthorizationCodeFlow

`/authorize` から `/token` に至る認可リクエストのライフサイクル。

Initial: `Received` Terminal: `Exchanged`, `Rejected`, `Expired`

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

Initial: `Issued` Terminal: `Exchanged`, `Denied`, `Expired`

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

人間の承認を待つ ApprovalRequest のライフサイクル。Pending から Approved / Denied / Expired へ一方向に進み、Consumed へ到達できるのは Approved からだけである。Consume は保存層の CAS でちょうど一度だけ成立し、並行するポーリングが二重にトークンを得ることはない。

Initial: `Pending` Terminal: `Denied`, `Expired`, `Consumed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Pending | Approve | now() < expires_at | Approved |  |
| Pending | Deny | — | Denied |  |
| Pending | Expire | — | Expired |  |
| Approved | Consume | now() < expires_at | Consumed |  |
| Approved | Expire | — | Expired |  |

### RefreshTokenLifecycle

RefreshToken のライフサイクル。Rotate で子トークンに引き継がれ、Revoke で失効、Expire で期限切れ。Rotated 後も家族失効により Revoked へ遷移しうる（RFC 9700 §4.14）。

Initial: `Active` Terminal: `Revoked`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Rotate | now() < absolute_expires_at | Rotated |  |
| Active | RevokeToken | — | Revoked |  |
| Active | Expire | — | Expired |  |
| Rotated | RevokeToken | — | Revoked |  |
| Rotated | Expire | — | Expired |  |

### LogoutNotificationLifecycle

LogoutNotification のライフサイクル。Deliver で成功確定、Exhaust で max_attempts 到達による最終失敗確定 (dead-letter)。Jobs 側の Retry は Pending のまま attempts のみ増やす (状態遷移ではない)。

Initial: `Pending` Terminal: `Delivered`, `Failed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Pending | Deliver | — | Delivered |  |
| Pending | Exhaust | — | Failed |  |

### AuthorizationCodeRecordLifecycle

発行された AuthorizationCode 本体のライフサイクル。AuthorizationCodeFlow（AuthorizationRequest 側）の Exchanged に対応するのが Redeemed。

Initial: `Issued` Terminal: `Redeemed`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Issued | RedeemCode | now() < expires_at | Redeemed |  |
| Issued | Expire | — | Expired |  |

### ConsentLifecycle

同意レコードのライフサイクル。GDPR Art.7(3) により Granted → Revoked が可能。

Initial: `Granted` Terminal: `Revoked`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Granted | RevokeConsent | — | Revoked |  |
| Granted | Expire | — | Expired |  |

### PARRecordLifecycle

PAR で発行した `request_uri` のライフサイクル。`/authorize` から一度だけ参照できる（RFC 9126）。

Initial: `Stored` Terminal: `Used`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Stored | Use | now() < expires_at | Used |  |
| Stored | Expire | — | Expired |  |

## Authorization Boundary

プロトコルの経路には 4 つの異なる境界がある。

- **クライアントの認証**: `/token`、`/introspect`、`/revoke` などは、`client_secret_basic`、`client_secret_post`、`private_key_jwt`、`tls_client_auth`、`self_signed_tls_client_auth` のいずれかでクライアントを認証する。認証に失敗した理由は区別せず、一律に `invalid_client` を返す。
- **ユーザーの認証と同意**: `/authorize` はブラウザーのログインセッションで主体を決め、Application の割り当てと実効サインインポリシーを満たしたうえで、`(subject, client_id)` の同意が要求スコープを覆う場合にだけ認可コードを発行する。
- **トークンによるアクセス**: 発行したトークンで到達できる範囲は、そのトークンのスコープが決める。`account:*` を含むユーザー紐付きのスコープは、User の subject を持たないグラント (`client_credentials` や、subject を伴わない Token Exchange) では発行しない。
- **管理 API**: クライアント (`admin:clients_manage`)、同意 (`admin:consents_manage`)、`authorization_details` の型 (`admin:authorization_detail_types_manage`) の管理は、`admin` ロールを持つ、有効かつ認証済みのユーザーが所属テナントに対して行う。

管理 API では、API アクセストークンにロールに加えてリソースごとのスコープを要求する。`oauth-clients:*`、`consents:*`、`authorization-detail-types:*`、`mcp-resource-servers:*` がそれぞれのリソースに対応し、`read` が参照だけを、`write` が変更を許可する。リソースをまたぐ流用はできず、あるリソースのスコープで別のリソースを操作することはない。ロールポリシー一覧だけはテナント設定の参照なので `settings:read` に対応させる。

すべての判定は AuthZEN 形式の `authorize()` ポートを通り、規則表が要件の論理積を評価する。判定を返せない場合、事実が欠けている場合、ストアへ到達できない場合のいずれも、許可へ退避しない。

代行 (Token Exchange) は権限を広げない。`act` チェーン上のすべての actor が有効であり、要求するスコープと `authorization_details` が元の権限の部分集合であることを求める。チェーンの深さはテナントの `max_delegation_depth` (システム既定 3) を超えられず、上書きを解決できない場合は交換を拒否する。

## Design

### Internal Interfaces

#### FrontChannelLogout
対象の `sid` に ClientSession を持つ RP のうち、`frontchannel_logout_uri` を登録したものについて、EndSession のレスポンスへ埋め込む iframe の送信先一覧を算出する（OpenID Connect Front-Channel Logout 1.0）。`frontchannel_logout_session_required=true` のクライアントには `iss` / `sid` クエリパラメーターを付与する。到達失敗（RP 側での iframe 読み込みエラーなど）は許容し、ローカル失効の成否に影響させない。

#### BackChannelLogout
1 件の LogoutNotification を配信するジョブハンドラー（OpenID Connect Back-Channel Logout 1.0）。署名済みのログアウトトークン（`iss`、`sub`、`aud`、`iat`、`jti`、イベント、`sid`）を `target_uri` へ POST し、2xx を成功、それ以外のレスポンス、タイムアウト、接続失敗を再試行対象とする。Jobs Context の `kind=backchannel_logout_delivery` Job から呼び出し、`max_attempts` へ到達すると LogoutNotification を `Failed`（配信不能）に確定する。ローカル失効はこのインターフェースの成否にかかわらず、すでに確定している。

### Authorization and device lifecycles as declarative state machines

`AuthorizationRequest` とデバイスコードのライフサイクルは、`if` / `switch` のロジックへ分散させず、本書の State Transitions が規定する宣言的な遷移表（状態、イベント、遷移）で表す。アダプター層を再生成しても、クライアントに許可する遷移の集合が暗黙にずれないようにするためである。リフレッシュトークンファミリーは意図的に対象外とする。その状態空間は実質的に `{active, revoked, rotated}` だけであり、遷移の適否より親子関係のローテーショングラフが重要なため、レコードのフィールドと失効規則で表す（下記の Refresh token rotation を参照）。`authorization/usecase` と `device/usecase` は遷移ロジックを再実装せず、これらの表を直接使う。

### PKCE and Pushed Authorization Requests

PKCE の要否はクライアントごとのメタデータ (`require_pkce`) で定め、パブリッククライアントと FAPI 2.0 クライアントではデフォルトで必須、従来のコンフィデンシャルクライアントでは任意とする。パブリッククライアントと FAPI クライアントに最も強いデフォルトを適用しつつ、RFC 6749 時代のコンフィデンシャルクライアントへ移行を強制しないためである。`code_challenge_method` は `S256` だけを許可し、インターセプターがログから verifier を復元できる `plain` は拒否する。認可コードは PKCE の有無にかかわらず一度だけ使用できる短命な値（60 秒以下）とする。再利用の検出とリプレイ期間の最小化はいずれもこの性質に依存する。

Pushed Authorization Requests（`/par`）により、クライアントは認証済みのバックチャネルで認可パラメーターを送り、`request_uri` を介して `/authorize` から参照できる。これにより、`/authorize` における URL の改ざん、オープンリダイレクトの悪用、未認証リクエストの偽造を防ぐ。FAPI 2.0 クライアント（`require_pushed_authorization_requests`）には PAR を必須とし、それ以外のクライアントはどちらの経路も利用できる。`request_uri` は一度だけ使用でき、TTL は 600 秒以下とする。`/authorize` が `request_uri` を解決した後は、攻撃者が正当なプッシュ済みリクエストへパラメーターを追加できないよう、同じリクエストに追加されたクエリパラメーターを無視し、送信済みの値を優先する（RFC 9126 §4）。

### Client authentication

`token_endpoint_auth_method` は `private_key_jwt`、`tls_client_auth`、`none`、`client_secret_post`、`client_secret_basic` の 5 種類に対応する。FAPI 級の非対称認証から従来の共有シークレット方式までを覆い、新しいクライアントを最も強い選択肢へ誘導しながら、既存環境の移行を継続できるようにする。`client_secret_jwt`（HMAC）は意図的に実装しない。`private_key_jwt` を利用できる場合、対称鍵による代替は能力を増やさずリスクだけを増やすためである。クライアント認証の失敗では `client_id` の登録有無を明かさず、常に `401 invalid_client` を返す。未登録の `client_id` にも同程度の検証コストを課し、タイミングオラクルを避ける。

`handlers_http` の `private_key_jwt` 検証は固定した規則の集合に従い、Discovery Metadata での広告とサーバーの実際の検査を一致させる。署名アルゴリズムは `PS256` と `ES256` だけを許可し（`none` と HMAC は不可）、`iss == sub == client_id`、このサーバーの発行者またはエンドポイント URL に一致する `audience`、クライアントが登録したインラインの `jwks` または `jwks_uri` から解決した署名鍵、上限付きのアサーション有効期間、`jti` の一度限りの利用を検査する。`jti` は DPoP と TTL および監査上の意味が異なるため、DPoP のリプレイストアとは別のストアに置く。同じリクエストで `client_assertion` と Basic 認証またはシークレット認証を組み合わせた場合は、RFC 6749 §2.3 に従い `invalid_request` として拒否する。

### Client ID Metadata Documents (CIMD): registry-less client resolution

RFC 7591 の Dynamic Client Registration に加え、パスを持つ `https` URL 形式の `client_id` は `OAuth2Client` Repository ではなく、クライアントがホストする Client ID Metadata Document からその場で解決する。解決結果は永続化しない。Repository に `client_id` がない場合は文書を取得して 5 分間キャッシュし、他の処理経路と同じ `OAuth2Client` の形へ変換する。これにより、`redirect_uri` の照合、同意画面の描画、PKCE、スコープの処理に CIMD 固有の分岐は不要になる。統合点は `OAuth2ClientRepository` を埋め込み、`FindByID` だけを上書きする Decorator、`client/cimd_http.ClientRepositoryWithCIMD` である。Repository で見つかれば取得前に終了し、他のメソッド（`Save`、`Delete`、`FindAll`、資格情報の一覧）は変更せず委譲する。Composition Root (`cmd/internal/bootstrap`) で一度だけ接続するため、`authorize.go`、`push_authorization_request.go`、`client_auth.go` は変更不要である。

取得には、`tokens_jose.JWKResolver` が `jwks_uri` に使うものと同じ SSRF 対策済みダイヤラー `shared/security/safehttp` を使う。HTTPS のみ、DNS 解決後にパブリック IP だけを許可、検証済み IP への直接接続、環境プロキシの不使用、リダイレクト回数・タイムアウト・レスポンス本文サイズの上限を強制する。プロキシは検査済みの接続経路の外で最終送信先を解決して接続し、トランスポート層の SSRF 境界を迂回するため、直接接続が必要である。共通パッケージでは 2 つの取得処理を別々に実装せず、1 つの堅牢化した実装の背後に置く。MVP が受け入れるのは、`token_endpoint_auth_method` を省略するか `none` と宣言する文書だけであり、それ以外はフェイルクローズに拒否する。文書の `client_id` フィールドは取得元 URL と完全に一致しなければならない。解決したクライアントの `scope` は文書の自己宣言値（デフォルトは `openid`）とし、新しい管理者管理カタログではなく RFC 7591 DCR と同じ自己宣言型の信頼モデルを使う。CIMD で解決したクライアントは `Application` に関連付けない。自己登録した DCR クライアントと同じであり、`ApplicationGate` は Application レコードがない場合をフェイルクローズに拒否せず、許可として扱う。

### Token formats: JWT access tokens, opaque refresh tokens

アクセストークンはデフォルトで自己完結型の JWT（RFC 9068）として、リフレッシュトークンはデータベースを裏付けとする中身の見えない参照として発行する。この非対称性は意図的である。リソースサーバーが多い場合、リクエストごとに `/introspect` を往復するとスケールしない。そのため、リソースサーバーが JWKS だけで検証できる JWT をデフォルトとし、即時失効できない期間を短い TTL（600 秒）で制限する。リフレッシュトークンにはローテーションとファミリー全体の失効が必要であり、これは自然にデータベースレコードの操作となる。トークンの中身を見えなくすることで、利点のない JWT と失効レコードの同期を避ける。リフレッシュトークンは平文ではなく SHA-256 ハッシュで保存する。送信者制約（`cnf`）やリアルタイムの失効状態を確認したいリソースサーバー向けに `/introspect` を公開するが、JWT アクセストークンのデフォルトの検証経路にはしない。

### Refresh token rotation and reuse detection

リフレッシュトークンは使用するたびにローテーションする。提示されたトークンを `rotated` にし、`parent_id` を持つ新しいトークンを発行する。1 回の認可コード交換から派生したすべてのトークンは `family_id` を共有する。すでにローテーション済みのトークンが提示された場合は再利用としてリクエストを拒否し、その `family_id` に属する全トークンを失効させ、`RefreshTokenReuseDetected` 監査イベントを発行する。運用と監査を単純に保つため、パブリッククライアントとコンフィデンシャルクライアントへ一様に適用する。2 つのブラウザータブなどによる正当な同時利用もリプレイと区別せず、一方だけが成功し、他方は再利用と判定する。猶予期間に伴う複雑さと外部状態のコストを避ける代わりに、まれに再ログインを求めることを受け入れる。`absolute_expires_at` は発行時に 30 日で固定し、ローテーションでは延長しない。

### Sender-constrained tokens: DPoP and mTLS

DPoP（RFC 9449）は TLS 終端プロキシの変更なしに Web アプリケーション、SPA、ネイティブクライアントで同じように動作するため、送信者制約のデフォルト方式とする。すでにクライアント PKI を運用する組織、特に FAPI や金融系のクライアントには、mTLS（RFC 8705）も選択肢として提供する。FAPI 2.0 プロファイルを宣言するクライアントは少なくとも一方を使い、一般プロファイルのクライアントは `dpop_bound_access_tokens` で選択する。DPoP Proof の検証では、`jwk` ヘッダーの署名、`htm` / `htu` / `iat` / `jti`、上限付きの時刻ずれ、`jti` のリプレイ期間を検査する。発行するトークンは JWK のサムプリントを `cnf.jkt` に持つ。

Proof の検証はパラメーター化せず、エンドポイントの種類で分ける。バインドすべき対象が 2 種類で異なるためである。トークンエンドポイントでは対象のアクセストークンがまだ存在しないため、Proof は `ath` を持たない。保護リソースでは Proof が `ath = base64url(SHA-256(提示された access token))` を持ち、Introspection 後の表現ではなく、クライアントが提示した生のトークン文字列と定数時間で比較する。`ath` がなければ Proof は鍵の所持だけを示し、同じ `htm` / `htu` と鍵に対して発行したすべてのトークンで使い回せる。これは長時間動作し、複数のリソースサーバーをまたぎ、委任の各段で異なるトークンを持つエージェントにとって特に重要である。保護リソースで `ath` がない場合は、猶予フラグの背後で許容せず拒否する。オプトアウトをデフォルトにすると、バインドされていない Proof を受け入れることになり、まさに防ぐべき失敗だからである。mTLS の検証では、TLS 終端プロキシが検証済みのクライアント証明書を渡すことを信頼し、登録済みの `tls_client_auth_subject_dn` と照合して、発行トークンを `cnf.x5t#S256` でバインドする。`/userinfo` は提示された証明書のサムプリントがトークンの `cnf` と一致する場合だけ受け入れる。アクセストークンとリフレッシュトークンの両方が送信者制約を持つ。リフレッシュトークンではストアレコードの `sender_constraint` フィールドに保持するため、Proof of Possession はローテーション後も維持される。`/introspect` のレスポンスにも `cnf` を含め、リソースサーバーがリクエストごとに DPoP Proof を再検証できるようにする。

### Consent

同意はクライアント単位や対話単位ではなく、`(subject, client_id)` ごとに付与済みスコープの集合として永続化する。クライアント単位のグラントでは、後から追加された高権限のスコープへ暗黙に拡張され、目的ごとの同意と矛盾する。毎回尋ねると同意疲れを招き、場当たり的な記憶機能を求める圧力が生じる。要求された全スコープが期限内かつ未失効のグラントですでに覆われている場合だけ同意 UI を省略し、新しいスコープがあれば差分だけを強調する UI を表示する。グラントは定期的な再同意の期待に合わせ、付与から 365 日で失効する。`prompt=consent` は既存グラントにかかわらず UI を強制する。同意の取り消しが影響するのは将来の認可だけである。同時にクライアントのリフレッシュトークンを失効させる処理は、リフレッシュトークンローテーションのファミリー失効を再利用する、明示的で別の操作とする。発行済みの短命なアクセストークンは自然な期限切れを待つ。

### Authorization policy (AuthZEN)

ポリシー境界をまたぐ認可判断には、AuthZEN 型の `authorize({subject, action, resource, context})` ポートを使う。ローカル実装は、アプリケーションが所有するアクションと要件の表、および名前付きのルール関数を Go で評価する。ロールポリシーの検査も同じ表を読み、表示する要件を実行時のローカル認可と一致させる。リモート AuthZEN アダプターは同じポートを実装し、組み立て時に選択する。ポリシーの失敗、未定義のアクション、不正なリモートレスポンス、利用できないリモート評価器では、許可へフォールバックせず保護対象の操作を拒否する。

Cedar は実行時ポリシーの正ではない。Go 実装は既存のリクエスト形式を評価できるが、スキーマバリデーターはまだ実験的 API である。実行時の評価だけを移すと、ロールポリシー検査用の Go の要件表が残る。検証機能に安定した互換性の契約があり、1 つのポリシー表現で、2 つ目の認可の正を導入せずに実行時ルールと検査用メタデータの両方を置き換えられるようになった場合だけ、Cedar を再検討する。

### Discovery

OAuth 2.0 Authorization Server Metadata と OIDC Discovery Metadata は手作業で保守せず、派生成果物とする。実行時に、TypeSpec から生成したランタイム契約 (`backend/shared/spec`) からリクエスト先テナントの発行者を当てはめて組み立てる。対応するグラント、認証方式、署名アルゴリズム、レスポンスタイプ、PKCE 方式はいずれも同じ契約に由来するため、広告した内容とサーバーが実際に強制する内容がずれることはない。テナントで有効な `authorization_details` の型だけは実行時に解決して重ねる。

### Device Authorization Grant

デバイスフロー（RFC 8628）、すなわち `POST /device_authorization`、`/device` の検証 UI、`/token` の `device_code` グラントでは、承認・拒否・交換の遷移をその場で再実装せず、`DeviceCodeFlow` の遷移表を共有する 1 つの遷移関数を使う。`device_code` は 32 バイトのランダム値であり、ベアラーシークレットとして SHA-256 ハッシュだけを保存する。`user_code` は母音と見分けにくい文字を除いた 20 文字の縮小済みで曖昧さのない文字集合を使い、`WDJB-MJHT` のようにグループ分けして表示する。ポーリングは仕様中核が所有する間隔とバックオフ増分に従い、`authorization_pending` / `slow_down` / `access_denied` / `expired_token` を返す。二重発行を防ぐため、承認済みコードをトークン発行前に `approved → exchanged` へ遷移させる。

### Lifetime, security, and retention configuration

プロトコルの時間値とセキュリティパラメーター、すなわち認可コードの TTL（60 秒、一度だけ使用可能）、PAR の `request_uri` の TTL（600 秒、一度だけ使用可能）、アクセストークンの TTL（600 秒）、ID トークンの TTL（3,600 秒）、リフレッシュトークンの TTL（スライディング 14 日、絶対期限 30 日）、デバイスコードとユーザーコードの TTL（600 秒）、デフォルトのポーリング間隔（5 秒、`slow_down` ごとに 5 秒加算）、クライアント認証とコード交換の流量制限、DPoP の時刻ずれとリプレイ期間、同意レコードの保持期間（7 年）は、製品目標ではなく 1 か所にまとめて記録する。これらはエラーバジェットの意味を持つ可用性・レイテンシー SLO ではなく、プロトコル、セキュリティ、運用の設定だからである。単一のモデルまたは状態インターフェースで自然に強制できる値は制約、ガード、契約として表す。複数リクエストにまたがる流量制限や、ライフサイクルにまたがる保持期間など、単一要素に属さない値は共有設定レコードを正とする。

### Agent principals and token-exchange delegation

`Agent` は `User` や `OAuth2Client` と異なる第一級のプリンシパルであり、アイデンティティ、所有関係、目的、緊急停止を含むライフサイクルを所有する。ただし独自の資格情報の基本要素は持たず、1 個以上の既存の `OAuth2Client` 登録に束縛するため、エージェントのガバナンスに重複する第 2 の資格情報・暗号の仕組みは不要である。すべてのエージェントは所有者 (`User` またはグループ) を必須とし、所有者の離任をエージェントのアクセスへ連鎖する。すべてのトークン発行経路で `status` (`active` / `disabled` / `killed`) を安全側に検査し、状態を解決できない場合は発行を許可しない。所有者の離任の連鎖は、エージェントの `status` を書き換える一度きりの状態遷移ではなく、発行のたびに所有者を解決して有効性を確かめる評価として実装する。所有者を解決できない場合、または所有者が無効化・削除されている場合は発行しない。エージェントの `status` を経由しないのは、一度きりの書き込みが届かなかったデプロイでガードごと失われるのを避けるためであり、所有者が復帰すれば発行も自動的に再開する。既に発行済みのトークンは、この評価ではなく SharedSignals の失効エポックが無効化する。アクセストークンのクレームは任意のプリンシパル種別の標識を持ち、既存のトークン利用者を壊さず、リソースサーバーと AuthZEN のポリシー層がエージェント向けに発行したトークンを区別できるようにする。

ユーザーの代理行為は `/token` の OAuth 2.0 Token Exchange（RFC 8693）として実装する。デフォルトの結果はなりすましではなく委任である。交換後のトークンは元のユーザーを `sub` に保ち、現在の行為者であるエージェントを `act` クレームに記録する。RFC 8693 §4.1 に従って以前の行為者を内側へ入れ子にするため、下位エージェントへの委任チェーンを追跡できる。なりすまし（`act` を削除して `sub` を置換）は、クライアントまたはエージェントへ明示的に許可した場合だけ利用できる。判断できない場合は、監査証跡を保つ委任をデフォルトとする。`may_act` と AuthZEN ポリシーが、許可する行為者、対象者、深さの組を共同で制御する。交換時には、結果を単一の対象へ狭める `resource` を必須とする（RFC 8707）。委任の最大深度で `act` チェーンの長さを制限する。上限はテナントごとに下げられるが、システム既定を超えて上げることはできない。テナント設定から認可の境界を緩める経路を作らないための非対称性であり、パスワードポリシーの上書きと同じ扱いである。テナントの委譲ポリシーを解決できない場合は、既定へ退避せず交換を拒否する。

交換後のトークンが自律実行と利用者の代理のどちらであるかは、新しい永続状態ではなく `act` チェーンとプリンシパル種別から導出する。導出は 1 つの関数に閉じ、イントロスペクションの応答と監査イベントの双方がそれを通る。リソースサーバーと監査の担当者が同じ規則を各自で書き直すと解釈がずれ、しかもその食い違いは調査のときに最も見つけにくい形で現れるからである。導出値であるため、モードがクレームと食い違う第二の真実は生まれない。交換後のトークンは短命であり、リフレッシュトークンを発行しない。継続には再交換が必要なため、失効が有効に働く。subject token の送信者制約は交換後のトークンへ引き継ぎ、鍵の所持証明を失わない。

### Rich Authorization Requests for agent-scoped permissions

粗い OAuth スコープでは「口座 X から最大 $100 を送金」のような権限を表せないため、`/authorize`、`/par`、`/token` (上記のトークン交換 grant を含む) は RFC 9396 `authorization_details` を受け入れ、広いスコープではなく構造化され上限のある権限を要求で宣言できるようにする。テナントごとに事前登録した `type` だけを受け入れ、各詳細をスキーマに対して安全側に検証する。未登録の種別やスキーマの不一致は部分的に受理せず拒否する。発行または交換するトークンが持てるのはユーザーが同意した内容の部分集合だけであり、後続の交換はその部分集合をさらに狭めることしかできず、広げられない。検査に使う半順序 (対象の包含、上限の単調な減少) は登録済みのスキーマ自体が定義する。同意 UI は生の JSON ではなく、スキーマに結び付いた人間向けテンプレートから各詳細を描画する。リソースサーバーは IdP が発行または introspect した詳細だけを信頼境界とし、付与内容を再解釈または拡張してはならない。`type` と粗い `scope` が同じ領域で重なる場合は構造化された詳細の上限を優先し、`authorization_details` で制限済みの領域を `scope` が再び広げる要求は拒否する。

### Backchannel human approval for agent actions (CIBA)

送金、データ削除、外部公開など重大な行為を始める自律型エージェントには、その場にいない人間による事前承認を要求できる。`POST /bc-authorize`（OpenID CIBA Core）により、認証済みクライアントは帯域外で承認要求を起票できる。`/token` の `urn:openid:params:grant-type:ciba` グラントは、人間が判断するまで要求を保留する。どの Agent にこの承認を義務付けるかはガバナンス層が決定し、OAuth2 Context は `AgentKind.Supervised` だけを理由にすべてのグラントを一律に拒否しない。

CIBA は別の認証方式ではなく、OAuth 2.0 上の承認機能としてモデル化し、同意やステップアップ認証を置き換えない。同意は長命な `(subject, client_id)` のスコープグラントのままであり、承認リクエストは 1 つの行為に対する短命な判断である。ステップアップ認証はバックチャネルフローに置き換えられず、承認対象の行為を保護する。アカウントポータルは判断を記録する前に再認証を要求する。

判断を保持するレコードは CIBA 固有の形にしない。OpenID AuthZEN の Access Request and Approval Profile が定義する、認可判断の前提条件を要求、追跡、充足、再評価するモデルに合わせ、Aggregate は UUID をキーとする `ApprovalRequest` とする。`auth_req_id` は SHA-256 の照合用ダイジェストだけを保存する 32 バイトのベアラーシークレットであり、`interval_seconds` と `last_polled_at` は転送方式に固有のフィールドである。判断と同時ポーリングを 1 つのストア境界で直列化するため、これらを同じ永続化レコードに置く。アカウントポータルは UUID で承認要求を指すため、人間向けインターフェースへベアラーシークレットは渡らない。

`Pending → Approved | Denied | Expired` と `Approved → Consumed` は一方向である。発行ではデバイスグラントの `Exchange` と同じ比較交換をストア上で行い、まだ `approved` の行だけを変更する。そのため、同時に行われた 2 回のポーリングが両方ともトークンを発行することはない。`/token` は未承認のすべての状態をフェイルクローズに扱う。`pending` のポーリングは `authorization_pending`、`interval` より速いポーリングは `slow_down`（`interval` に 5 秒を加算）、拒否は `access_denied`、期限切れは `expired_token`、消費済みリクエストの 2 回目の交換は `invalid_grant` を返す。

実装し、メタデータで広告する配信モードは `poll` だけである。`ping` は接続せず拡張点として残し、通知基盤が必要になる `push` は対象外とする。`user_code` は未対応として広告する。承認画面はすでに User の認証済みセッションとステップアップ認証の背後にあるため、その前へ 2 つ目の弱い共有シークレットを追加しても利点がない。リクエストの有効期間はデフォルトで 300 秒とし、正の `requested_expiry` は 600 秒以下だけを受け入れ、それを超える値または 0 以下の値は拒否する。ポーリング間隔には別の規約を導入せず、デバイスグラントの 5 秒および 5 秒ずつ増加する設定を再利用する。帯域外通知は、テンプレートキーを 1 個追加した既存のテナント上書き可能な通知カタログを再利用し、パスワードリセットやセキュリティ警告と同じ方法で人間へ承認リクエストを届ける。

承認する User は `/bc-authorize` で、`login_hint`（ユーザー名またはメールアドレス）と検証済みの `id_token_hint` のいずれか正確に 1 個から解決する。ヒントがない場合や複数ある場合は `invalid_request` とする。解決できないヒント、`active` ではない User、別テナントの User はすべて同じ `unknown_user_id` にまとめ、存在するアカウントの探索にエンドポイントを悪用できないようにする。生成したリクエストはその User にバインドし、判断にはアカウントセッション、ステップアップ認証、CSRF 検証を要求する。別の User のリクエストは一覧に表示せず、ID で直接指定しても拒否する。`binding_message` だけではどのリクエストかは分かっても操作内容が分からないため、承認画面にはエージェント、クライアント、要求されたスコープ、`authorization_details` を表示する。`authorization_details` の表示には RAR の人間向けレンダリングを再利用する。

承認リクエストは別の短命なストアではなく、他の揮発性 OAuth2 レコードと同じ `UNLOGGED` PostgreSQL テーブルおよびメモリアダプターへ保存し、共通の一時データ削除処理で消去する。1 種類のレコードのために 2 つ目のストレージ技術を導入しても、既存の規約にはない利点を得られないためである。

### OIDC session binding and logout propagation

`sid` クレームは `LoginSession.id` 自体であり、RP ごとの値ではなく、1 つのブラウザーセッションについてすべての relying party が共有する。OIDC の `sid` は OP セッションを表すため、RP ごとの `sid` では 1 回のセッション失効から影響する全 RP をたどれない。`sid` は `authenticate_user` の完了時に一度だけ `AuthorizationRequest` へ伝播し、その後 `AuthorizationCodeRecord` → `RefreshTokenRecord` → `IdTokenClaims` を通る。Authentication の `LoginSession` が唯一の正であり、その属性を OAuth2 へ複製しない。`ClientSession` はログアウト通知用の `(sid, client_id)` 配信インデックスであり、2 つ目のセッション状態ではない。`RefreshTokenRecord.sid` はローテーション後も残るため、ファミリーごとにたどらず、1 回の「このブラウザーセッション」の失効で、同じ `sid` にバインドされた全クライアント・全ファミリーのリフレッシュトークンを失効できる。

`/end_session` の `id_token_hint` は、署名、`iss`、`aud`、`sub`、`sid` をフェイルクローズに検証する。`aud` は明示的な `client_id` パラメーターと一致しなければならず、暗黙には無視しない。ログアウト時に ID トークンが期限切れであることは一般的なため、`exp` は意図的に検査しない。ヒントがなければ `client_id` とブラウザーの Cookie で解決する。バックチャネルログアウトの配信は専用キューではなく、永続的で冪等な `Job` として Jobs Context に渡す。配信に失敗してもローカルセッションとリフレッシュトークンの失効はロールバックしない。フロントチャネルログアウトは同じリクエスト内で計算する `iframe` の送信先一覧であり、RP 側の `iframe` の失敗は許容し、配信を保証しない。アクセストークンの失効は対象外とする。アクセストークンは署名だけで検証する自己完結型 JWT のままとし、即時失効のために全リソースサーバーの検証をストア参照へ変える代わりに、リフレッシュトークンファミリーの即時失効と RP 通知に加えて最大 600 秒の残存リスクを受け入れる。`check_session_iframe`（OIDC Session Management 1.0）は、Discovery Metadata での広告と、ブラウザーの Cookie が有効なセッションを示すかどうかの静的検査だけを提供する。

### Conventions

曖昧さがなく列挙できる形を持つプロトコル上重要な振る舞いは、仕様で一度だけ宣言し、ユースケースやアダプターで再実装せずに使う。状態遷移、認可規則、Discovery Metadata、デバイスフローの遷移はいずれもこの形に従う。Context 直下の `domain`、`ports`、`usecase` パッケージは機能単位の上に置く互換ファサードであり、Composition Root は `module.go` だけである。

### Design Decisions

- 認可リクエストとデバイスコードのライフサイクルは、その場の条件分岐ではなく宣言的な遷移表で表す。条件分岐に散らすと、クライアントに許可する遷移の集合が実装のたびに暗黙にずれるからである。
- PKCE はすべてのクライアントに一律で強制せず、公開クライアントと FAPI 2.0 クライアントではデフォルトで必須、従来の confidential クライアントでは任意とする。
- Pushed Authorization Requests は FAPI 2.0 クライアントで必須、その他では任意とし、最も強い保証が必要なクライアントの `/authorize` で URL の改ざんと未認証リクエストの偽造を防ぐ。
- クライアント認証は FAPI 級の非対称認証から従来の共有シークレットまで 5 方式に対応し、`client_secret_jwt` は意図的に除外する。
- `private_key_jwt` の検証は、アルゴリズムの許可リスト、発行者 / subject / audience、上限付きの Assertion 有効期間、リプレイ防止という固定規則に従い、Discovery の広告内容とサーバーでの強制を一致させる。
- HTTPS URL 形式の `client_id` を解決するため、Dynamic Client Registration に代わり、クライアント登録情報を永続化せずに利用できる Client ID Metadata Documents に対応する。
- アクセストークンは自己完結型 JWT、リフレッシュトークンはデータベースに保存する不透明な参照をデフォルトとし、失効時と検証時の異なるスケーリング特性に合わせる。
- リフレッシュトークンは交換のたびにローテーションし、ローテーション済みトークンが提示された場合は、公開クライアントと confidential クライアントのどちらでもファミリー全体を失効させる。
- 送信者制約は DPoP をデフォルトとし、クライアント PKI を運用するクライアントには mTLS も選択肢として提供する。
- 同意はクライアントごとや操作ごとではなく、`(subject, client_id)` ごとに付与済みスコープ集合として永続化し、暗黙のスコープ拡大と同意疲れを避ける。
- ポリシー境界をまたぐ認可判断には AuthZEN 型の `authorize()` ポートを使用する。ローカルの Go 規則表はロールポリシーの確認にも使用し、外部 AuthZEN サービスはリモートアダプターの背後に隔離する。
- Go のスキーマ検証器が実験段階にあり、移行後もロールポリシー確認用の第 2 の Go 認可情報源が残る間は Cedar を採用しない。
- Discovery Metadata は手作業でもビルド時でもなく、実行時に契約から組み立てる。手作業では実装とずれ、ビルド時では手順の省略で生成物が古くなるからである。
- トークン、コード、PAR の TTL、レート制限、DPoP のリプレイ防止期間、同意の保持期間は、可用性 SLO ではなくプロトコルとセキュリティの設定であるため、製品目標にはせず 1 か所にまとめる。
- `Agent` はアイデンティティとライフサイクルを所有する第一級のプリンシパルだが、独自の資格情報は持たず、既存の `OAuth2Client` 登録に関連付ける。
- ユーザーの代理行為は OAuth 2.0 Token Exchange として実装し、偽装ではなく、元の `sub` と `act` の Agent を保つ委任をデフォルトとする。
- Agent 単位の権限は粗いスコープではなく RFC 9396 `authorization_details` で表し、送金上限などの制約を宣言する。後続の Token Exchange では権限を狭めることだけを許す。
- `sid` クレームには、ブラウザーセッションに参加するすべての RP が共有する `LoginSession.id` そのものを使う。1 回のセッション失効から、影響するすべての RP をたどれるようにするためである。
- Agent の操作に対する人間の承認は、同意やステップアップ認証を置き換えず、OAuth 2.0 上の承認機能として CIBA で実装する。
- 承認判断の記録には、UUID をキーとし通信方式に依存しない `ApprovalRequest` を使用する。CIBA の検索フィールドとポーリング用フィールドは通信上の記録だが、ストアが判断とポーリングを不可分に直列化できるよう同じ場所に置く。
- 承認済みリクエストはストア単位の Compare-and-Set でトークンに交換し、同時ポーリングによる二重発行を防ぐ。それ以外の状態は `/token` でフェイルクローズに扱う。
- CIBA の配信モードは `poll` だけを実装する。承認画面はすでにユーザーセッションとステップアップ認証で保護されているため、`user_code` は未対応として広告する。
- `supervised` の Agent に承認を義務付けるかどうかはガバナンス層が決定し、OAuth2 Context では一律に強制しない。

## Scenarios

### REQ-OAUTH2-001: ユーザーに紐づく OAuth グラントは account スコープのアクセストークンを発行できる
- ACTOR RegisteredClient
- GIVEN クライアントは `account:read` と `account:write` を許可スコープとして登録している
- GIVEN 有効な User が Authorization Code + PKCE または Device Authorization で `account:read` に同意している
- WHEN クライアントがユーザーに紐づくグラントを `/token` で交換する
  - ALT `client_credentials` または User の subject を持たない Token Exchange で account スコープを要求する → トークンリクエストを InvalidScopeError で拒否する
  - ALT クライアントの許可スコープまたは User の同意に account スコープが含まれない → account スコープは発行されない
- THEN アクセストークンの `sub` は同意した User、audience はレルムの IdMagic API、スコープは `account:read` になる
- THEN account リソースサーバーは、トークンの subject 本人による参照操作だけを許可する

### REQ-OAUTH2-002: API トークン発行者は account 同意スコープで自分の同意だけを操作できる
- ACTOR SelfApiClient
- GIVEN クライアントは対象テナントの active User に固定された有効な API access トークンを提示している
- WHEN クライアントが自身の active 同意の参照または撤回を要求する
  - ALT account:read だけで同意 revoke を要求する → 操作は AccessDeniedError で拒否される
  - ALT トークンのテナントまたは user_id が操作対象と一致しない → 操作は AccessDeniedError で拒否される
- THEN account:read scope は自身の active 同意の参照だけを許可する
- THEN account:consents:write scope は自身の同意の撤回だけを許可する

### REQ-OAUTH2-003: 管理 API クライアントは OAuth リソースのスコープで許可された操作だけを実行できる
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API access トークンを提示している
- WHEN クライアントが OAuth2 クライアント、認可詳細タイプ、または MCP リソースサーバーの操作をリクエストする
  - ALT oauth-clients:read だけで OAuth2 クライアントの変更を要求する → 操作は AccessDeniedError で拒否される
  - ALT 別 resource の scope で操作を要求する → 操作は AccessDeniedError で拒否される
  - ALT トークンのテナントとリクエスト先のテナントが一致しない → 操作は `AccessDeniedError` で拒否される
- THEN oauth-clients:read scope は OAuth2 クライアントの参照だけを許可する
- THEN `authorization-detail-types:write` スコープは認可詳細タイプの変更だけを許可する
- THEN `mcp-resource-servers:read` スコープは MCP リソースサーバーの参照だけを許可する

### REQ-OAUTH2-004: 管理者は自身に可視なロールポリシーを確認できる
- ACTOR TenantAdministrator
- GIVEN ロール=["admin"] の管理者が認証済みである
- WHEN 管理者がロールポリシー一覧を取得する
  - ALT プリンシパルが `admin` または `system_admin` ではない → ロールポリシー一覧を AccessDeniedError で拒否する
- THEN レスポンスには参照可能なロール、権限、対応する HTTP インターフェースが含まれる

### REQ-OAUTH2-005: 認可コードフローでアクセストークンと ID トークンを取得できる
- ACTOR RegisteredClient
- GIVEN "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- GIVEN ユーザー "alice" は "web-app" に scope "openid プロファイル" を同意済みである
- WHEN "web-app" として scope "openid プロファイル" で認可リクエストを送る
  - ALT 認可リクエストの redirect_uri が未登録である → リダイレクトは行われず IdP がエラーページを表示する → エラー "InvalidRequestError"
  - ALT 単一値の認可パラメーターが重複する、`prompt` に重複または未対応のトークンがある、または `none` がほかの `prompt` トークンと併用される → 認可コードは発行されない → 安全に確定した登録済みの `redirect_uri` があれば `state` と発行者の識別子を含む `invalid_request` を返す → それ以外はリダイレクトせず IdP がエラーページを表示する
  - ALT `request_uri` と、併用を許可しないフロントチャネル認可パラメーターが混在する → 認可コードは発行されない → エラー "InvalidRequestError"
  - ALT `prompt=none` で既存セッションまたは必要な同意がない → UI とログインへのリダイレクトは発生しない → 既存セッションがなければ `state` と発行者識別子を含む `login_required` を登録済みの `redirect_uri` へ返す → 同意がなければ `state` と発行者識別子を含む `consent_required` を登録済みの `redirect_uri` へ返す
- WHEN クライアントが発行された認可コードを正しい PKCE verifier で交換する
  - ALT PKCE verifier が一致しない → 認可コードを誤った code_verifier で交換する → エラー "InvalidGrantError" → トークンは発行されない
  - ALT 同じ認可コードを 2 回交換する → 1 回目の応答には access_token が含まれる → 2 回目はエラー "InvalidGrantError" → 発行ファミリーのトークンがすべて失効する → "RefreshTokenReuseDetected" が発行される → "TokenRevoked" が発行される
  - ALT 認可コードが発行から 60 秒を超えている → 認可コードの交換はエラー "InvalidGrantError" → 認可コードの状態は Expired になる
- THEN レスポンスに `access_token`、`id_token`、`refresh_token` が含まれ、`token_type` は `Bearer`
- THEN "UserAuthenticated" が発行される
- THEN "AuthorizationCodeIssued" が発行される
- THEN "AuthorizationCodeRedeemed" が発行される
- THEN "AccessTokenIssued" が発行される
- THEN "RefreshTokenIssued" が発行される

### REQ-OAUTH2-006: リフレッシュトークンをローテーションして新しいトークンを得る
- ACTOR RegisteredClient
- GIVEN 有効な refresh トークン "RT1" が存在する
- WHEN リフレッシュトークン "RT1" を交換する
  - ALT ローテーション済みの旧 refresh トークンを再使用する → family_id "F1" の refresh トークン "RT1" をローテーション後に再使用する → 再使用はエラー "InvalidGrantError" → "RT1" の状態は "Revoked" → family_id "F1" のトークンがすべて失効する → "RefreshTokenReuseDetected" が発行される → "TokenRevoked" が発行される
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
- GIVEN ユーザー "alice" が "web-app" に scope "openid プロファイル" を同意済みである
- WHEN "web-app" として scope "openid プロファイル" で認可リクエストを送る
  - ALT prompt=同意で再同意を要求する → "web-app" として prompt "同意" で認可リクエストを送る → 認可リクエストの状態は ConsentPending
- THEN 認可リクエストの状態は Consented
- THEN 同意 UI は表示されない

### REQ-OAUTH2-009: PAR で送信した認可リクエストを request_uri 経由で実行する
- ACTOR RegisteredClient
- GIVEN クライアント "web-app" が存在する
- WHEN "web-app" として認可リクエストを事前送信する
  - ALT PAR 必須の FAPI クライアントが PAR なしで直接送信する → PAR 必須の FAPI クライアント "fapi-app" として scope "openid" で直接認可リクエストを送る → エラー "InvalidRequestError"
- THEN PAR 応答に request_uri が含まれ expires_in は 600 以下
- WHEN クライアントが request_uri "<返された値>" で認可リクエストを送る
- THEN その PAR レコードの状態は "Used"
- THEN "PARStored" が発行される
- THEN "AuthorizationCodeIssued" が発行される

### REQ-OAUTH2-010: DPoP 証明付き要求はセンダー制約付きトークンを発行する
- ACTOR RegisteredClient
- WHEN 有効な DPoP 証明を付けて認可コードを交換する
  - ALT iat が 60 秒以上古い DPoP 証明である → iat "2026-01-01T00:00:00Z" の DPoP 証明を時刻 "2026-01-01T00:01:30Z" で付けて認可コードを交換する → エラー "InvalidDpopProofError"
  - ALT 同一 DPoP jti を再使用する → jti "ABC" の DPoP 証明を付けて認可コードを交換する → 同じ jti "ABC" の DPoP 証明を付けて認可コードを交換する → 1 回目の応答には access_token が含まれる → 2 回目はエラー "InvalidDpopProofError"
- THEN 発行された access トークンは DPoP 鍵サムプリントに cnf でバインドされる
- THEN 発行された refresh トークンのセンダー制約は Dpop

### REQ-OAUTH2-011: 失効済みトークンのイントロスペクションは `active=false` だけを返す
- ACTOR ResourceServer
- GIVEN 失効済み access トークン "AT1" が存在する
- WHEN トークン "AT1" を検査する
- THEN 応答は active=false のみで他のフィールドを含まない

### REQ-OAUTH2-012: キルスイッチ作動後の Agent トークンはイントロスペクションで active=false になる
- ACTOR ResourceServer
- GIVEN Agent "A1" に issued_at が古い access トークン "AT1" が発行済みである
- GIVEN "A1" は kill-switch により revocation epoch が "AT1" の issued_at より後へ前進している
- WHEN トークン "AT1" を検査する
  - ALT "AT1" が revocation epoch より後に発行された (kill 後に再発行された) トークンである → 応答は通常どおり active=true と claim を返す
- THEN 応答は active=false のみで他のフィールドを含まない

### REQ-OAUTH2-013: UserInfo は openid スコープのトークンに sub を返す
- ACTOR RegisteredClient
- GIVEN scope "openid プロファイル" の access トークン "AT1" が存在する
- WHEN トークン "AT1" でユーザー情報を取得する
  - ALT openid スコープを持たないトークンで取得する → scope "プロファイル" のみの access トークン "AT1" でユーザー情報を取得する → エラー "InsufficientScopeError"
- THEN レスポンスに `sub`、`name`、`preferred_username` が含まれる

### REQ-OAUTH2-014: Discovery Metadata は宣言された全エンドポイントを広告する
- ACTOR RegisteredClient
- WHEN Discovery Metadata を取得する
- THEN レスポンスに `issuer`、`authorization_endpoint`、`token_endpoint`、`userinfo_endpoint`、`jwks_uri`、`introspection_endpoint`、`revocation_endpoint`、`pushed_authorization_request_endpoint`、`device_authorization_endpoint`、`backchannel_authentication_endpoint`、`registration_endpoint` が含まれる

### REQ-OAUTH2-015: 認可コードの並行交換はちょうど一方だけ成功する
- ACTOR RegisteredClient
- GIVEN 発行済み認可コード "AC1"（family_id "F1"）が存在する
- WHEN 認可コード "AC1" を verifier "v" で並行に 2 回交換する
- THEN ちょうど一方が成功し、もう一方はエラー "InvalidGrantError"
- THEN family_id "F1" のトークンがすべて失効する
- THEN "AuthorizationCodeRedeemed" が発行される
- THEN "RefreshTokenReuseDetected" が発行される
- THEN "TokenRevoked" が発行される

### REQ-OAUTH2-016: 動的クライアント登録は `client_id` を採番して返す
- ACTOR Client
- WHEN confidential クライアント "web-app" を redirect_uri "https://app.example.com/callback" で登録する
  - ALT redirect_uri を持たない登録要求である → confidential クライアント "web-app" を redirect_uri "" で登録する → エラー "InvalidRequestError"
- THEN 応答に client_id と client_secret が含まれる
- THEN "ClientRegistered" が発行される

### REQ-OAUTH2-017: クライアントメタデータの取得では公開 IP へ直接接続する
- ACTOR RegisteredClient
- GIVEN `client_id` は公開 IP に解決される、クライアント所有の HTTPS メタデータ URL である
- GIVEN Authorization Server の環境に HTTPS プロキシが設定されている
- WHEN Authorization Server がクライアントのメタデータ URL を取得する
  - ALT メタデータのホストがプライベート、ループバック、リンクローカル、または CGNAT 100.64.0.0/10 の IP に解決される → Authorization Server は対象 IP へ接続しない → クライアントメタデータの解決をフェイルクローズで拒否する
- THEN 環境のプロキシを使用せず、DNS 検査済みの公開 IP へ直接接続する
- THEN metadata document の取得と検証に成功する

### REQ-OAUTH2-018: 絶対有効期限を過ぎたリフレッシュトークンはローテーションできない
- ACTOR RegisteredClient
- GIVEN absolute_expires_at "2026-01-01T00:00:00Z" の refresh トークン "RT1" が存在する
- GIVEN 現在時刻は "2026-01-02T00:00:00Z" である
- WHEN クライアントがリフレッシュトークン "RT1" を交換する
- THEN エラー "InvalidGrantError"

### REQ-OAUTH2-019: クライアントは自分のトークンを失効できる
- ACTOR RegisteredClient
- GIVEN 有効な refresh トークン "RT1" が存在する
- WHEN トークン "RT1" を失効させる
  - ALT 所有者でないクライアントが失効を要求する → クライアント "client-A" が所有する refresh トークン "RT1" に対し "client-B" として失効を要求する → 盗難検知防止のため 200 OK のみ返り "RT1" の状態は "Active" のまま
- THEN "RT1" の状態は "Revoked"
- THEN "TokenRevoked" が発行される

### REQ-OAUTH2-020: 失効したアクセストークンによる UserInfo の取得は `invalid_token` で拒否される
- ACTOR RegisteredClient
- GIVEN 有効な access トークン "AT1" が存在する
- WHEN トークン "AT1" を失効させる
- THEN トークン "AT1" は失効状態になる
- WHEN クライアントがトークン "AT1" でユーザー情報を取得する
- THEN エラー "InvalidTokenError"

### REQ-OAUTH2-021: リフレッシュトークンは `offline_access` スコープを付与したときだけ発行する
- ACTOR RegisteredClient
- GIVEN confidential クライアント "web-app" が grant_types に "authorization_code"・"refresh_token" を含めて登録済みである
- WHEN "web-app" として scope "openid offline_access" で認可リクエストを送る
  - ALT offline_access を要求しない → "web-app" として scope "openid プロファイル" で認可リクエストを送る → 発行された認可コードを verifier "v" で交換する → 応答に refresh_token は含まれない
- WHEN クライアントが発行された認可コードを verifier "v" で交換する
- THEN 応答に refresh_token が含まれる
- THEN "RefreshTokenIssued" が発行される

### REQ-OAUTH2-022: 認可リクエストの nonce は ID トークンに伝播する
- ACTOR RegisteredClient
- WHEN "web-app" として scope "openid"、nonce "n-12345" で認可リクエストを送る
- WHEN クライアントが発行された認可コードを verifier "v" で交換する
- THEN 応答の id_token の nonce クレームは "n-12345"

### REQ-OAUTH2-023: RP-Initiated Logout は登録済み post_logout_redirect_uri にだけ戻す
- ACTOR ResourceOwner
- GIVEN confidential クライアント "web-app" が redirect_uri "https://app.example.com/cb" で登録済みである
- WHEN "web-app" として post_logout_redirect_uri "https://app.example.com/cb" でログアウトする
  - ALT 未登録の post_logout_redirect_uri を指定する → "web-app" として post_logout_redirect_uri "https://evil.example.com/cb" でログアウトする → エラー "InvalidRequestError"
- THEN `state` が `post_logout_redirect_uri` に伝播する

### REQ-OAUTH2-024: RP-Initiated Logout は id_token_hint からセッションとクライアントを特定する
- ACTOR ResourceOwner
- GIVEN ユーザー "alice" が "web-app" として認可コードを交換し、`sid` 付きの ID Token を持つ
- WHEN "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
  - ALT `id_token_hint` の `aud` が指定された `client_id` と一致しない → `client_id` "other-app" と "web-app" 発行の ID Token を `id_token_hint` に付けて `/end_session` を呼ぶ → エラー "InvalidRequestError"
  - ALT `id_token_hint` の署名を IdMagic の署名鍵で検証できない → 他の発行者が署名した JWT を `id_token_hint` に付けて `/end_session` を呼ぶ → エラー "InvalidRequestError"
  - ALT id_token_hint が期限切れ (exp 経過) である → 期限切れの発行済み ID Token を id_token_hint として /end_session を呼ぶ → exp 切れのみを理由にした拒否はされず sid によるセッション解決が成功する
- THEN id_token_hint の sid が示す LoginSession が失効する
- THEN 同じ sid を持つ全クライアントの RefreshTokenRecord が Revoked へ遷移する

### REQ-OAUTH2-025: セッション失効時は `backchannel_logout_uri` を登録済みの RP へログアウトトークンを配信する
- ACTOR ResourceOwner
- GIVEN "web-app" が backchannel_logout_uri "https://app.example.com/backchannel_logout" を登録済みである
- GIVEN ユーザー "alice" が "web-app" とのブラウザセッションを持つ
- WHEN "alice" が /end_session でログアウトする
- THEN 対象 sid と "web-app" の LogoutNotification が作成される
- THEN 署名済みのログアウトトークンが `backchannel_logout_uri` へ配信され、`Delivered` になる
  - ALT 配送が一時的に失敗する (5xx / timeout) → LogoutNotification は Pending のまま attempts が増え再試行され、ローカルのセッション/refresh トークン失効は取り消されない
  - ALT max_attempts まで再試行しても配送が成功しない → LogoutNotification は Failed (dead-letter) に確定し、ローカルのセッション/refresh トークン失効は取り消されない

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
- WHEN "tv-app" として scope "openid プロファイル" でデバイス認可を開始する
- THEN 応答に device_code・user_code・verification_uri・interval が含まれる
- WHEN ユーザー "alice" が verification_uri で user_code を入力し承認する
- THEN device authorization は承認済みになる
- WHEN クライアントが device_code "DC1" を交換する
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
- WHEN クライアントが mTLS 証明書を提示して access_token を要求する
- THEN access_token は提示された証明書にバインドされる
- WHEN クライアントが証明書にバインドされた access_token で userinfo を取得する
  - ALT 別の証明書を提示する → invalid_token で拒否される
- THEN 同じ証明書を提示した要求は 200 を返す

### REQ-OAUTH2-030: RFC 8414 メタデータ文書は OIDC Discovery と同等の内容を返す
- ACTOR RegisteredClient
- WHEN Authorization Server メタデータを取得する
- THEN レスポンスに `issuer`、`authorization_endpoint`、`token_endpoint`、`jwks_uri`、`grant_types_supported` が含まれる

### REQ-OAUTH2-031: 管理者は所属テナントの同意を参照・撤回できるが付与は代行できない
- ACTOR TenantAdministrator
- GIVEN tenant_id "acme" のロール=["admin"] のユーザー "operator" が認証済みである
- GIVEN tenant_id "acme" のユーザー "alice" とクライアント "portal" の Consent が Granted で存在する
- WHEN 管理者 "operator" が Consent 一覧と単一 Consent を取得する
- THEN 所属テナントの Consent だけが返る
- WHEN 管理者 "operator" がユーザー "alice" とクライアント "portal" の Consent を撤回する
- THEN `Consent.state` は `Revoked` となり、`revoked_at` が記録される
- THEN "ConsentRevoked" が actorUserId "operator" で発行される
- THEN 管理者が Consent を作成または scope 拡張する interface は存在しない

### REQ-OAUTH2-032: ユーザーは接続済みアプリの同意を自分で撤回できる
- ACTOR ResourceOwner
- GIVEN ユーザー "alice" がクライアント "web-app" に scope "openid プロファイル" を同意済みである
- GIVEN ユーザー "alice" が認証済みで接続済みアプリ画面を開いている
- WHEN ユーザー "alice" が接続済みアプリ一覧を取得する
- THEN 一覧に "web-app" が表示される
- WHEN ユーザー "alice" が "web-app" の同意を撤回する
- THEN `Consent.state` は `Revoked` となり、一覧から消える

### REQ-OAUTH2-033: realm 接頭辞付きの Discovery Metadata は同じ接頭辞を持つ発行者を返す
- ACTOR RegisteredClient
- GIVEN tenant_id "acme" が Active で存在する
- WHEN /realms/acme/.well-known/openid-configuration を取得する
- THEN レスポンスの `issuer` は基底 URL + `/realms/acme` となる
- THEN レスポンスの `authorization_endpoint` は基底 URL + `/realms/acme/authorize` となる

### REQ-OAUTH2-034: トークンエンドポイントはテナント境界を越えた資格情報を受理しない
- ACTOR RegisteredClient
- WHEN tenant_id "acme" で発行した認可コード "AC1" を "/realms/default/token" で交換する
  - ALT 他テナントの client_id を使う → tenant_id "acme" に登録した client_id "web-app" で "/realms/default/token" に交換を要求する → エラー "InvalidClientError"
  - ALT 他テナントのリフレッシュトークンを再発行する → tenant_id "acme" で発行した refresh トークン "RT1" を "/realms/default/token" で再発行する → エラー "InvalidGrantError"
  - ALT 他テナントの device_code を交換する → tenant_id "acme" で発行し承認した device_code "DC1" を "/realms/default/token" で交換する → エラー "InvalidGrantError"
- THEN エラー "InvalidGrantError"
- WHEN 永続化層へ `tenant_id=acme` のリフレッシュトークンと `tenant_id=default` の `client_id` または `sub` を書き込む
- THEN 永続化層が参照整合性エラーで拒否する

### REQ-OAUTH2-035: 管理者は所属テナントのクライアントを作成・更新・削除できる
- ACTOR TenantAdministrator
- GIVEN tenant_id "acme" の admin "operator" が認証済みである
- WHEN "operator" がクライアント "portal" を作成する
- THEN client_secret が一度だけ返る
- WHEN "operator" がクライアント "portal" を取得する
  - ALT 別テナントの管理者が同じ client_id を指定する → InvalidRequestError で拒否される
- THEN 所属テナントのクライアントだけが返る
- WHEN "operator" がクライアント "portal" の redirect_uris を更新する
- THEN redirect_uris が保存される
- WHEN "operator" がクライアント "portal" を削除する
- THEN "AdminOAuth2ClientCreated"、"AdminOAuth2ClientUpdated"、"AdminOAuth2ClientDeleted" が発行される

### REQ-OAUTH2-036: 管理者は Application から期限付きクライアントシークレットを追加発行し、個別に失効できる
- ACTOR TenantAdministrator
- GIVEN `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- GIVEN 有効期限のない従来のシークレット "S1" が `Active` である
- WHEN 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
  - ALT `expires_in_days` が 1..730 の範囲外である → エラー "InvalidRequestError"
  - ALT `Active` の資格情報がすでに 2 件存在する → 追加発行をエラー "ClientSecretLimitExceededError" で拒否し、既存の資格情報は変更しない
  - ALT クライアントが `private_key_jwt`、mTLS、または公開クライアントである → エラー "InvalidRequestError"
- THEN レスポンスで新しいシークレットを一度だけ受け取り、メタデータは 90 日後の `expires_at` と `Active` ステータスを持つ
- THEN 追加発行によって既存シークレットの期限とステータスは変わらない
- THEN 新旧両方のシークレットでトークンエンドポイントの認証に成功する
- WHEN 管理者が以前の資格情報だけを個別に失効する
  - ALT 別クライアントの `credential_id` または存在しない `credential_id` を失効する → エラー "InvalidRequestError"
  - ALT すでに `Revoked` の資格情報を再び失効する → 冪等に成功し、ClientSecretRevoked は重複発行されない
- THEN 以前のシークレットは InvalidClientError で拒否され、新しいシークレットでは引き続き認証に成功する
- THEN ClientSecretIssued と ClientSecretRevoked は、`actor`、クライアント、`credential`、`expiry` の非機密メタデータだけを含んで発行される

### REQ-OAUTH2-037: 管理者は互換インターフェースからクライアントシークレットを無停止でローテーションできる
- ACTOR TenantAdministrator
- GIVEN `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- GIVEN 以前のシークレット "S1" が有効である
- WHEN 管理者が `grace_days=7` でシークレットをローテーションする
  - ALT `grace_days` が 0 以外で 1..30 の範囲外である → エラー "InvalidRequestError"
  - ALT クライアントが `private_key_jwt`、mTLS、または公開クライアントである → エラー "InvalidRequestError"
- THEN レスポンスで新しいシークレットを一度だけ受け取る
- THEN 新旧両方のシークレットは `grace_until` より前にトークンエンドポイントの認証に成功する
- THEN `grace_until` より後は以前のシークレットが InvalidClientError で拒否される
- THEN ClientSecretRotated は `actor`、クライアント、`grace_until` だけを含んで発行される

### REQ-OAUTH2-038: 同意管理 API は別テナントの同意を公開しない
- ACTOR TenantAdministrator
- GIVEN tenant_id "acme" のユーザーとクライアントの Consent が存在する
- WHEN `tenant_id=default` の管理者が同じ `user_id` と `client_id` の Consent を取得する
- THEN エラー "InvalidRequestError"

### REQ-OAUTH2-039: KeyProvider の障害時は新しいトークンの発行を拒否する
- ACTOR RegisteredClient
- GIVEN tenant_id "acme" の KeyProvider が到達不能である
- WHEN tenant_id "acme" のクライアントがトークン発行を要求する
- THEN 新規署名は行われずエラー "ServerError" で拒否される

### REQ-OAUTH2-040: プロトコルエンドポイントは閾値を超えたリクエストをレート制限で拒否する
- ACTOR RegisteredClient
- GIVEN クライアントがある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している
- WHEN 同一 window 内で追加リクエストを送る
  - ALT 対象 endpoint が /tokenである → client_id と IP の組で閾値超過している状態でトークンを要求する → エラー "RateLimitedError"
  - ALT 対象 endpoint が /authorize または /par である → IP と client_id の組で閾値超過している状態で認可リクエストを送る → エラー "RateLimitedError"
  - ALT 対象 endpoint が /device_authorization である → client_id と IP の組で閾値超過している状態でデバイス認可を開始する → エラー "RateLimitedError"
  - ALT 対象 endpoint が /bc-authorize である → client_id と IP の組で閾値超過している状態で backchannel 認可を開始する → エラー "RateLimitedError"
  - ALT 共有カウンタストアに到達できない → リクエストは fail-closed で "RateLimitedError" として拒否される
- THEN エラー "RateLimitedError" (HTTP 429、Retry-After ヘッダ付き)

### REQ-OAUTH2-041: バックチャネル認可要求は人間の承認が成立してからトークンを発行する
- ACTOR RegisteredClient
- GIVEN `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- GIVEN active User "alice" が存在する
- WHEN `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
  - ALT scope が未指定または openid を含まない → エラー "InvalidScopeError"
  - ALT login_hint と id_token_hint が両方指定される、または両方未指定である → エラー "InvalidRequestError"
  - ALT requested_expiry が非正または 600 秒を超える → エラー "InvalidRequestError"
  - ALT binding_message が 64 文字を超える、または制御文字を含む → エラー "InvalidBindingMessageError"
  - ALT login_hint が User を解決できない → 未知の login_hint "nobody" で backchannel 認可を開始する → エラー "UnknownUserIdError"
  - ALT login_hint の User が別テナントまたは非 active である → エラー "UnknownUserIdError"
  - ALT クライアントの許可 scope に含まれない scope を要求する → エラー "InvalidScopeError"
- THEN 応答に auth_req_id・expires_in・interval が含まれる
- THEN 承認要求の状態は Pending になる
- THEN "BackchannelAuthRequested" が発行される
- WHEN `agent-app` が `auth_req_id=AR1` を交換する
  - ALT ユーザー判断前にポーリングする → Pending 状態の auth_req_id "AR1" を交換する → エラー "AuthorizationPendingError"
  - ALT interval より短い間隔で再試行する → interval 5 秒の auth_req_id "AR1" を交換し "2s" 経過後に再度交換する → 2 回目はエラー "SlowDownError"
- WHEN ユーザー "alice" が承認要求 "AR1" を承認する
- THEN 承認要求 "AR1" の状態は Approved になる
- THEN "BackchannelAuthApproved" が発行される
- WHEN `agent-app` が `auth_req_id=AR1` を交換する
- THEN 応答に access_token と id_token が含まれ sub は "alice"、scope は要求した "openid" になる
- THEN 承認要求 "AR1" の状態は Consumed になる
- THEN "AccessTokenIssued" が発行される

### REQ-OAUTH2-042: 承認が成立していない承認要求はトークンを発行しない
- ACTOR RegisteredClient
- GIVEN `agent-app` が起票した承認要求 `AR1` が存在する
- WHEN `agent-app` が `auth_req_id=AR1` を交換する
  - ALT ユーザーが "AR1" を拒否済みである → エラー "AccessDeniedError" → 承認要求 "AR1" の状態は Denied のままになる
  - ALT "AR1" が expires_at を過ぎている → requested_at "2026-01-01T00:00:00Z"・expires_at "2026-01-01T00:05:00Z" の "AR1" を時刻 "2026-01-01T00:06:00Z" で交換する → エラー "ExpiredTokenError" → 承認要求 "AR1" の状態は Expired になる
  - ALT 承認済みの "AR1" を 2 回交換する → 1 回目の応答には access_token が含まれる → 2 回目はエラー "InvalidGrantError" → 承認要求 "AR1" の状態は Consumed のままになる
  - ALT 承認済みの "AR1" を並行に 2 回交換する → ちょうど一方が成功し、もう一方はエラー "InvalidGrantError"
  - ALT 起票元でないクライアント "other-app" が "AR1" を交換する → エラー "InvalidGrantError"
  - ALT 別テナントのトークン endpoint で "AR1" を交換する → エラー "InvalidGrantError"
  - ALT "AR1" の承認後に Agent が kill-switch で停止されている → エラー "InvalidGrantError" → トークンは発行されない
- THEN 承認要求が Approved のときだけ応答に access_token が含まれる

### REQ-OAUTH2-043: 承認要求を判断できるのは対象ユーザー本人のステップアップ認証済みセッションだけである
- ACTOR ResourceOwner
- GIVEN ユーザー "alice" 宛の承認要求 "AR1" が Pending で存在する
- GIVEN ユーザー "bob" 宛の承認要求 "AR2" が Pending で存在する
- GIVEN "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- WHEN "alice" が保留中の承認要求一覧を取得する
- THEN 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- THEN 一覧に "AR2" と期限切れの承認要求は含まれない
- WHEN "alice" が承認要求 "AR1" を承認する
  - ALT ステップアップ認証の有効期間を過ぎている → 操作を AccessDeniedError で拒否し、承認要求 "AR1" の状態は Pending のままとなる
  - ALT CSRF トークンが一致しない → 操作は拒否され承認要求 "AR1" の状態は Pending のままになる
  - ALT "alice" が他人宛の承認要求 "AR2" を判断する → 操作は AccessDeniedError で拒否される
  - ALT 既に終端状態の承認要求を判断する → 操作は InvalidRequestError で拒否され、記録済みの判断は上書きされない
- THEN 承認要求 "AR1" の状態は Approved になる
- THEN "BackchannelAuthApproved" が発行される

### REQ-OAUTH2-044: Bearer 保護リソースの認証エラーはメタデータ URL を提示する
- ACTOR RegisteredClient
- GIVEN レルム "acme" の発行者は "https://idp.example.com/realms/acme" である
- WHEN クライアントが無効なアクセストークンでレルム "acme" の保護 API を呼ぶ
  - ALT アクセストークンに必要なスコープがない → HTTP 403 を返し、WWW-Authenticate に `error="insufficient_scope"` と必要なスコープ、および `resource_metadata="https://idp.example.com/realms/acme/.well-known/oauth-protected-resource"` を含める
  - ALT レルム "acme" がホストルート形式のエンドポイントを使う → `resource_metadata` はホストルートの発行者配下にある `/.well-known/oauth-protected-resource` を指す
- THEN HTTP 401 を返し、WWW-Authenticate は Bearer の `error="invalid_token"` と `resource_metadata="https://idp.example.com/realms/acme/.well-known/oauth-protected-resource"` を引用符付きの auth-param として含む
- THEN `resource_metadata` URL は、`resource` 未指定時のレルムの IdMagic API Protected Resource Metadata を返す

### REQ-OAUTH2-045: 保護リソースの DPoP Proof は ath でアクセストークンに結び付けられる
- ACTOR RegisteredClient
- GIVEN DPoP 鍵 "K1" に結び付けられたアクセストークン "AT1" と "AT2" が存在する
- WHEN クライアントが "AT1" を提示し、`ath` が "AT1" の base64url(SHA-256) である "K1" 署名の DPoP Proof で保護リソースを呼ぶ
  - ALT Proof が `ath` を含まない → エラー "InvalidTokenError"
  - ALT Proof の `ath` が "AT2" の base64url(SHA-256) である → エラー "InvalidTokenError"
  - ALT トークンエンドポイントへ `ath` を含まない "K1" 署名の DPoP Proof を提示する → リクエストは受理され、アクセストークンが発行される
- THEN 要求は受理される

### REQ-OAUTH2-046: 所有者がオフボードされた Agent は client_credentials で新しいトークンを取得できない
- ACTOR RegisteredClient
- GIVEN User "owner1" が所有する `Active` の Agent "A1" が confidential クライアント "agent-app" に束縛されている
- GIVEN 管理者が "owner1" を無効化した
- WHEN "agent-app" として client_credentials でトークンを要求する
  - ALT "owner1" がハード削除され解決できない → エラー "InvalidClientError" → トークンは発行されない
  - ALT "owner1" が再び有効化された → リクエストは受理され、アクセストークンが発行される
  - ALT "agent-app" にどの Agent も束縛されていない → 所有者の解決を行わず、リクエストは受理される
- THEN エラー "InvalidClientError" で拒否され、トークンは発行されない
- THEN 所有者の状態は "A1" の `status` を書き換えず、発行のたびに解決する（"A1" は `Active` のまま）

### REQ-OAUTH2-047: 管理コンソールとアカウントポータルの Bearer 認証も失効判定を通る
- ACTOR ResourceServer
- GIVEN Agent "A1" に束縛されたクライアントへ発行済みの access トークン "AT1" がある
- GIVEN "A1" の revocation epoch が "AT1" の issued_at より後へ前進している
- WHEN "AT1" を Bearer として `/api/admin/v1/` 配下の API へ提示する
  - ALT "AT1" の jti が失効リストに載っている → エラー "InvalidTokenError"
  - ALT "AT1" が revocation epoch より後に発行されている → 認証は成立し、以後はスコープとロールの境界で判定する
- THEN エラー "InvalidTokenError" で拒否される（イントロスペクションと同じ失効判定を通す）

### REQ-OAUTH2-048: テナントが定めた委譲深さの上限を超えるトークン交換は拒否される
- ACTOR Agent
- GIVEN テナントが委譲深さの上限を設定している、または上書きを持たずシステム既定を継承している
- WHEN エージェントが Token Exchange で委任トークンを要求する
  - ALT 発行トークンの `act` 入れ子の深さが上限以内である → 交換は成立する
  - ALT 深さが上限を超える → 交換を拒否し、拒否理由を監査へ残す
  - ALT テナントの委譲ポリシーを解決できない → システム既定へ退避せず拒否する
- THEN 監査イベントは発行トークンの深さと、判定に適用した上限の双方を残す

### REQ-OAUTH2-049: イントロスペクションと監査は同じ規則で委譲モードを示す
- ACTOR ResourceServer
- GIVEN Token Exchange で発行した委任トークンがある
- WHEN リソースサーバーがそのトークンをイントロスペクトする
  - ALT `act` に subject と異なる行為者がいる → 利用者の代理として返す
  - ALT 代行が無く subject が非人間のプリンシパルである → 自律実行として返す
  - ALT 代行が無く subject が人間の利用者である → 直接のアクセスとして返す
- THEN 応答の委譲モードは、同じ交換が監査へ残したモードと一致する
- THEN リソースサーバーは `act` と principal 種別から導出し直す必要がない
