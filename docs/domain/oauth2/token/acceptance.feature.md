# Feature: トークンの例

## Rule: REQ-OAUTH2-001 ユーザーに紐づく OAuth グラントは account スコープのアクセストークンを発行できる

### Example: EX-OAUTH2-001-01 通常経路

- Given クライアントは `account:read` と `account:write` を許可スコープとして登録している
- And 有効な User が Authorization Code + PKCE または Device Authorization で `account:read` に同意している
- When クライアントがユーザーに紐づくグラントを `/token` で交換する
- Then アクセストークンの `sub` は同意した User、audience はレルムの IdMagic API（レルムの発行者識別子）、スコープは `account:read` になる
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

### Example: EX-OAUTH2-001-04 audience がレルムの IdMagic API ではない account スコープのトークンを提示する

- Given 有効な User を subject とし、スコープに `account:read` を持つアクセストークンがある
- But アクセストークンの audience はリクエスト先レルムの IdMagic API を含まない
- When クライアントがそのトークンで account リソースサーバーを参照する
- Then account リソースサーバーは 401 の InvalidAccessTokenError で拒否する
- Then ポータル境界のスコープだけを持ち account スコープを持たないトークンは、この audience の検査を受けない

## Rule: REQ-OAUTH2-006 リフレッシュトークンをローテーションして新しいトークンを得る

### Example: EX-OAUTH2-006-01 通常経路

- Given 有効な refresh トークン "RT1" が存在する
- When リフレッシュトークン "RT1" を交換する
- Then レスポンスに新しい access_token と refresh_token が含まれる
- Then "RT1" の状態は "Rotated"
- Then "RefreshTokenRotated" が発行される
- Then "AccessTokenIssued" が発行される

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

## Rule: REQ-OAUTH2-010 DPoP 証明付き要求はセンダー制約付きトークンを発行する

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

### Example: EX-OAUTH2-011-01 通常経路

- Given 失効済み access トークン "AT1" が存在する
- When トークン "AT1" を検査する
- Then レスポンスは active=false のみで他のフィールドを含まない

## Rule: REQ-OAUTH2-012 キルスイッチ作動後の Agent トークンはイントロスペクションで active=false になる

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

## Rule: REQ-OAUTH2-018 絶対有効期限を過ぎたリフレッシュトークンはローテーションできない

### Example: EX-OAUTH2-018-01 通常経路

- Given absolute_expires_at "2026-01-01T00:00:00Z" の refresh トークン "RT1" が存在する
- And 現在時刻は "2026-01-02T00:00:00Z" である
- When クライアントがリフレッシュトークン "RT1" を交換する
- Then エラー "InvalidGrantError"

## Rule: REQ-OAUTH2-019 クライアントは自分のトークンを失効できる

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

## Rule: REQ-OAUTH2-026 client_credentials グラントで M2M トークンが発行される

### Example: EX-OAUTH2-026-01 通常経路

- Given confidential クライアント "backend" が grant_types に "client_credentials" を含めて登録済みである
- When "backend" として client_credentials で scope "api:read" のトークンを取得する
- Then レスポンスに access_token が含まれ refresh_token は含まれない
- Then 発行された access_token の sub は client_id と一致する
- Then "AccessTokenIssued" が発行される
- Given public クライアント "spa-app" が grant_types に "client_credentials" を含めて登録済みである
- When "spa-app" として client_credentials でトークンを要求する
- Then client_credentials は confidential 限定であるため UnauthorizedClientError で拒否され、トークンは発行されない

## Rule: REQ-OAUTH2-029 mTLS バインド AT は同じ証明書のリクエストでのみ受理される

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

## Rule: REQ-OAUTH2-034 トークンエンドポイントはテナント境界を越えた資格情報を受理しない

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

## Rule: REQ-OAUTH2-039 KeyProvider の障害時は新しいトークンの発行を拒否する

### Example: EX-OAUTH2-039-01 通常経路

- Given tenant_id "acme" の KeyProvider が到達不能である
- When tenant_id "acme" のクライアントがトークン発行を要求する
- Then 新規署名は行われずエラー "ServerError" で拒否される

## Rule: REQ-OAUTH2-044 Bearer 保護リソースの認証エラーはメタデータ URL を提示する

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

### Example: EX-OAUTH2-048-01 通常経路

- Given テナントが委譲深さの上限を設定している、または上書きを持たずシステムデフォルトを継承している
- When エージェントが Token Exchange で委任トークンを要求する
- Then 監査イベントは発行トークンの深さと、判定に適用した上限の双方を残す

### Example: EX-OAUTH2-048-02 発行トークンの `act` 入れ子の深さが上限以内である

- Given テナントが委譲深さの上限を設定している、または上書きを持たずシステムデフォルトを継承している
- When エージェントが Token Exchange で委任トークンを要求する
- But 発行トークンの `act` 入れ子の深さが上限以内である
- Then 交換は成立する

### Example: EX-OAUTH2-048-03 深さが上限を超える

- Given テナントが委譲深さの上限を設定している、または上書きを持たずシステムデフォルトを継承している
- When エージェントが Token Exchange で委任トークンを要求する
- But 深さが上限を超える
- Then 交換を拒否し、拒否理由を監査へ残す

### Example: EX-OAUTH2-048-04 テナントの委譲ポリシーを解決できない

- Given テナントが委譲深さの上限を設定している、または上書きを持たずシステムデフォルトを継承している
- When エージェントが Token Exchange で委任トークンを要求する
- But テナントの委譲ポリシーを解決できない
- Then システムデフォルトへ退避せず拒否する

## Rule: REQ-OAUTH2-049 イントロスペクションと監査は同じ規則で委譲モードを示す

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
