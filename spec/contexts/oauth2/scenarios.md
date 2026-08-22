# OAuth2 Scenarios

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
  - ALT ユーザーが "AR1" を拒否済みである → エラー "OAuthAccessDeniedError" → トークンは発行されず、承認要求 "AR1" の状態は Denied のままになる
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

### REQ-OAUTH2-050: `Supervised` な Agent は人間の承認を経ずに新しいトークンを得られない
- ACTOR Agent
- GIVEN `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- GIVEN "A1" の所有者は active User "alice" である
- WHEN "agent-app" として client_credentials でトークンを要求する
  - ALT "A1" の `kind` が `autonomous` である → リクエストは受理され、アクセストークンが発行される
  - ALT "A1" の `kind` が既知のどの値でもない → 承認が必要な側へ倒し、エラー "UnauthorizedClientError"
  - ALT "agent-app" にどの Agent も束縛されていない → 区分の判定を行わず、リクエストは受理される
- THEN エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- THEN "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- WHEN "agent-app" が Token Exchange で委任トークンを要求する
  - ALT ワークロード ID 連携の attestation が "A1" の client へ写る交換である → エラー "UnauthorizedClientError"
  - ALT `subject_token` が承認を経て "A1" へ発行済みのトークンである → エラー "UnauthorizedClientError" → 一つの承認は一つのトークンに対応し、派生トークンへは継承しない
  - ALT 交換に関与するどの Agent も `autonomous` である → 交換は成立する
- THEN エラー "UnauthorizedClientError" で拒否され、"AgentApprovalRequired" が発行される
- WHEN "agent-app" が "alice" 宛のバックチャネル認可を開始し、"alice" の承認後に `auth_req_id` を交換する
- THEN アクセストークンが発行される (REQ-OAUTH2-041)
- THEN "BackchannelAuthApproved" は承認の対象となった Agent の id を含む
