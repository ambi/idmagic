# Feature: OAuth2 のシナリオ

## Rule: REQ-OAUTH2-001 ユーザーに紐づく OAuth グラントは account スコープのアクセストークンを発行できる

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-001-01 通常経路

- Given クライアントは `account:read` と `account:write` を許可スコープとして登録している
- And 有効な User が Authorization Code + PKCE または Device Authorization で `account:read` に同意している
- When クライアントがユーザーに紐づくグラントを `/token` で交換する
- Then アクセストークンの `sub` は同意した User、audience はレルムの IdMagic API、スコープは `account:read` になる
- Then account リソースサーバーは、トークンの subject 本人による参照操作だけを許可する

### Example: EX-OAUTH2-001-02 `client_credentials` または User の subject を持たない Token Exchange で account スコープを要求する

- Given クライアントは `account:read` と `account:write` を許可スコープとして登録している
- And 有効な User が Authorization Code + PKCE または Device Authorization で `account:read` に同意している
- When クライアントがユーザーに紐づくグラントを `/token` で交換する
- But `client_credentials` または User の subject を持たない Token Exchange で account スコープを要求する
- Then トークンリクエストを InvalidScopeError で拒否する

### Example: EX-OAUTH2-001-03 クライアントの許可スコープまたは User の同意に account スコープが含まれない

- Given クライアントは `account:read` と `account:write` を許可スコープとして登録している
- And 有効な User が Authorization Code + PKCE または Device Authorization で `account:read` に同意している
- When クライアントがユーザーに紐づくグラントを `/token` で交換する
- But クライアントの許可スコープまたは User の同意に account スコープが含まれない
- Then account スコープは発行されない

## Rule: REQ-OAUTH2-002 API トークン発行者は account 同意スコープで自分の同意だけを操作できる

Primary actor: `SelfApiClient`

### Example: EX-OAUTH2-002-01 通常経路

- Given クライアントは対象テナントの active User に固定された有効な API access トークンを提示している
- When クライアントが自身の active 同意の参照または撤回を要求する
- Then account:read scope は自身の active 同意の参照だけを許可する
- Then account:consents:write scope は自身の同意の撤回だけを許可する

### Example: EX-OAUTH2-002-02 account:read だけで同意 revoke を要求する

- Given クライアントは対象テナントの active User に固定された有効な API access トークンを提示している
- When クライアントが自身の active 同意の参照または撤回を要求する
- But account:read だけで同意 revoke を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-OAUTH2-002-03 トークンのテナントまたは user_id が操作対象と一致しない

- Given クライアントは対象テナントの active User に固定された有効な API access トークンを提示している
- When クライアントが自身の active 同意の参照または撤回を要求する
- But トークンのテナントまたは user_id が操作対象と一致しない
- Then 操作は AccessDeniedError で拒否される

## Rule: REQ-OAUTH2-003 管理 API クライアントは OAuth リソースのスコープで許可された操作だけを実行できる

Primary actor: `ManagementApiClient`

### Example: EX-OAUTH2-003-01 通常経路

- Given クライアントは対象テナントの有効な API access トークンを提示している
- When クライアントが OAuth2 クライアント、認可詳細タイプ、または MCP リソースサーバーの操作をリクエストする
- Then oauth-clients:read scope は OAuth2 クライアントの参照だけを許可する
- Then `authorization-detail-types:write` スコープは認可詳細タイプの変更だけを許可する
- Then `mcp-resource-servers:read` スコープは MCP リソースサーバーの参照だけを許可する

### Example: EX-OAUTH2-003-02 oauth-clients:read だけで OAuth2 クライアントの変更を要求する

- Given クライアントは対象テナントの有効な API access トークンを提示している
- When クライアントが OAuth2 クライアント、認可詳細タイプ、または MCP リソースサーバーの操作をリクエストする
- But oauth-clients:read だけで OAuth2 クライアントの変更を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-OAUTH2-003-03 別 resource の scope で操作を要求する

- Given クライアントは対象テナントの有効な API access トークンを提示している
- When クライアントが OAuth2 クライアント、認可詳細タイプ、または MCP リソースサーバーの操作をリクエストする
- But 別 resource の scope で操作を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-OAUTH2-003-04 トークンのテナントとリクエスト先のテナントが一致しない

- Given クライアントは対象テナントの有効な API access トークンを提示している
- When クライアントが OAuth2 クライアント、認可詳細タイプ、または MCP リソースサーバーの操作をリクエストする
- But トークンのテナントとリクエスト先のテナントが一致しない
- Then 操作は `AccessDeniedError` で拒否される

## Rule: REQ-OAUTH2-004 管理者は自身に可視なロールポリシーを確認できる

Primary actor: `TenantAdministrator`

### Example: EX-OAUTH2-004-01 通常経路

- Given ロール=["admin"] の管理者が認証済みである
- When 管理者がロールポリシー一覧を取得する
- Then レスポンスには参照可能なロール、権限、対応する HTTP インターフェースが含まれる

### Example: EX-OAUTH2-004-02 プリンシパルが `admin` または `system_admin` ではない

- Given ロール=["admin"] の管理者が認証済みである
- When 管理者がロールポリシー一覧を取得する
- But プリンシパルが `admin` または `system_admin` ではない
- Then ロールポリシー一覧を AccessDeniedError で拒否する

## Rule: REQ-OAUTH2-005 認可コードフローでアクセストークンと ID トークンを取得できる

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-005-01 通常経路

- Given "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- And ユーザー "alice" は "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- When クライアントが発行された認可コードを正しい PKCE verifier で交換する
- Then レスポンスに `access_token`、`id_token`、`refresh_token` が含まれ、`token_type` は `Bearer`
- Then "UserAuthenticated" が発行される
- Then "AuthorizationCodeIssued" が発行される
- Then "AuthorizationCodeRedeemed" が発行される
- Then "AccessTokenIssued" が発行される
- Then "RefreshTokenIssued" が発行される

### Example: EX-OAUTH2-005-02 認可リクエストの redirect_uri が未登録である

- Given "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- And ユーザー "alice" は "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- But 認可リクエストの redirect_uri が未登録である
- Then リダイレクトは行われず IdP がエラーページを表示する
- And エラー "InvalidRequestError"

### Example: EX-OAUTH2-005-03 単一値の認可パラメーターが重複する、`prompt` に重複または未対応のトークンがある、または `none` がほかの `prompt` トークンと併用される

- Given "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- And ユーザー "alice" は "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- But 単一値の認可パラメーターが重複する、`prompt` に重複または未対応のトークンがある、または `none` がほかの `prompt` トークンと併用される
- Then 認可コードは発行されない
- And 安全に確定した登録済みの `redirect_uri` があれば `state` と発行者の識別子を含む `invalid_request` を返す
- And それ以外はリダイレクトせず IdP がエラーページを表示する

### Example: EX-OAUTH2-005-04 `request_uri` と、併用を許可しないフロントチャネル認可パラメーターが混在する

- Given "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- And ユーザー "alice" は "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- But `request_uri` と、併用を許可しないフロントチャネル認可パラメーターが混在する
- Then 認可コードは発行されない
- And エラー "InvalidRequestError"

### Example: EX-OAUTH2-005-05 `prompt=none` で既存セッションまたは必要な同意がない

- Given "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- And ユーザー "alice" は "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- But `prompt=none` で既存セッションまたは必要な同意がない
- Then UI とログインへのリダイレクトは発生しない
- And 既存セッションがなければ `state` と発行者識別子を含む `login_required` を登録済みの `redirect_uri` へ返す
- And 同意がなければ `state` と発行者識別子を含む `consent_required` を登録済みの `redirect_uri` へ返す

### Example: EX-OAUTH2-005-06 PKCE verifier が一致しない

- Given "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- And ユーザー "alice" は "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- When クライアントが発行された認可コードを正しい PKCE verifier で交換する
- But PKCE verifier が一致しない
- Then 認可コードを誤った code_verifier で交換する
- And エラー "InvalidGrantError"
- And トークンは発行されない

