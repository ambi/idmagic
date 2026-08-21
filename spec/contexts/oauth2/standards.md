# OAuth2 Standards

## The OAuth 2.0 Authorization Framework

RFC 6749 — https://www.rfc-editor.org/rfc/rfc6749.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6749-AUTHORIZATION-CODE | required | MUST | Authorization Code Grant を認可エンドポイントとトークンエンドポイントで提供し、単一値であるべきセキュリティパラメーターが認可リクエスト内で重複していれば `invalid_request` として拒否する。 |
| RFC6749-CLIENT-CREDENTIALS | optional | MAY | Client Credentials Grant は `confidential` クライアントに限って許可する。 |
| RFC6749-IMPLICIT | excluded | MAY | Implicit Grant を提供する。 |
| RFC6749-PASSWORD-GRANT | excluded | MAY | Resource Owner Password Credentials Grant を提供する。 |

## The OAuth 2.0 Authorization Framework Bearer Token Usage

RFC 6750 — https://www.rfc-editor.org/rfc/rfc6750.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6750-AUTHORIZATION-HEADER | required | MUST | ベアラーアクセストークンは `Authorization` ヘッダーで受け付ける。 |
| RFC6750-QUERY-TOKEN | excluded | MAY | URI のクエリパラメーターによるアクセストークンの提示を受け付ける。 |

## OAuth 2.0 Token Revocation

RFC 7009 — https://www.rfc-editor.org/rfc/rfc7009.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7009-REVOCATION-ENDPOINT | required | MUST | 認証済みクライアントへトークン失効エンドポイントを提供する。 |
| RFC7009-UNKNOWN-TOKEN | required | MUST | 無効または他クライアント所有のトークンに対しても成功応答を返し情報を漏らさない。 |

## JSON Web Key

RFC 7517 — https://www.rfc-editor.org/rfc/rfc7517.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7517-JWKS | required | MUST | 公開可能な検証鍵を JWK Set として配布する。 |

## JSON Web Algorithms

RFC 7518 — https://www.rfc-editor.org/rfc/rfc7518.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7518-SIGNATURE-ALGORITHMS | required | MUST | JWT の署名には PS256 または ES256 を使い、アクセストークンと ID トークンには対称鍵署名を使わない。 |

## JSON Web Token

RFC 7519 — https://www.rfc-editor.org/rfc/rfc7519.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7519-REGISTERED-CLAIMS | required | MUST | JWT の発行者、subject、audience、有効期限、発行時刻、一意識別子を用途に応じて検証または発行する。 |

## JSON Web Token Profile for OAuth 2.0 Client Authentication and Authorization Grants

RFC 7523 — https://www.rfc-editor.org/rfc/rfc7523.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7523-CLIENT-ASSERTION | optional | MAY | クライアントアサーションの署名、発行者、subject、audience、有効期限、`jti` を検証する。 |

## OAuth 2.0 Dynamic Client Registration Protocol

RFC 7591 — https://www.rfc-editor.org/rfc/rfc7591.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7591-REGISTER | optional | MAY | クライアントメタデータを受け取り、`client_id` と登録結果を返す。 |
| RFC7591-REDIRECT-URI | required | MUST | Authorization Code Grant を利用するクライアントには `redirect_uri` の登録を要求する。 |

## OAuth Client ID Metadata Document

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

## Proof Key for Code Exchange by OAuth Public Clients

RFC 7636 — https://www.rfc-editor.org/rfc/rfc7636.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7636-VERIFY | required | MUST | トークンリクエストの `code_verifier` を認可時の `code_challenge` と照合し、認可リクエスト内で PKCE パラメーターが重複していれば拒否する。 |
| RFC7636-S256 | required | SHOULD | `code_challenge_method` は `S256` だけを許可する。 |
| RFC7636-PLAIN | excluded | MAY | `plain` 方式のコードチャレンジを許可する。 |

## OAuth 2.0 Token Introspection

RFC 7662 — https://www.rfc-editor.org/rfc/rfc7662.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7662-INTROSPECT | required | MUST | 認証済みのリソースサーバーへ `active` ステータスと許可されたメタデータを返す。 |
| RFC7662-INACTIVE | required | MUST | 無効なトークンには `active=false` だけを返す。 |

## Proof-of-Possession Key Semantics for JSON Web Tokens

RFC 7800 — https://www.rfc-editor.org/rfc/rfc7800.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7800-CONFIRMATION | optional | MAY | 送信者制約付きトークンの確認鍵情報を `cnf` クレームに格納する。 |

