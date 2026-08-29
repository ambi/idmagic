# OAuth2 Glossary

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

この Context の Aggregate root は `OAuth2Client`、`Consent`、`ApprovalRequest`、`AuthorizationRequest`、`AuthorizationCodeRecord`、`PARRecord`、`DeviceAuthorization`、`RefreshTokenRecord`、`McpResourceServer`、`AuthorizationDetailType` である。`ClaimMappingPolicy` は `OAuth2Client` の内側の値であり、独立した Aggregate ではない。