### Example: EX-OAUTH2-005-07 同じ認可コードを 2 回交換する

- Given "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- And ユーザー "alice" は "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- When クライアントが発行された認可コードを正しい PKCE verifier で交換する
- But 同じ認可コードを 2 回交換する
- Then 1 回目のレスポンスには access_token が含まれる
- And 2 回目はエラー "InvalidGrantError"
- And 発行ファミリーのトークンがすべて失効する
- And "RefreshTokenReuseDetected" が発行される
- And "TokenRevoked" が発行される

### Example: EX-OAUTH2-005-08 認可コードが発行から 60 秒を超えている

- Given "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- And ユーザー "alice" は "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- When クライアントが発行された認可コードを正しい PKCE verifier で交換する
- But 認可コードが発行から 60 秒を超えている
- Then 認可コードの交換はエラー "InvalidGrantError"
- And 認可コードの状態は Expired になる

## Rule: REQ-OAUTH2-006 リフレッシュトークンをローテーションして新しいトークンを得る

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-006-01 通常経路

- Given 有効な refresh トークン "RT1" が存在する
- When リフレッシュトークン "RT1" を交換する
- Then レスポンスに新しい access_token と refresh_token が含まれる
- Then "RT1" の状態は "Rotated"
- Then "RefreshTokenRotated" が発行される
- Then "AccessTokenIssued" が発行される
- Then "RefreshTokenIssued" が発行される

### Example: EX-OAUTH2-006-02 ローテーション済みの旧 refresh トークンを再使用する

- Given 有効な refresh トークン "RT1" が存在する
- When リフレッシュトークン "RT1" を交換する
- But ローテーション済みの旧 refresh トークンを再使用する
- Then family_id "F1" の refresh トークン "RT1" をローテーション後に再使用する
- And 再使用はエラー "InvalidGrantError"
- And "RT1" の状態は "Revoked"
- And family_id "F1" のトークンがすべて失効する
- And "RefreshTokenReuseDetected" が発行される
- And "TokenRevoked" が発行される

## Rule: REQ-OAUTH2-007 不正なクライアント認証は invalid_client で一律拒否される

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-007-01 通常経路

- When 既知のクライアントを誤った client_secret で認可コードを交換する
- Then エラー "InvalidClientError"

### Example: EX-OAUTH2-007-02 未知の client_id で交換する

- When 既知のクライアントを誤った client_secret で認可コードを交換する
- But 未知の client_id で交換する
- Then 未知の client_id で認可コードを交換する
- And エラー "InvalidClientError"

## Rule: REQ-OAUTH2-008 既存同意の有無に応じて同意画面を出し分ける

Primary actor: `ResourceOwner`

### Example: EX-OAUTH2-008-01 通常経路

- Given ユーザー "alice" が "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- Then 認可リクエストの状態は Consented
- Then 同意 UI は表示されない

### Example: EX-OAUTH2-008-02 prompt=同意で再同意を要求する

- Given ユーザー "alice" が "web-app" に scope "openid プロファイル" を同意済みである
- When "web-app" として scope "openid プロファイル" で認可リクエストを送る
- But prompt=同意で再同意を要求する
- Then "web-app" として prompt "同意" で認可リクエストを送る
- And 認可リクエストの状態は ConsentPending

## Rule: REQ-OAUTH2-009 PAR で送信した認可リクエストを request_uri 経由で実行する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-009-01 通常経路

- Given クライアント "web-app" が存在する
- When "web-app" として認可リクエストを事前送信する
- Then PAR レスポンスに request_uri が含まれ expires_in は 600 以下
- When クライアントが request_uri "<返された値>" で認可リクエストを送る
- Then その PAR レコードの状態は "Used"
- Then "PARStored" が発行される
- Then "AuthorizationCodeIssued" が発行される

### Example: EX-OAUTH2-009-02 PAR 必須の FAPI クライアントが PAR なしで直接送信する

- Given クライアント "web-app" が存在する
- When "web-app" として認可リクエストを事前送信する
- But PAR 必須の FAPI クライアントが PAR なしで直接送信する
- Then PAR 必須の FAPI クライアント "fapi-app" として scope "openid" で直接認可リクエストを送る
- And エラー "InvalidRequestError"

## Rule: REQ-OAUTH2-010 DPoP 証明付き要求はセンダー制約付きトークンを発行する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-010-01 通常経路

- When 有効な DPoP 証明を付けて認可コードを交換する
- Then 発行された access トークンは DPoP 鍵サムプリントに cnf でバインドされる
- Then 発行された refresh トークンのセンダー制約は Dpop

### Example: EX-OAUTH2-010-02 iat が 60 秒以上古い DPoP 証明である

- When 有効な DPoP 証明を付けて認可コードを交換する
- But iat が 60 秒以上古い DPoP 証明である
- Then iat "2026-01-01T00:00:00Z" の DPoP 証明を時刻 "2026-01-01T00:01:30Z" で付けて認可コードを交換する
- And エラー "InvalidDpopProofError"

### Example: EX-OAUTH2-010-03 同一 DPoP jti を再使用する

- When 有効な DPoP 証明を付けて認可コードを交換する
- But 同一 DPoP jti を再使用する
- Then jti "ABC" の DPoP 証明を付けて認可コードを交換する
- And 同じ jti "ABC" の DPoP 証明を付けて認可コードを交換する
- And 1 回目のレスポンスには access_token が含まれる
- And 2 回目はエラー "InvalidDpopProofError"

## Rule: REQ-OAUTH2-011 失効済みトークンのイントロスペクションは `active=false` だけを返す

Primary actor: `ResourceServer`

### Example: EX-OAUTH2-011-01 通常経路

- Given 失効済み access トークン "AT1" が存在する
- When トークン "AT1" を検査する
- Then レスポンスは active=false のみで他のフィールドを含まない

## Rule: REQ-OAUTH2-012 キルスイッチ作動後の Agent トークンはイントロスペクションで active=false になる

Primary actor: `ResourceServer`

### Example: EX-OAUTH2-012-01 通常経路

- Given Agent "A1" に issued_at が古い access トークン "AT1" が発行済みである
- And "A1" は kill-switch により revocation epoch が "AT1" の issued_at より後へ前進している
- When トークン "AT1" を検査する
- Then レスポンスは active=false のみで他のフィールドを含まない

### Example: EX-OAUTH2-012-02 "AT1" が revocation epoch より後に発行された (kill 後に再発行された) トークンである

- Given Agent "A1" に issued_at が古い access トークン "AT1" が発行済みである
- And "A1" は kill-switch により revocation epoch が "AT1" の issued_at より後へ前進している
- When トークン "AT1" を検査する
- But "AT1" が revocation epoch より後に発行された (kill 後に再発行された) トークンである
- Then レスポンスは通常どおり active=true と claim を返す

## Rule: REQ-OAUTH2-013 UserInfo は openid スコープのトークンに sub を返す

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-013-01 通常経路

- Given scope "openid プロファイル" の access トークン "AT1" が存在する
- When トークン "AT1" でユーザー情報を取得する
- Then レスポンスに `sub`、`name`、`preferred_username` が含まれる

### Example: EX-OAUTH2-013-02 openid スコープを持たないトークンで取得する

- Given scope "openid プロファイル" の access トークン "AT1" が存在する
- When トークン "AT1" でユーザー情報を取得する
- But openid スコープを持たないトークンで取得する
- Then scope "プロファイル" のみの access トークン "AT1" でユーザー情報を取得する
- And エラー "OAuthInsufficientScopeError"

## Rule: REQ-OAUTH2-014 Discovery Metadata は宣言された全エンドポイントを広告する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-014-01 通常経路

- When Discovery Metadata を取得する
- Then レスポンスに `issuer`、`authorization_endpoint`、`token_endpoint`、`userinfo_endpoint`、`jwks_uri`、`introspection_endpoint`、`revocation_endpoint`、`pushed_authorization_request_endpoint`、`device_authorization_endpoint`、`backchannel_authentication_endpoint`、`registration_endpoint` が含まれる