## Authentication Method Reference Values

RFC 8176 — https://www.rfc-editor.org/rfc/rfc8176.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8176-AMR | required | MAY | 実際に成立した認証方法を `amr` 値として記録する。 |

## OAuth 2.0 Authorization Server Metadata

RFC 8414 — https://www.rfc-editor.org/rfc/rfc8414.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8414-METADATA | required | MUST | 発行者と利用可能なエンドポイントおよび機能を Authorization Server Metadata として公開する。 |

## OAuth 2.0 Device Authorization Grant

RFC 8628 — https://www.rfc-editor.org/rfc/rfc8628.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8628-DEVICE-AUTHORIZATION | optional | MUST | `device_code`、`user_code`、`verification_uri` を発行し、ResourceOwner の判断を受け付ける。 |
| RFC8628-POLLING | required | MUST | `authorization_pending`、`slow_down`、`expired_token` のポーリングセマンティクスを守る。 |

## OAuth 2.0 Mutual-TLS Client Authentication and Certificate-Bound Access Tokens

RFC 8705 — https://www.rfc-editor.org/rfc/rfc8705.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8705-CLIENT-AUTH | optional | MAY | 登録済みの Subject DN と検証済みクライアント証明書を照合してクライアントを認証する。 |
| RFC8705-CERT-BOUND | optional | MAY | アクセストークンをクライアント証明書のサムプリントへバインドし、リソースへのアクセス時に照合する。 |

## Resource Indicators for OAuth 2.0

RFC 8707 — https://www.rfc-editor.org/rfc/rfc8707.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8707-AUDIENCE | required | SHOULD | 発行するアクセストークンに空でない `audience` を設定し、意図しないリソースサーバーでの利用を防ぐ。 |
| RFC8707-MCP-RESOURCE-BINDING | required | MUST | `resource` パラメーターで指定された `McpResourceServer` にアクセストークンの `audience` を厳格に限定し、未登録、無効、複数指定の `resource` は fail-closed で拒否する。認可、Pushed Authorization Requests、トークン発行（認可コードの交換、リフレッシュトークンのローテーション、`client_credentials`、`device_code`、トークン交換）の全経路へ一様に適用する。`resource` が未指定であれば `client_id` を `audience` とする。 |

## OAuth 2.0 Token Exchange

RFC 8693 — https://www.rfc-editor.org/rfc/rfc8693.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8693-DELEGATION-DEFAULT | required | MUST | 交換の既定は委譲とする。発行するトークンは元のユーザーを `sub` に保ち、現在の行為者を `act` に記録し、以前の行為者を §4.1 に従って内側へ入れ子にする。 |
| RFC8693-IMPERSONATION | optional | MAY | なりすまし (`act` を落として `sub` を置き換える形) は、クライアントまたは Agent へ明示的に許可した場合だけ受け付ける。 |
| RFC8693-SUBJECT-TOKEN | required | MUST | 受け付ける `subject_token` は、自身が発行しイントロスペクションを通過したトークンか、`subject_token_type` が JWT-SVID の登録済み外部アテステーションに限る。 |
| RFC8693-DELEGATION-DEPTH | required | MUST | `act` チェーンの長さをテナントの実効委譲深さで制限する。テナントはシステム既定を下げられるが上げられず、ポリシーを解決できない場合は交換を拒否する。 |

## OAuth 2.0 Rich Authorization Requests

RFC 9396 — https://www.rfc-editor.org/rfc/rfc9396.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9396-REGISTERED-TYPES | required | MUST | `authorization_details` は、テナントが事前登録した `type` とそのスキーマに対して検証する。未登録の型やスキーマの不一致は部分的に受理せず拒否する。 |
| RFC9396-MONOTONIC-NARROWING | required | MUST | 発行または交換するトークンが持てるのは、同意した権限の部分集合に限る。後続の交換は権限を狭めることだけを許し、広げる要求は拒否する。 |
| RFC9396-SCOPE-PRECEDENCE | required | MUST | 同じ領域で `type` と粗い `scope` が重なる場合は構造化された詳細の上限を優先し、`authorization_details` で制限した領域を `scope` が再び広げる要求は拒否する。 |

## JSON Web Token Profile for OAuth 2.0 Access Tokens

RFC 9068 — https://www.rfc-editor.org/rfc/rfc9068.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9068-CLAIMS | required | MUST | JWT アクセストークンに `iss`、`sub`、`aud`、`exp`、`iat`、`jti`、`client_id` を含める。 |
| RFC9068-ASYMMETRIC-SIGNATURE | required | MUST | アクセストークンを非対称アルゴリズムで署名し、公開鍵で検証可能にする。 |