## Rule: REQ-OAUTH2-015 認可コードの並行交換はちょうど一方だけ成功する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-015-01 通常経路

- Given 発行済み認可コード "AC1"（family_id "F1"）が存在する
- When 認可コード "AC1" を verifier "v" で並行に 2 回交換する
- Then ちょうど一方が成功し、もう一方はエラー "InvalidGrantError"
- Then family_id "F1" のトークンがすべて失効する
- Then "AuthorizationCodeRedeemed" が発行される
- Then "RefreshTokenReuseDetected" が発行される
- Then "TokenRevoked" が発行される

## Rule: REQ-OAUTH2-016 動的クライアント登録は `client_id` を採番して返す

Primary actor: `Client`

### Example: EX-OAUTH2-016-01 通常経路

- When confidential クライアント "web-app" を redirect_uri "https://app.example.com/callback" で登録する
- Then レスポンスに client_id と client_secret が含まれる
- Then "ClientRegistered" が発行される

### Example: EX-OAUTH2-016-02 redirect_uri を持たない登録要求である

- When confidential クライアント "web-app" を redirect_uri "https://app.example.com/callback" で登録する
- But redirect_uri を持たない登録要求である
- Then confidential クライアント "web-app" を redirect_uri "" で登録する
- And エラー "InvalidRequestError"

## Rule: REQ-OAUTH2-017 クライアントメタデータの取得では公開 IP へ直接接続する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-017-01 通常経路

- Given `client_id` は公開 IP に解決される、クライアント所有の HTTPS メタデータ URL である
- And Authorization Server の環境に HTTPS プロキシが設定されている
- When Authorization Server がクライアントのメタデータ URL を取得する
- Then 環境のプロキシを使用せず、DNS 検査済みの公開 IP へ直接接続する
- Then metadata document の取得と検証に成功する

### Example: EX-OAUTH2-017-02 メタデータのホストがプライベート、ループバック、リンクローカル、または CGNAT 100.64.0.0/10 の IP に解決される

- Given `client_id` は公開 IP に解決される、クライアント所有の HTTPS メタデータ URL である
- And Authorization Server の環境に HTTPS プロキシが設定されている
- When Authorization Server がクライアントのメタデータ URL を取得する
- But メタデータのホストがプライベート、ループバック、リンクローカル、または CGNAT 100.64.0.0/10 の IP に解決される
- Then Authorization Server は対象 IP へ接続しない
- And クライアントメタデータの解決をフェイルクローズで拒否する

## Rule: REQ-OAUTH2-018 絶対有効期限を過ぎたリフレッシュトークンはローテーションできない

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-018-01 通常経路

- Given absolute_expires_at "2026-01-01T00:00:00Z" の refresh トークン "RT1" が存在する
- And 現在時刻は "2026-01-02T00:00:00Z" である
- When クライアントがリフレッシュトークン "RT1" を交換する
- Then エラー "InvalidGrantError"

## Rule: REQ-OAUTH2-019 クライアントは自分のトークンを失効できる

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-019-01 通常経路

- Given 有効な refresh トークン "RT1" が存在する
- When トークン "RT1" を失効させる
- Then "RT1" の状態は "Revoked"
- Then "TokenRevoked" が発行される

### Example: EX-OAUTH2-019-02 所有者でないクライアントが失効を要求する

- Given 有効な refresh トークン "RT1" が存在する
- When トークン "RT1" を失効させる
- But 所有者でないクライアントが失効を要求する
- Then クライアント "client-A" が所有する refresh トークン "RT1" に対し "client-B" として失効を要求する
- And 盗難検知防止のため 200 OK のみ返り "RT1" の状態は "Active" のまま

## Rule: REQ-OAUTH2-020 失効したアクセストークンによる UserInfo の取得は `invalid_token` で拒否される

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-020-01 通常経路

- Given 有効な access トークン "AT1" が存在する
- When トークン "AT1" を失効させる
- Then トークン "AT1" は失効状態になる
- When クライアントがトークン "AT1" でユーザー情報を取得する
- Then 401 のエラー "InvalidTokenError" と `WWW-Authenticate: Bearer` challenge が返る
- And レスポンスに `sub` とユーザークレームは含まれない

### Example: EX-OAUTH2-020-02 POST binding

- Given 失効済みの access トークン "AT1" が存在する
- When クライアントが POST binding でトークン "AT1" を使いユーザー情報を取得する
- Then 401 のエラー "InvalidTokenError" と `WWW-Authenticate: Bearer` challenge が返る
- And レスポンスに `sub` とユーザークレームは含まれない

## Rule: REQ-OAUTH2-021 リフレッシュトークンは `offline_access` スコープを付与したときだけ発行する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-021-01 通常経路

- Given confidential クライアント "web-app" が grant_types に "authorization_code"・"refresh_token" を含めて登録済みである
- When "web-app" として scope "openid offline_access" で認可リクエストを送る
- When クライアントが発行された認可コードを verifier "v" で交換する
- Then レスポンスに refresh_token が含まれる
- Then "RefreshTokenIssued" が発行される

### Example: EX-OAUTH2-021-02 offline_access を要求しない

- Given confidential クライアント "web-app" が grant_types に "authorization_code"・"refresh_token" を含めて登録済みである
- When "web-app" として scope "openid offline_access" で認可リクエストを送る
- But offline_access を要求しない
- Then "web-app" として scope "openid プロファイル" で認可リクエストを送る
- And 発行された認可コードを verifier "v" で交換する
- And レスポンスに refresh_token は含まれない

## Rule: REQ-OAUTH2-022 認可リクエストの nonce は ID トークンに伝播する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-022-01 通常経路

- When "web-app" として scope "openid"、nonce "n-12345" で認可リクエストを送る
- When クライアントが発行された認可コードを verifier "v" で交換する
- Then レスポンスの id_token の nonce クレームは "n-12345"

## Rule: REQ-OAUTH2-023 RP-Initiated Logout は登録済み post_logout_redirect_uri にだけ戻す

Primary actor: `ResourceOwner`

### Example: EX-OAUTH2-023-01 通常経路

- Given confidential クライアント "web-app" が redirect_uri "https://app.example.com/cb" で登録済みである
- When "web-app" として post_logout_redirect_uri "https://app.example.com/cb" でログアウトする
- Then `state` が `post_logout_redirect_uri` に伝播する

### Example: EX-OAUTH2-023-02 未登録の post_logout_redirect_uri を指定する

- Given confidential クライアント "web-app" が redirect_uri "https://app.example.com/cb" で登録済みである
- When "web-app" として post_logout_redirect_uri "https://app.example.com/cb" でログアウトする
- But 未登録の post_logout_redirect_uri を指定する
- Then "web-app" として post_logout_redirect_uri "https://evil.example.com/cb" でログアウトする
- And エラー "InvalidRequestError"

## Rule: REQ-OAUTH2-024 RP-Initiated Logout は id_token_hint からセッションとクライアントを特定する

Primary actor: `ResourceOwner`

### Example: EX-OAUTH2-024-01 通常経路

- Given ユーザー "alice" が "web-app" として認可コードを交換し、`sid` 付きの ID Token を持つ
- When "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
- Then id_token_hint の sid が示す LoginSession が失効する
- Then 同じ sid を持つ全クライアントの RefreshTokenRecord が Revoked へ遷移する

### Example: EX-OAUTH2-024-02 `id_token_hint` の `aud` が指定された `client_id` と一致しない

- Given ユーザー "alice" が "web-app" として認可コードを交換し、`sid` 付きの ID Token を持つ
- When "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
- But `id_token_hint` の `aud` が指定された `client_id` と一致しない
- Then `client_id` "other-app" と "web-app" 発行の ID Token を `id_token_hint` に付けて `/end_session` を呼ぶ
- And エラー "InvalidRequestError"

### Example: EX-OAUTH2-024-03 `id_token_hint` の署名を IdMagic の署名鍵で検証できない

- Given ユーザー "alice" が "web-app" として認可コードを交換し、`sid` 付きの ID Token を持つ
- When "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
- But `id_token_hint` の署名を IdMagic の署名鍵で検証できない
- Then 他の発行者が署名した JWT を `id_token_hint` に付けて `/end_session` を呼ぶ
- And エラー "InvalidRequestError"

### Example: EX-OAUTH2-024-04 id_token_hint が期限切れ (exp 経過) である

- Given ユーザー "alice" が "web-app" として認可コードを交換し、`sid` 付きの ID Token を持つ
- When "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
- But id_token_hint が期限切れ (exp 経過) である
- Then 期限切れの発行済み ID Token を id_token_hint として /end_session を呼ぶ
- And exp 切れのみを理由にした拒否はされず sid によるセッション解決が成功する

### Example: EX-OAUTH2-024-05 `id_token_hint` に `sid`、`sub`、`aud` のいずれかが無い

- Given ユーザー "alice" が "web-app" として認可コードを交換し、`sid` 付きの ID Token を持つ
- When "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
- But `id_token_hint` に `sid`、`sub`、`aud` のいずれかが無い
- Then エラー "InvalidRequestError"
- And ブラウザー Cookie が示す LoginSession は失効しない
- And その LoginSession と同じ sid の RefreshTokenRecord も失効しない

### Example: EX-OAUTH2-024-06 `id_token_hint` の `sub` が `sid` の LoginSession の主体と一致しない

- Given ユーザー "alice" が "web-app" として認可コードを交換し、`sid` 付きの ID Token を持つ
- When "alice" が発行済み ID Token を `id_token_hint` として `/end_session` を呼ぶ
- But `id_token_hint` の `sub` が `sid` の LoginSession の主体と一致しない
- Then エラー "InvalidRequestError"
- And `sid` が示す LoginSession は失効しない
- And 同じ sid の RefreshTokenRecord も失効しない

## Rule: REQ-OAUTH2-025 セッション失効時は `backchannel_logout_uri` を登録済みの RP へログアウトトークンを配信する

Primary actor: `ResourceOwner`

### Example: EX-OAUTH2-025-01 通常経路

- Given "web-app" が backchannel_logout_uri "https://app.example.com/backchannel_logout" を登録済みである
- And ユーザー "alice" が "web-app" とのブラウザセッションを持つ
- When "alice" が /end_session でログアウトする
- Then 対象 sid と "web-app" の LogoutNotification が作成される
- Then 署名済みのログアウトトークンが `backchannel_logout_uri` へ配信され、`Delivered` になる

### Example: EX-OAUTH2-025-02 配送が一時的に失敗する (5xx / timeout)

- Given "web-app" が backchannel_logout_uri "https://app.example.com/backchannel_logout" を登録済みである
- And ユーザー "alice" が "web-app" とのブラウザセッションを持つ
- When "alice" が /end_session でログアウトする
- Then 対象 sid と "web-app" の LogoutNotification が作成される
- Then 配送が一時的に失敗する (5xx / timeout)
- Then LogoutNotification は Pending のまま attempts が増え再試行され、ローカルのセッション/refresh トークン失効は取り消されない

### Example: EX-OAUTH2-025-03 max_attempts まで再試行しても配送が成功しない

- Given "web-app" が backchannel_logout_uri "https://app.example.com/backchannel_logout" を登録済みである
- And ユーザー "alice" が "web-app" とのブラウザセッションを持つ
- When "alice" が /end_session でログアウトする
- Then 対象 sid と "web-app" の LogoutNotification が作成される
- Then max_attempts まで再試行しても配送が成功しない
- Then LogoutNotification は Failed (dead-letter) に確定し、ローカルのセッション/refresh トークン失効は取り消されない

## Rule: REQ-OAUTH2-026 client_credentials グラントで M2M トークンが発行される

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-026-01 通常経路

- Given confidential クライアント "backend" が grant_types に "client_credentials" を含めて登録済みである
- When "backend" として client_credentials で scope "api:read" のトークンを取得する
- Then レスポンスに access_token が含まれ refresh_token は含まれない
- Then 発行された access_token の sub は client_id と一致する
- Then "AccessTokenIssued" が発行される
- When public クライアント "spa-app" を grant_types に "client_credentials" を含めて登録する
- Then client_credentials は confidential 限定であるため InvalidRequestError で拒否される

## Rule: REQ-OAUTH2-027 デバイス認可フローでアクセストークンを取得できる

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-027-01 通常経路

- Given confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- When "tv-app" として scope "openid プロファイル" でデバイス認可を開始する
- Then レスポンスに device_code・user_code・verification_uri・interval が含まれる
- When ユーザー "alice" が verification_uri で user_code を入力し承認する
- Then device authorization は承認済みになる
- When クライアントが device_code "DC1" を交換する
- Then レスポンスに access_token と id_token が含まれる
- Then "DeviceAuthorizationRequested" が発行される
- Then "DeviceAuthorizationApproved" が発行される
- Then "AccessTokenIssued" が発行される

### Example: EX-OAUTH2-027-02 ユーザー承認前にポーリングする

- Given confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- When "tv-app" として scope "openid プロファイル" でデバイス認可を開始する
- Then レスポンスに device_code・user_code・verification_uri・interval が含まれる
- When ユーザー "alice" が verification_uri で user_code を入力し承認する
- Then device authorization は承認済みになる
- When クライアントが device_code "DC1" を交換する
- But ユーザー承認前にポーリングする
- Then Issued 状態の device_code "DC1" を交換する
- And エラー "AuthorizationPendingError"

### Example: EX-OAUTH2-027-03 ポーリング間隔より短い再試行をする

- Given confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- When "tv-app" として scope "openid プロファイル" でデバイス認可を開始する
- Then レスポンスに device_code・user_code・verification_uri・interval が含まれる
- When ユーザー "alice" が verification_uri で user_code を入力し承認する
- Then device authorization は承認済みになる
- When クライアントが device_code "DC1" を交換する
- But ポーリング間隔より短い再試行をする
- Then interval 5 秒の device_code "DC1" を Issued 状態で用意する
- And device_code "DC1" を交換し "2s" 経過後に再度交換する
- And 2 回目はエラー "SlowDownError"

### Example: EX-OAUTH2-027-04 device_code が expires_in を超えている

- Given confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- When "tv-app" として scope "openid プロファイル" でデバイス認可を開始する
- Then レスポンスに device_code・user_code・verification_uri・interval が含まれる
- When ユーザー "alice" が verification_uri で user_code を入力し承認する
- Then device authorization は承認済みになる
- When クライアントが device_code "DC1" を交換する
- But device_code が expires_in を超えている
- Then issued_at "2026-01-01T00:00:00Z"・expires_at "2026-01-01T00:10:00Z" の device_code "DC1" を時刻 "2026-01-01T00:11:00Z" で交換する
- And エラー "ExpiredTokenError"

## Rule: REQ-OAUTH2-028 改ざんされた client_assertion は invalid_client で拒否される

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-028-01 通常経路

- Given confidential クライアント "fapi-app" が token_endpoint_auth_method "private_key_jwt"・jwks 登録で存在する
- When 認可コード "AC1" を verifier "v"・改ざんされた client_assertion で交換する
- Then エラー "InvalidClientError"

## Rule: REQ-OAUTH2-029 mTLS バインド AT は同じ証明書のリクエストでのみ受理される

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-029-01 通常経路

- Given confidential クライアント "mtls-app" が token_endpoint_auth_method "tls_client_auth" で存在する
- When クライアントが mTLS 証明書を提示して access_token を要求する
- Then access_token は提示された証明書にバインドされる
- When クライアントが証明書にバインドされた access_token で userinfo を取得する
- Then 同じ証明書を提示した要求は 200 を返す

### Example: EX-OAUTH2-029-02 別の証明書を提示する

- Given confidential クライアント "mtls-app" が token_endpoint_auth_method "tls_client_auth" で存在する
- When クライアントが mTLS 証明書を提示して access_token を要求する
- Then access_token は提示された証明書にバインドされる
- When クライアントが証明書にバインドされた access_token で userinfo を取得する
- But 別の証明書を提示する
- Then invalid_token で拒否される