## OAuth 2.0 Pushed Authorization Requests

RFC 9126 — https://www.rfc-editor.org/rfc/rfc9126.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9126-PAR | optional | MAY | クライアント認証済みの PAR を保存し、短命な `request_uri` を返す。 |
| RFC9126-SINGLE-USE | required | SHOULD | `request_uri` は短命かつ一度だけ使用可能とする。 |

## OAuth 2.0 Authorization Server Issuer Identification

RFC 9207 — https://www.rfc-editor.org/rfc/rfc9207.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9207-ISS | required | MUST | 認可レスポンス、および安全に確定した `redirect_uri` へ返す認可エラーに発行者の識別子を含める。 |

## OAuth 2.0 Demonstrating Proof of Possession

RFC 9449 — https://www.rfc-editor.org/rfc/rfc9449.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9449-PROOF | optional | MAY | DPoP proof の署名、`htm`、`htu`、`iat`、`jti` を検証する。 |
| RFC9449-TOKEN-BINDING | optional | MUST | DPoP 利用時はアクセストークンに `jkt` を含め、提示された proof の鍵と照合する。 |
| RFC9449-ATH | optional | MUST | 保護リソースへ提示する DPoP proof は `ath` を含み、`base64url(SHA-256(access_token))` と一致しなければ拒否する。 |

## Best Current Practice for OAuth 2.0 Security

RFC 9700 / BCP 240 — https://www.rfc-editor.org/rfc/rfc9700.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9700-REDIRECT-MATCH | required | MUST | `redirect_uri` は登録値と完全に一致させ、未検証の URI へリダイレクトしない。 |
| RFC9700-AUTHORIZATION-CODE | required | MUST | リダイレクトを使うフローは Authorization Code Grant と PKCE で保護する。 |
| RFC9700-SENDER-CONSTRAINT | optional | SHOULD | 高いセキュリティを必要とするクライアントでは、アクセストークンに DPoP または mTLS による送信者制約を付ける。 |
| RFC9700-REFRESH-REPLAY | required | MUST | リフレッシュトークンをローテーションし、再利用を検知したら関連トークンを失効させる。 |

## OAuth 2.0 Protected Resource Metadata

RFC 9728 — https://www.rfc-editor.org/rfc/rfc9728.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC9728-METADATA | required | MUST | 登録済みの `McpResourceServer` ごとに、対象リソースに対応する `authorization_servers` と対応スコープを含む Protected Resource Metadata を配信する。 |
| RFC9728-WELL-KNOWN | required | MUST | `/.well-known/oauth-protected-resource` で `resource` を指定して Protected Resource Metadata を取得できるようにする。 |
| RFC9728-IDMAGIC-API | required | MUST | `resource` が未指定であれば、realm の IdMagic API に対する Protected Resource Metadata と `account`、`management`、SCIM の各スコープ、対応する `bearer_methods_supported` を公開する。 |
| RFC9728-CHALLENGE | required | MUST | ベアラー保護リソースの `401 invalid_token` と `403 insufficient_scope` レスポンスでは、当該 realm の Protected Resource Metadata URL を `resource_metadata` 認証パラメーターで提示する。 |

## OpenID Connect Core 1.0 incorporating errata set 1

Final — https://openid.net/specs/openid-connect-core-1_0-18.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-CORE-CODE-FLOW | required | MUST | `code` レスポンスタイプによる OpenID Connect Authentication を提供し、`prompt` を空白区切りの重複しないトークン集合として検証する。`login` と `consent` をそれぞれ適用し、`none` はほかのトークンと併用せず UI を表示しない。 |
| OIDC-CORE-ID-TOKEN | required | MUST | ID トークンに `iss`、`sub`、`aud`、`exp`、`iat` と認証コンテキストを含める。 |
| OIDC-CORE-USERINFO | required | SHOULD | `openid` スコープのアクセストークンに対して、`sub` を含む UserInfo を返す。 |
| OIDC-CORE-HYBRID-IMPLICIT | excluded | MAY | Implicit Flow および Hybrid Flow を提供する。 |

## OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0