## Rule: REQ-OAUTH2-030 RFC 8414 メタデータ文書は OIDC Discovery と同等の内容を返す

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-030-01 通常経路

- When Authorization Server メタデータを取得する
- Then レスポンスに `issuer`、`authorization_endpoint`、`token_endpoint`、`jwks_uri`、`grant_types_supported` が含まれる

## Rule: REQ-OAUTH2-031 管理者は所属テナントの同意を参照・撤回できるが付与は代行できない

Primary actor: `TenantAdministrator`

### Example: EX-OAUTH2-031-01 通常経路

- Given tenant_id "acme" のロール=["admin"] のユーザー "operator" が認証済みである
- And tenant_id "acme" のユーザー "alice" とクライアント "portal" の Consent が Granted で存在する
- When 管理者 "operator" が Consent 一覧と単一 Consent を取得する
- Then 所属テナントの Consent だけが返る
- When 管理者 "operator" がユーザー "alice" とクライアント "portal" の Consent を撤回する
- Then `Consent.state` は `Revoked` となり、`revoked_at` が記録される
- Then "ConsentRevoked" が actorUserId "operator" で発行される
- Then 管理者が Consent を作成または scope 拡張する interface は存在しない

## Rule: REQ-OAUTH2-032 ユーザーは接続済みアプリの同意を自分で撤回できる

Primary actor: `ResourceOwner`

### Example: EX-OAUTH2-032-01 通常経路

- Given ユーザー "alice" がクライアント "web-app" に scope "openid プロファイル" を同意済みである
- And ユーザー "alice" が認証済みで接続済みアプリ画面を開いている
- When ユーザー "alice" が接続済みアプリ一覧を取得する
- Then 一覧に "web-app" が表示される
- When ユーザー "alice" が "web-app" の同意を撤回する
- Then `Consent.state` は `Revoked` となり、一覧から消える

## Rule: REQ-OAUTH2-033 realm 接頭辞付きの Discovery Metadata は同じ接頭辞を持つ発行者を返す

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-033-01 通常経路

- Given tenant_id "acme" が Active で存在する
- When /realms/acme/.well-known/openid-configuration を取得する
- Then レスポンスの `issuer` は基底 URL + `/realms/acme` となる
- Then レスポンスの `authorization_endpoint` は基底 URL + `/realms/acme/authorize` となる

## Rule: REQ-OAUTH2-034 トークンエンドポイントはテナント境界を越えた資格情報を受理しない

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-034-01 通常経路

- When tenant_id "acme" で発行した認可コード "AC1" を "/realms/default/token" で交換する
- Then エラー "InvalidGrantError"
- When 永続化層へ `tenant_id=acme` のリフレッシュトークンと `tenant_id=default` の `client_id` または `sub` を書き込む
- Then 永続化層が参照整合性エラーで拒否する

### Example: EX-OAUTH2-034-02 他テナントの client_id を使う

- When tenant_id "acme" で発行した認可コード "AC1" を "/realms/default/token" で交換する
- But 他テナントの client_id を使う
- Then tenant_id "acme" に登録した client_id "web-app" で "/realms/default/token" に交換を要求する
- And エラー "InvalidClientError"

### Example: EX-OAUTH2-034-03 他テナントのリフレッシュトークンを再発行する

- When tenant_id "acme" で発行した認可コード "AC1" を "/realms/default/token" で交換する
- But 他テナントのリフレッシュトークンを再発行する
- Then tenant_id "acme" で発行した refresh トークン "RT1" を "/realms/default/token" で再発行する
- And エラー "InvalidGrantError"

### Example: EX-OAUTH2-034-04 他テナントの device_code を交換する

- When tenant_id "acme" で発行した認可コード "AC1" を "/realms/default/token" で交換する
- But 他テナントの device_code を交換する
- Then tenant_id "acme" で発行し承認した device_code "DC1" を "/realms/default/token" で交換する
- And エラー "InvalidGrantError"

## Rule: REQ-OAUTH2-035 管理者は所属テナントのクライアントを作成・更新・削除できる

Primary actor: `TenantAdministrator`

### Example: EX-OAUTH2-035-01 通常経路

- Given tenant_id "acme" の admin "operator" が認証済みである
- When "operator" がクライアント "portal" を作成する
- Then client_secret が一度だけ返る
- When "operator" がクライアント "portal" を取得する
- Then 所属テナントのクライアントだけが返る
- When "operator" がクライアント "portal" の redirect_uris を更新する
- Then redirect_uris が保存される
- When "operator" がクライアント "portal" を削除する
- Then "AdminOAuth2ClientCreated"、"AdminOAuth2ClientUpdated"、"AdminOAuth2ClientDeleted" が発行される

### Example: EX-OAUTH2-035-02 別テナントの管理者が同じ client_id を指定する

- Given tenant_id "acme" の admin "operator" が認証済みである
- When "operator" がクライアント "portal" を作成する
- Then client_secret が一度だけ返る
- When "operator" がクライアント "portal" を取得する
- But 別テナントの管理者が同じ client_id を指定する
- Then InvalidRequestError で拒否される

## Rule: REQ-OAUTH2-036 管理者は Application から期限付きクライアントシークレットを追加発行し、個別に失効できる

Primary actor: `TenantAdministrator`

### Example: EX-OAUTH2-036-01 通常経路

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 有効期限のない従来のシークレット "S1" が `Active` である
- When 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
- Then レスポンスで新しいシークレットを一度だけ受け取り、メタデータは 90 日後の `expires_at` と `Active` ステータスを持つ
- Then 追加発行によって既存シークレットの期限とステータスは変わらない
- Then 新旧両方のシークレットでトークンエンドポイントの認証に成功する
- When 管理者が以前の資格情報だけを個別に失効する
- Then 以前のシークレットは InvalidClientError で拒否され、新しいシークレットでは引き続き認証に成功する
- Then ClientSecretIssued と ClientSecretRevoked は、`actor`、クライアント、`credential`、`expiry` の非機密メタデータだけを含んで発行される

### Example: EX-OAUTH2-036-02 `expires_in_days` が 1..730 の範囲外である

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 有効期限のない従来のシークレット "S1" が `Active` である
- When 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
- But `expires_in_days` が 1..730 の範囲外である
- Then エラー "InvalidRequestError"

### Example: EX-OAUTH2-036-03 `Active` の資格情報がすでに 2 件存在する

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 有効期限のない従来のシークレット "S1" が `Active` である
- When 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
- But `Active` の資格情報がすでに 2 件存在する
- Then 追加発行をエラー "ClientSecretLimitExceededError" で拒否し、既存の資格情報は変更しない

### Example: EX-OAUTH2-036-04 クライアントが `private_key_jwt`、mTLS、または公開クライアントである

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 有効期限のない従来のシークレット "S1" が `Active` である
- When 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
- But クライアントが `private_key_jwt`、mTLS、または公開クライアントである
- Then エラー "InvalidRequestError"

### Example: EX-OAUTH2-036-05 別クライアントの `credential_id` または存在しない `credential_id` を失効する

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 有効期限のない従来のシークレット "S1" が `Active` である
- When 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
- Then レスポンスで新しいシークレットを一度だけ受け取り、メタデータは 90 日後の `expires_at` と `Active` ステータスを持つ
- Then 追加発行によって既存シークレットの期限とステータスは変わらない
- Then 新旧両方のシークレットでトークンエンドポイントの認証に成功する
- When 管理者が以前の資格情報だけを個別に失効する
- But 別クライアントの `credential_id` または存在しない `credential_id` を失効する
- Then エラー "InvalidRequestError"