Final — https://openid.net/specs/openid-client-initiated-backchannel-authentication-core-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| CIBA-CORE-BACKCHANNEL-REQUEST | optional | MUST | クライアント認証済みのバックチャネル認証リクエストを受け付ける。`scope` は必須で `openid` を含み、`login_hint` または `id_token_hint` のちょうど一方から承認対象の User を解決し、`auth_req_id`、`expires_in`、`interval` を返す。解決できなければ `unknown_user_id` で拒否する。 |
| CIBA-CORE-POLL-MODE | optional | MUST | トークンエンドポイントの CIBA グラントで `authorization_pending`、`slow_down`、`access_denied`、`expired_token` のポーリングセマンティクスを守り、承認成立後の `auth_req_id` をちょうど一度だけトークン化する。 |
| CIBA-CORE-BINDING-MESSAGE | optional | SHOULD | `binding_message` を承認画面に表示し、クライアント、要求スコープ、`authorization_details` と併せて承認内容を示す。 |
| CIBA-CORE-PING-PUSH | excluded | MAY | `ping` および `push` のトークン配信モードを提供する。 |
| CIBA-CORE-USER-CODE | excluded | MAY | `user_code` パラメーターによる認証デバイス側の本人確認補助を受け付ける。 |
| CIBA-CORE-SIGNED-REQUEST | excluded | MAY | 署名済み JWT によるバックチャネル認証リクエストを受け付ける。 |

## OpenID Connect Discovery 1.0 incorporating errata set 1

Final — https://openid.net/specs/openid-connect-discovery-1_0-21.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-DISCOVERY-CONFIGURATION | required | MUST | well-known 設定から発行者、エンドポイント、対応機能を Discovery Metadata として公開する。 |

## OpenID Connect RP-Initiated Logout 1.0

Final — https://openid.net/specs/openid-connect-rpinitiated-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-LOGOUT-ENDPOINT | required | MUST | `end_session_endpoint` を公開し、RP からのログアウトリクエストを受け付ける。 |
| OIDC-LOGOUT-REDIRECT | required | MUST | `post_logout_redirect_uri` はクライアントに登録済みの値だけを許可する。 |
| OIDC-LOGOUT-ID-TOKEN-HINT | required | SHOULD | `id_token_hint` が与えられた場合は、署名、発行者、audience、subject、`sid` を検証してログアウト対象のセッションとクライアントを解決し、`client_id` パラメーターと矛盾するヒントを拒否する。 |

## OpenID Connect Front-Channel Logout 1.0

Final — https://openid.net/specs/openid-connect-frontchannel-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-FRONTCHANNEL-IFRAME | required | MUST | `end_session` レスポンスに、`frontchannel_logout_uri` を登録した各クライアントへの `iframe` を含める。`frontchannel_logout_session_required=true` のクライアントには `iss` と `sid` のクエリパラメーターを付与する。 |
| OIDC-FRONTCHANNEL-BEST-EFFORT | required | MUST | `iframe` の到達失敗を許容し、ローカルセッションの失効結果に影響させない。 |

## OpenID Connect Back-Channel Logout 1.0

Final — https://openid.net/specs/openid-connect-backchannel-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-BACKCHANNEL-LOGOUT-TOKEN | required | MUST | ログアウトトークンに `iss`、`sub`、`aud`、`iat`、`jti`、イベント（`http://schemas.openid.net/event/backchannel-logout`）を含め、対象がブラウザーセッションに由来する場合は `sid` も含めて署名する。 |
| OIDC-BACKCHANNEL-DELIVERY-RETRY | required | MUST | `backchannel_logout_uri` への配信失敗を再試行可能なジョブとして扱い、ローカルセッションやリフレッシュトークンの失効を配信結果に依存させない。 |
| OIDC-BACKCHANNEL-REPLAY | required | MUST | ログアウトトークンの `jti` により RP 側でリプレイを検出できる一意な値を発行する。 |

## OpenID Connect Session Management 1.0

Draft 28 — https://openid.net/specs/openid-connect-session-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-SESSION-MGMT-CHECK-IFRAME | optional | MAY | `check_session_iframe` を公開し、RP からの `postMessage` に対して OP セッションの状態を返す。 |

## FAPI 2.0 Security Profile

Final — https://openid.net/specs/fapi-security-profile-2_0-final.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| FAPI2-PROFILE-SELECTION | optional | MUST | `Fapi2SecurityProfile` を選択したクライアントだけに本プロファイルの追加制約を適用する。 |
| FAPI2-PAR-PKCE | optional | MUST | FAPI クライアントは PAR と S256 PKCE を使用する。 |
| FAPI2-CLIENT-AUTH | optional | MUST | FAPI クライアントは `private_key_jwt` または mTLS で認証する。 |
| FAPI2-SENDER-CONSTRAINT | optional | MUST | FAPI アクセストークンに DPoP または mTLS による送信者制約を付ける。 |