### Example: EX-OAUTH2-036-06 すでに `Revoked` の資格情報を再び失効する

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 有効期限のない従来のシークレット "S1" が `Active` である
- When 管理者が `expires_in_days=90` で新しいシークレットを追加発行する
- Then レスポンスで新しいシークレットを一度だけ受け取り、メタデータは 90 日後の `expires_at` と `Active` ステータスを持つ
- Then 追加発行によって既存シークレットの期限とステータスは変わらない
- Then 新旧両方のシークレットでトークンエンドポイントの認証に成功する
- When 管理者が以前の資格情報だけを個別に失効する
- But すでに `Revoked` の資格情報を再び失効する
- Then 冪等に成功し、ClientSecretRevoked は重複発行されない

## Rule: REQ-OAUTH2-037 管理者は互換インターフェースからクライアントシークレットを無停止でローテーションできる

Primary actor: `TenantAdministrator`

### Example: EX-OAUTH2-037-01 通常経路

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 以前のシークレット "S1" が有効である
- When 管理者が `grace_days=7` でシークレットをローテーションする
- Then レスポンスで新しいシークレットを一度だけ受け取る
- Then 新旧両方のシークレットは `grace_until` より前にトークンエンドポイントの認証に成功する
- Then `grace_until` より後は以前のシークレットが InvalidClientError で拒否される
- Then ClientSecretRotated は `actor`、クライアント、`grace_until` だけを含んで発行される

### Example: EX-OAUTH2-037-02 `grace_days` が 0 以外で 1..30 の範囲外である

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 以前のシークレット "S1" が有効である
- When 管理者が `grace_days=7` でシークレットをローテーションする
- But `grace_days` が 0 以外で 1..30 の範囲外である
- Then エラー "InvalidRequestError"

### Example: EX-OAUTH2-037-03 クライアントが `private_key_jwt`、mTLS、または公開クライアントである

- Given `tenant_id` "acme" の Application "billing" は、`client_secret_basic` を使う confidential OIDC クライアントをプロトコル設定として持つ
- And 以前のシークレット "S1" が有効である
- When 管理者が `grace_days=7` でシークレットをローテーションする
- But クライアントが `private_key_jwt`、mTLS、または公開クライアントである
- Then エラー "InvalidRequestError"

## Rule: REQ-OAUTH2-038 同意管理 API は別テナントの同意を公開しない

Primary actor: `TenantAdministrator`

### Example: EX-OAUTH2-038-01 通常経路

- Given tenant_id "acme" のユーザーとクライアントの Consent が存在する
- When `tenant_id=default` の管理者が同じ `user_id` と `client_id` の Consent を取得する
- Then エラー "InvalidRequestError"

## Rule: REQ-OAUTH2-039 KeyProvider の障害時は新しいトークンの発行を拒否する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-039-01 通常経路

- Given tenant_id "acme" の KeyProvider が到達不能である
- When tenant_id "acme" のクライアントがトークン発行を要求する
- Then 新規署名は行われずエラー "ServerError" で拒否される

## Rule: REQ-OAUTH2-040 プロトコルエンドポイントは閾値を超えたリクエストをレート制限で拒否する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-040-01 通常経路

- Given クライアントがある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している
- When 同一 window 内で追加リクエストを送る
- Then エラー "RateLimitedError" (HTTP 429、Retry-After ヘッダ付き)

### Example: EX-OAUTH2-040-02 対象 endpoint が /tokenである

- Given クライアントがある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している
- When 同一 window 内で追加リクエストを送る
- But 対象 endpoint が /tokenである
- Then client_id と IP の組で閾値超過している状態でトークンを要求する
- And エラー "RateLimitedError"

### Example: EX-OAUTH2-040-03 対象 endpoint が /authorize または /par である

- Given クライアントがある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している
- When 同一 window 内で追加リクエストを送る
- But 対象 endpoint が /authorize または /par である
- Then IP と client_id の組で閾値超過している状態で認可リクエストを送る
- And エラー "RateLimitedError"

### Example: EX-OAUTH2-040-04 対象 endpoint が /device_authorization である

- Given クライアントがある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している
- When 同一 window 内で追加リクエストを送る
- But 対象 endpoint が /device_authorization である
- Then client_id と IP の組で閾値超過している状態でデバイス認可を開始する
- And エラー "RateLimitedError"

### Example: EX-OAUTH2-040-05 対象 endpoint が /bc-authorize である

- Given クライアントがある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している
- When 同一 window 内で追加リクエストを送る
- But 対象 endpoint が /bc-authorize である
- Then client_id と IP の組で閾値超過している状態で backchannel 認可を開始する
- And エラー "RateLimitedError"

### Example: EX-OAUTH2-040-06 共有カウンタストアに到達できない

- Given クライアントがある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している
- When 同一 window 内で追加リクエストを送る
- But 共有カウンタストアに到達できない
- Then リクエストは fail-closed で "RateLimitedError" として拒否される

## Rule: REQ-OAUTH2-041 バックチャネル認可要求は人間の承認が成立してからトークンを発行する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-041-01 通常経路

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- Then レスポンスに auth_req_id・expires_in・interval が含まれる
- Then 承認リクエストの状態は Pending になる
- Then "BackchannelAuthRequested" が発行される
- When `agent-app` が `auth_req_id=AR1` を交換する
- When ユーザー "alice" が承認リクエスト "AR1" を承認する
- Then 承認リクエスト "AR1" の状態は Approved になる
- Then "BackchannelAuthApproved" が発行される
- When `agent-app` が `auth_req_id=AR1` を交換する
- Then レスポンスに access_token と id_token が含まれ sub は "alice"、scope は要求した "openid" になる
- Then 承認リクエスト "AR1" の状態は Consumed になる
- Then "AccessTokenIssued" が発行される

### Example: EX-OAUTH2-041-02 scope が未指定または openid を含まない

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But scope が未指定または openid を含まない
- Then エラー "InvalidScopeError"

### Example: EX-OAUTH2-041-03 login_hint と id_token_hint が両方指定される、または両方未指定である

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But login_hint と id_token_hint が両方指定される、または両方未指定である
- Then エラー "InvalidRequestError"

### Example: EX-OAUTH2-041-04 requested_expiry が非正または 600 秒を超える

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But requested_expiry が非正または 600 秒を超える
- Then エラー "InvalidRequestError"

### Example: EX-OAUTH2-041-05 binding_message が 64 文字を超える、または制御文字を含む

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But binding_message が 64 文字を超える、または制御文字を含む
- Then エラー "InvalidBindingMessageError"

### Example: EX-OAUTH2-041-06 login_hint が User を解決できない

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But login_hint が User を解決できない
- Then 未知の login_hint "nobody" で backchannel 認可を開始する
- And エラー "UnknownUserIdError"

### Example: EX-OAUTH2-041-07 login_hint の User が別テナントまたは非 active である

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But login_hint の User が別テナントまたは非 active である
- Then エラー "UnknownUserIdError"

### Example: EX-OAUTH2-041-08 クライアントの許可 scope に含まれない scope を要求する

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- But クライアントの許可 scope に含まれない scope を要求する
- Then エラー "InvalidScopeError"

### Example: EX-OAUTH2-041-09 ユーザー判断前にポーリングする

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- Then レスポンスに auth_req_id・expires_in・interval が含まれる
- Then 承認リクエストの状態は Pending になる
- Then "BackchannelAuthRequested" が発行される
- When `agent-app` が `auth_req_id=AR1` を交換する
- But ユーザー判断前にポーリングする
- Then Pending 状態の auth_req_id "AR1" を交換する
- And エラー "AuthorizationPendingError"

### Example: EX-OAUTH2-041-10 interval より短い間隔で再試行する

- Given `confidential` クライアント `agent-app` が `grant_types` に `urn:openid:params:grant-type:ciba` を含めて登録済みである
- And active User "alice" が存在する
- When `agent-app` として `login_hint=alice`、`scope=openid`、`binding_message=W-123` でバックチャネル認可を開始する
- Then レスポンスに auth_req_id・expires_in・interval が含まれる
- Then 承認リクエストの状態は Pending になる
- Then "BackchannelAuthRequested" が発行される
- When `agent-app` が `auth_req_id=AR1` を交換する
- But interval より短い間隔で再試行する
- Then interval 5 秒の auth_req_id "AR1" を交換し "2s" 経過後に再度交換する
- And 2 回目はエラー "SlowDownError"

## Rule: REQ-OAUTH2-042 承認が成立していない承認リクエストはトークンを発行しない

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-042-01 通常経路

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- Then 承認リクエストが Approved のときだけレスポンスに access_token が含まれる

### Example: EX-OAUTH2-042-02 ユーザーが "AR1" を拒否済みである

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But ユーザーが "AR1" を拒否済みである
- Then エラー "OAuthAccessDeniedError"
- And トークンは発行されず、承認リクエスト "AR1" の状態は Denied のままになる

### Example: EX-OAUTH2-042-03 "AR1" が expires_at を過ぎている

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But "AR1" が expires_at を過ぎている
- Then requested_at "2026-01-01T00:00:00Z"・expires_at "2026-01-01T00:05:00Z" の "AR1" を時刻 "2026-01-01T00:06:00Z" で交換する
- And エラー "ExpiredTokenError"
- And 承認リクエスト "AR1" の状態は Expired になる

### Example: EX-OAUTH2-042-04 承認済みの "AR1" を 2 回交換する

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But 承認済みの "AR1" を 2 回交換する
- Then 1 回目のレスポンスには access_token が含まれる
- And 2 回目はエラー "InvalidGrantError"
- And 承認リクエスト "AR1" の状態は Consumed のままになる

### Example: EX-OAUTH2-042-05 承認済みの "AR1" を並行に 2 回交換する

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But 承認済みの "AR1" を並行に 2 回交換する
- Then ちょうど一方が成功し、もう一方はエラー "InvalidGrantError"

### Example: EX-OAUTH2-042-06 起票元でないクライアント "other-app" が "AR1" を交換する

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But 起票元でないクライアント "other-app" が "AR1" を交換する
- Then エラー "InvalidGrantError"

### Example: EX-OAUTH2-042-07 別テナントのトークン endpoint で "AR1" を交換する

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But 別テナントのトークン endpoint で "AR1" を交換する
- Then エラー "InvalidGrantError"

### Example: EX-OAUTH2-042-08 "AR1" の承認後に Agent が kill-switch で停止されている

- Given `agent-app` が起票した承認リクエスト `AR1` が存在する
- When `agent-app` が `auth_req_id=AR1` を交換する
- But "AR1" の承認後に Agent が kill-switch で停止されている
- Then エラー "InvalidGrantError"
- And トークンは発行されない

## Rule: REQ-OAUTH2-043 承認リクエストを判断できるのは対象ユーザー本人のステップアップ認証済みセッションだけである

Primary actor: `ResourceOwner`

### Example: EX-OAUTH2-043-01 通常経路

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- Then 承認リクエスト "AR1" の状態は Approved になる
- Then "BackchannelAuthApproved" が発行される

### Example: EX-OAUTH2-043-02 ステップアップ認証の有効期間を過ぎている

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- But ステップアップ認証の有効期間を過ぎている
- Then 操作を StepUpRequiredError で拒否し、承認リクエスト "AR1" の状態は Pending のままとなる

### Example: EX-OAUTH2-043-03 CSRF トークンが一致しない

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- But CSRF トークンが一致しない
- Then 操作は拒否され承認リクエスト "AR1" の状態は Pending のままになる

### Example: EX-OAUTH2-043-04 "alice" が他人宛の承認リクエスト "AR2" を判断する

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- But "alice" が他人宛の承認リクエスト "AR2" を判断する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-OAUTH2-043-05 既に終端状態の承認リクエストを判断する

- Given ユーザー "alice" 宛の承認リクエスト "AR1" が Pending で存在する
- And ユーザー "bob" 宛の承認リクエスト "AR2" が Pending で存在する
- And "alice" が認証済みで、ステップアップ認証の有効期間内にいる
- When "alice" が保留中の承認リクエスト一覧を取得する
- Then 一覧には "AR1" と、リクエスト元クライアントの表示名、Agent 名、要求スコープ、`authorization_details`、`binding_message` が含まれる
- Then 一覧に "AR2" と期限切れの承認リクエストは含まれない
- When "alice" が承認リクエスト "AR1" を承認する
- But 既に終端状態の承認リクエストを判断する
- Then 操作は InvalidRequestError で拒否され、記録済みの判断は上書きされない

## Rule: REQ-OAUTH2-044 Bearer 保護リソースの認証エラーはメタデータ URL を提示する

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-044-01 通常経路

- Given レルム "acme" の発行者は "https://idp.example.com/realms/acme" である
- When クライアントが無効なアクセストークンでレルム "acme" の保護 API を呼ぶ
- Then HTTP 401 を返し、WWW-Authenticate は Bearer の `error="invalid_token"` と `resource_metadata="https://idp.example.com/realms/acme/.well-known/oauth-protected-resource"` を引用符付きの auth-param として含む
- Then `resource_metadata` URL は、`resource` 未指定時のレルムの IdMagic API Protected Resource Metadata を返す

### Example: EX-OAUTH2-044-02 アクセストークンに必要なスコープがない

- Given レルム "acme" の発行者は "https://idp.example.com/realms/acme" である
- When クライアントが無効なアクセストークンでレルム "acme" の保護 API を呼ぶ
- But アクセストークンに必要なスコープがない
- Then HTTP 403 を返し、WWW-Authenticate に `error="insufficient_scope"` と必要なスコープ、および `resource_metadata="https://idp.example.com/realms/acme/.well-known/oauth-protected-resource"` を含める

### Example: EX-OAUTH2-044-03 レルム "acme" がホストルート形式のエンドポイントを使う

- Given レルム "acme" の発行者は "https://idp.example.com/realms/acme" である
- When クライアントが無効なアクセストークンでレルム "acme" の保護 API を呼ぶ
- But レルム "acme" がホストルート形式のエンドポイントを使う
- Then `resource_metadata` はホストルートの発行者配下にある `/.well-known/oauth-protected-resource` を指す

## Rule: REQ-OAUTH2-045 保護リソースの DPoP Proof は ath でアクセストークンに結び付けられる

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-045-01 通常経路

- Given DPoP 鍵 "K1" に結び付けられたアクセストークン "AT1" と "AT2" が存在する
- When クライアントが "AT1" を提示し、`ath` が "AT1" の base64url(SHA-256) である "K1" 署名の DPoP Proof で保護リソースを呼ぶ
- Then 要求は受理される

### Example: EX-OAUTH2-045-02 Proof が `ath` を含まない

- Given DPoP 鍵 "K1" に結び付けられたアクセストークン "AT1" と "AT2" が存在する
- When クライアントが "AT1" を提示し、`ath` が "AT1" の base64url(SHA-256) である "K1" 署名の DPoP Proof で保護リソースを呼ぶ
- But Proof が `ath` を含まない
- Then エラー "InvalidTokenError"

### Example: EX-OAUTH2-045-03 Proof の `ath` が "AT2" の base64url(SHA-256) である

- Given DPoP 鍵 "K1" に結び付けられたアクセストークン "AT1" と "AT2" が存在する
- When クライアントが "AT1" を提示し、`ath` が "AT1" の base64url(SHA-256) である "K1" 署名の DPoP Proof で保護リソースを呼ぶ
- But Proof の `ath` が "AT2" の base64url(SHA-256) である
- Then エラー "InvalidTokenError"

### Example: EX-OAUTH2-045-04 トークンエンドポイントへ `ath` を含まない "K1" 署名の DPoP Proof を提示する

- Given DPoP 鍵 "K1" に結び付けられたアクセストークン "AT1" と "AT2" が存在する
- When クライアントが "AT1" を提示し、`ath` が "AT1" の base64url(SHA-256) である "K1" 署名の DPoP Proof で保護リソースを呼ぶ
- But トークンエンドポイントへ `ath` を含まない "K1" 署名の DPoP Proof を提示する
- Then リクエストは受理され、アクセストークンが発行される

## Rule: REQ-OAUTH2-046 所有者がオフボードされた Agent は client_credentials で新しいトークンを取得できない

Primary actor: `RegisteredClient`

### Example: EX-OAUTH2-046-01 通常経路

- Given User "owner1" が所有する `Active` の Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And 管理者が "owner1" を無効化した
- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "InvalidClientError" で拒否され、トークンは発行されない
- Then 所有者の状態は "A1" の `status` を書き換えず、発行のたびに解決する（"A1" は `Active` のまま）

### Example: EX-OAUTH2-046-02 "owner1" がハード削除され解決できない

- Given User "owner1" が所有する `Active` の Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And 管理者が "owner1" を無効化した
- When "agent-app" として client_credentials でトークンを要求する
- But "owner1" がハード削除され解決できない
- Then エラー "InvalidClientError"
- And トークンは発行されない

### Example: EX-OAUTH2-046-03 "owner1" が再び有効化された

- Given User "owner1" が所有する `Active` の Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And 管理者が "owner1" を無効化した
- When "agent-app" として client_credentials でトークンを要求する
- But "owner1" が再び有効化された
- Then リクエストは受理され、アクセストークンが発行される

### Example: EX-OAUTH2-046-04 "agent-app" にどの Agent も束縛されていない

- Given User "owner1" が所有する `Active` の Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And 管理者が "owner1" を無効化した
- When "agent-app" として client_credentials でトークンを要求する
- But "agent-app" にどの Agent も束縛されていない
- Then 所有者の解決を行わず、リクエストは受理される

## Rule: REQ-OAUTH2-047 管理コンソールとアカウントポータルの Bearer 認証も失効判定を通る

Primary actor: `ResourceServer`

### Example: EX-OAUTH2-047-01 通常経路

- Given Agent "A1" に束縛されたクライアントへ発行済みの access トークン "AT1" がある
- And "A1" の revocation epoch が "AT1" の issued_at より後へ前進している
- When "AT1" を Bearer として `/api/admin/v1/` 配下の API へ提示する
- Then エラー "InvalidTokenError" で拒否される（イントロスペクションと同じ失効判定を通す）

### Example: EX-OAUTH2-047-02 "AT1" の jti が失効リストに載っている

- Given Agent "A1" に束縛されたクライアントへ発行済みの access トークン "AT1" がある
- And "A1" の revocation epoch が "AT1" の issued_at より後へ前進している
- When "AT1" を Bearer として `/api/admin/v1/` 配下の API へ提示する
- But "AT1" の jti が失効リストに載っている
- Then エラー "InvalidTokenError"

### Example: EX-OAUTH2-047-03 "AT1" が revocation epoch より後に発行されている

- Given Agent "A1" に束縛されたクライアントへ発行済みの access トークン "AT1" がある
- And "A1" の revocation epoch が "AT1" の issued_at より後へ前進している
- When "AT1" を Bearer として `/api/admin/v1/` 配下の API へ提示する
- But "AT1" が revocation epoch より後に発行されている
- Then 認証は成立し、以後はスコープとロールの境界で判定する

## Rule: REQ-OAUTH2-048 テナントが定めた委譲深さの上限を超えるトークン交換は拒否される

Primary actor: `Agent`

### Example: EX-OAUTH2-048-01 通常経路

- Given テナントが委譲深さの上限を設定している、または上書きを持たずシステム既定を継承している
- When エージェントが Token Exchange で委任トークンを要求する
- Then 監査イベントは発行トークンの深さと、判定に適用した上限の双方を残す

### Example: EX-OAUTH2-048-02 発行トークンの `act` 入れ子の深さが上限以内である

- Given テナントが委譲深さの上限を設定している、または上書きを持たずシステム既定を継承している
- When エージェントが Token Exchange で委任トークンを要求する
- But 発行トークンの `act` 入れ子の深さが上限以内である
- Then 交換は成立する

### Example: EX-OAUTH2-048-03 深さが上限を超える

- Given テナントが委譲深さの上限を設定している、または上書きを持たずシステム既定を継承している
- When エージェントが Token Exchange で委任トークンを要求する
- But 深さが上限を超える
- Then 交換を拒否し、拒否理由を監査へ残す

### Example: EX-OAUTH2-048-04 テナントの委譲ポリシーを解決できない

- Given テナントが委譲深さの上限を設定している、または上書きを持たずシステム既定を継承している
- When エージェントが Token Exchange で委任トークンを要求する
- But テナントの委譲ポリシーを解決できない
- Then システム既定へ退避せず拒否する

## Rule: REQ-OAUTH2-049 イントロスペクションと監査は同じ規則で委譲モードを示す

Primary actor: `ResourceServer`

### Example: EX-OAUTH2-049-01 通常経路

- Given Token Exchange で発行した委任トークンがある
- When リソースサーバーがそのトークンをイントロスペクトする
- Then レスポンスの委譲モードは、同じ交換が監査へ残したモードと一致する
- Then リソースサーバーは `act` と principal 種別から導出し直す必要がない

### Example: EX-OAUTH2-049-02 `act` に subject と異なる行為者がいる

- Given Token Exchange で発行した委任トークンがある
- When リソースサーバーがそのトークンをイントロスペクトする
- But `act` に subject と異なる行為者がいる
- Then 利用者の代理として返す

### Example: EX-OAUTH2-049-03 代行が無く subject が非人間のプリンシパルである

- Given Token Exchange で発行した委任トークンがある
- When リソースサーバーがそのトークンをイントロスペクトする
- But 代行が無く subject が非人間のプリンシパルである
- Then 自律実行として返す

### Example: EX-OAUTH2-049-04 代行が無く subject が人間の利用者である

- Given Token Exchange で発行した委任トークンがある
- When リソースサーバーがそのトークンをイントロスペクトする
- But 代行が無く subject が人間の利用者である
- Then 直接のアクセスとして返す

## Rule: REQ-OAUTH2-050 `Supervised` な Agent は人間の承認を経ずに新しいトークンを得られない

Primary actor: `Agent`

### Example: EX-OAUTH2-050-01 通常経路

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、"AgentApprovalRequired" が発行される
- When "agent-app" が "alice" 宛のバックチャネル認可を開始し、"alice" の承認後に `auth_req_id` を交換する
- Then アクセストークンが発行される (REQ-OAUTH2-041)
- Then "BackchannelAuthApproved" は承認の対象となった Agent の id を含む

### Example: EX-OAUTH2-050-02 "A1" の `kind` が `autonomous` である

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- But "A1" の `kind` が `autonomous` である
- Then リクエストは受理され、アクセストークンが発行される

### Example: EX-OAUTH2-050-03 "A1" の `kind` が既知のどの値でもない

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- But "A1" の `kind` が既知のどの値でもない
- Then 承認が必要な側へ倒し、エラー "UnauthorizedClientError"

### Example: EX-OAUTH2-050-04 "agent-app" にどの Agent も束縛されていない

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- But "agent-app" にどの Agent も束縛されていない
- Then 区分の判定を行わず、リクエストは受理される

### Example: EX-OAUTH2-050-05 ワークロード ID 連携の attestation が "A1" の client へ写る交換である

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- But ワークロード ID 連携の attestation が "A1" の client へ写る交換である
- Then エラー "UnauthorizedClientError"

### Example: EX-OAUTH2-050-06 `subject_token` が承認を経て "A1" へ発行済みのトークンである

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- But `subject_token` が承認を経て "A1" へ発行済みのトークンである
- Then エラー "UnauthorizedClientError"
- And 一つの承認は一つのトークンに対応し、派生トークンへは継承しない

### Example: EX-OAUTH2-050-07 交換に関与するどの Agent も `autonomous` である

- Given `kind` が `supervised` の `Active` な Agent "A1" が confidential クライアント "agent-app" に束縛されている
- And "A1" の所有者は active User "alice" である
- When "agent-app" として client_credentials でトークンを要求する
- Then エラー "UnauthorizedClientError" で拒否され、トークンは発行されない
- Then "AgentApprovalRequired" が発行され、判断の根拠とした区分が残る
- When "agent-app" が Token Exchange で委任トークンを要求する
- But 交換に関与するどの Agent も `autonomous` である
- Then 交換は成立する
