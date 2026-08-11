# OAuth2 Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-OAUTH2-001: user-bound OAuth grantはaccount scopeのaccess tokenを発行できる
- Actor: RegisteredClient
- Given: client は account:read と account:write を許可 scope として登録している
- Given: active User が Authorization Code + PKCE または Device Authorization で account:read に同意している
- Then: client が user-bound grant を /token で交換する
- Then: access token の sub は同意した User、audience は realm の IdMagic API、scope は account:read になる
- Then: account resource server は token の subject 本人に限って read 操作を許可する
- Alternative (client_credentials または User subject のない token exchange で account scope を要求する): token request は InvalidScopeError で拒否される
- Alternative (client の許可 scope または User consent に account scope が含まれない): account scope は発行されない

### REQ-OAUTH2-002: API token発行者はaccount consent scope内で自身のconsentだけを操作できる
- Actor: SelfApiClient
- Given: client は対象 tenant の active User に固定された有効な API access token を提示している
- Then: account:read scope で自身の active consent を参照できる
- Then: account:consents:write scope で自身の consent を撤回できる
- Alternative (account:read だけで consent revoke を要求する): 操作は AccessDeniedError で拒否される
- Alternative (token の tenant または user_id が操作対象と一致しない): 操作は AccessDeniedError で拒否される

### REQ-OAUTH2-003: management API clientはOAuth resourceごとのscope内だけを操作できる
- Actor: ManagementApiClient
- Given: client は対象 tenant の有効な API access token を提示している
- Then: oauth-clients:read scope で OAuth2 client を参照できる
- Then: authorization-detail-types:write scope で authorization detail type を変更できる
- Then: mcp-resource-servers:read scope で MCP resource server を参照できる
- Alternative (oauth-clients:read だけで OAuth2 client の変更を要求する): 操作は AccessDeniedError で拒否される
- Alternative (別 resource の scope で操作を要求する): 操作は AccessDeniedError で拒否される
- Alternative (token の tenant と request tenant が一致しない): 操作は AccessDeniedError で拒否される

### REQ-OAUTH2-004: 管理者は自身に可視なrole policyを確認できる
- Actor: TenantAdministrator
- Given: roles=["admin"] の管理者が認証済みである
- Then: 管理者が role policy 一覧を取得する
- Then: 応答には可視な role、permission、対応 HTTP interface が含まれる
- Alternative (principal が admin または system_admin ではない): role policy 一覧は AccessDeniedError で拒否される

### REQ-OAUTH2-005: 認可コードフローでアクセストークンと ID トークンを取得できる
- Actor: RegisteredClient
- Given: "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- Given: ユーザー "alice" は "web-app" に scope "openid profile" を同意済みである
- Then: "web-app" として scope "openid profile" で認可リクエストを送る
- Then: 発行された認可コードを正しい PKCE verifier で交換する
- Then: 応答に access_token・id_token・refresh_token が含まれ token_type は Bearer
- Then: "UserAuthenticated" が発行される
- Then: "AuthorizationCodeIssued" が発行される
- Then: "AuthorizationCodeRedeemed" が発行される
- Then: "AccessTokenIssued" が発行される
- Then: "RefreshTokenIssued" が発行される
- Alternative (認可リクエストの redirect_uri が未登録である): リダイレクトは行われず IdP がエラーページを表示する → エラー "InvalidRequestError"
- Alternative (単一値認可parameterが重複する、promptに重複または未対応tokenがある、またはnoneが他のprompt tokenと併用される): 認可コードは発行されない → 安全に確定済みの登録redirect_uriがあればstateとissuer識別子を含むinvalid_requestを返す → それ以外はリダイレクトせず IdP がエラーページを表示する
- Alternative (request_uriと許容されないfront-channel authorization parameterが混在する): 認可コードは発行されない → エラー "InvalidRequestError"
- Alternative (promptがnoneで既存セッションまたは必要な同意がない): UIおよびログインリダイレクトは発生しない → 既存セッションがなければstateとissuer識別子を含むlogin_requiredを登録redirect_uriへ返す → 同意がなければstateとissuer識別子を含むconsent_requiredを登録redirect_uriへ返す
- Alternative (PKCE verifier が一致しない): 認可コードを誤った code_verifier で交換する → エラー "InvalidGrantError" → トークンは発行されない
- Alternative (同じ認可コードを 2 回交換する): 1 回目の応答には access_token が含まれる → 2 回目はエラー "InvalidGrantError" → 発行ファミリーのトークンがすべて失効する → "RefreshTokenReuseDetected" が発行される → "TokenRevoked" が発行される
- Alternative (認可コードが発行から 60 秒を超えている): 認可コードの交換はエラー "InvalidGrantError" → 認可コードの状態は Expired になる

### REQ-OAUTH2-006: リフレッシュトークンをローテーションして新しいトークンを得る
- Actor: RegisteredClient
- Given: 有効な refresh token "RT1" が存在する
- Then: リフレッシュトークン "RT1" を交換する
- Then: 応答に新しい access_token と refresh_token が含まれる
- Then: "RT1" の状態は "Rotated"
- Then: "RefreshTokenRotated" が発行される
- Then: "AccessTokenIssued" が発行される
- Then: "RefreshTokenIssued" が発行される
- Alternative (ローテーション済みの旧 refresh token を再使用する): family_id "F1" の refresh token "RT1" をローテーション後に再使用する → 再使用はエラー "InvalidGrantError" → "RT1" の状態は "Revoked" → family_id "F1" のトークンがすべて失効する → "RefreshTokenReuseDetected" が発行される → "TokenRevoked" が発行される

### REQ-OAUTH2-007: 不正なクライアント認証は invalid_client で一律拒否される
- Actor: RegisteredClient
- Then: 既知のクライアントを誤った client_secret で認可コードを交換する
- Then: エラー "InvalidClientError"
- Alternative (未知の client_id で交換する): 未知の client_id で認可コードを交換する → エラー "InvalidClientError"

### REQ-OAUTH2-008: 既存同意の有無に応じて同意画面を出し分ける
- Actor: ResourceOwner
- Given: ユーザー "alice" が "web-app" に scope "openid profile" を同意済みである
- Then: "web-app" として scope "openid profile" で認可リクエストを送る
- Then: 認可リクエストの状態は Consented
- Then: 同意 UI は表示されない
- Alternative (prompt=consent で再同意を要求する): "web-app" として prompt "consent" で認可リクエストを送る → 認可リクエストの状態は ConsentPending

### REQ-OAUTH2-009: PAR で送信した認可リクエストを request_uri 経由で実行する
- Actor: RegisteredClient
- Given: クライアント "web-app" が存在する
- Then: "web-app" として認可リクエストを事前送信する
- Then: PAR 応答に request_uri が含まれ expires_in は 600 以下
- Then: request_uri "<返された値>" で認可リクエストを送る
- Then: その PAR レコードの状態は "Used"
- Then: "PARStored" が発行される
- Then: "AuthorizationCodeIssued" が発行される
- Alternative (PAR 必須の FAPI クライアントが PAR なしで直接送信する): PAR 必須の FAPI クライアント "fapi-app" として scope "openid" で直接認可リクエストを送る → エラー "InvalidRequestError"

### REQ-OAUTH2-010: DPoP 証明付き要求はセンダー制約付きトークンを発行する
- Actor: RegisteredClient
- Then: 有効な DPoP 証明を付けて認可コードを交換する
- Then: 発行された access token は DPoP 鍵サムプリントに cnf でバインドされる
- Then: 発行された refresh token のセンダー制約は Dpop
- Alternative (iat が 60 秒以上古い DPoP 証明である): iat "2026-01-01T00:00:00Z" の DPoP 証明を時刻 "2026-01-01T00:01:30Z" で付けて認可コードを交換する → エラー "InvalidDpopProofError"
- Alternative (同一 DPoP jti を再使用する): jti "ABC" の DPoP 証明を付けて認可コードを交換する → 同じ jti "ABC" の DPoP 証明を付けて認可コードを交換する → 1 回目の応答には access_token が含まれる → 2 回目はエラー "InvalidDpopProofError"

### REQ-OAUTH2-011: 失効済みトークンの introspection は active=false のみ返す
- Actor: ResourceServer
- Given: 失効済み access token "AT1" が存在する
- Then: トークン "AT1" を検査する
- Then: 応答は active=false のみで他のフィールドを含まない

### REQ-OAUTH2-012: kill-switch後のAgentトークンはintrospectionでactive=falseになる
- Actor: ResourceServer
- Given: Agent "A1" に issued_at が古い access token "AT1" が発行済みである
- Given: "A1" は kill-switch により revocation epoch が "AT1" の issued_at より後へ前進している
- Then: トークン "AT1" を検査する
- Then: 応答は active=false のみで他のフィールドを含まない
- Alternative ("AT1" が revocation epoch より後に発行された (kill 後に再発行された) token である): 応答は通常どおり active=true と claim を返す

### REQ-OAUTH2-013: UserInfo は openid スコープのトークンに sub を返す
- Actor: RegisteredClient
- Given: scope "openid profile" の access token "AT1" が存在する
- Then: トークン "AT1" でユーザー情報を取得する
- Then: 応答に sub・name・preferred_username が含まれる
- Alternative (openid スコープを持たないトークンで取得する): scope "profile" のみの access token "AT1" でユーザー情報を取得する → エラー "InsufficientScopeError"

### REQ-OAUTH2-014: Discovery 文書は宣言された全エンドポイントを広告する
- Actor: RegisteredClient
- Then: Discovery 文書を取得する
- Then: 応答に issuer・authorization_endpoint・token_endpoint・userinfo_endpoint・jwks_uri・introspection_endpoint・revocation_endpoint・pushed_authorization_request_endpoint・device_authorization_endpoint・registration_endpoint が含まれる

### REQ-OAUTH2-015: 認可コードの並行交換はちょうど一方だけ成功する
- Actor: RegisteredClient
- Given: 発行済み認可コード "AC1"（family_id "F1"）が存在する
- Then: 認可コード "AC1" を verifier "v" で並行に 2 回交換する
- Then: ちょうど一方が成功し、もう一方はエラー "InvalidGrantError"
- Then: family_id "F1" のトークンがすべて失効する
- Then: "AuthorizationCodeRedeemed" が発行される
- Then: "RefreshTokenReuseDetected" が発行される
- Then: "TokenRevoked" が発行される

### REQ-OAUTH2-016: 動的クライアント登録は client_id を採番して返す
- Actor: Client
- Then: confidential クライアント "web-app" を redirect_uri "https://app.example.com/callback" で登録する
- Then: 応答に client_id と client_secret が含まれる
- Then: "ClientRegistered" が発行される
- Alternative (redirect_uri を持たない登録要求である): confidential クライアント "web-app" を redirect_uri "" で登録する → エラー "InvalidRequestError"

### REQ-OAUTH2-017: Client metadata fetchは公開IPへ直接接続する
- Actor: RegisteredClient
- Given: client_id は public IP に解決される client 所有の HTTPS metadata URL である
- Given: Authorization Server の環境に HTTPS proxy が設定されている
- Then: Authorization Server は環境 proxy を使用せず、DNS 検査済みの public IP へ直接接続する
- Then: metadata document の fetch と検証に成功する
- Alternative (metadata host が private、loopback、link-local、または CGNAT 100.64.0.0/10 の IP に解決される): Authorization Server は対象 IP へ接続しない → client metadata の解決を fail-closed で拒否する

### REQ-OAUTH2-018: absolute_expires_at を超えた refresh token はローテーション不可
- Actor: RegisteredClient
- Given: absolute_expires_at "2026-01-01T00:00:00Z" の refresh token "RT1" が存在する
- Then: 時刻が "2026-01-02T00:00:00Z" になる
- Then: リフレッシュトークン "RT1" を交換する
- Then: エラー "InvalidGrantError"

### REQ-OAUTH2-019: クライアントは自分のトークンを失効できる
- Actor: RegisteredClient
- Given: 有効な refresh token "RT1" が存在する
- Then: トークン "RT1" を失効させる
- Then: "RT1" の状態は "Revoked"
- Then: "TokenRevoked" が発行される
- Alternative (所有者でないクライアントが失効を要求する): クライアント "client-A" が所有する refresh token "RT1" に対し "client-B" として失効を要求する → 盗難検知防止のため 200 OK のみ返り "RT1" の状態は "Active" のまま

### REQ-OAUTH2-020: 失効した access_token でユーザー情報取得は invalid_token で拒否される
- Actor: RegisteredClient
- Given: 有効な access token "AT1" が存在する
- Then: トークン "AT1" を失効させる
- Then: トークン "AT1" でユーザー情報を取得する
- Then: エラー "InvalidTokenError"

### REQ-OAUTH2-021: refresh_token は offline_access スコープ付与時のみ発行される
- Actor: RegisteredClient
- Given: confidential クライアント "web-app" が grant_types に "authorization_code"・"refresh_token" を含めて登録済みである
- Then: "web-app" として scope "openid offline_access" で認可リクエストを送る
- Then: 発行された認可コードを verifier "v" で交換する
- Then: 応答に refresh_token が含まれる
- Then: "RefreshTokenIssued" が発行される
- Alternative (offline_access を要求しない): "web-app" として scope "openid profile" で認可リクエストを送る → 発行された認可コードを verifier "v" で交換する → 応答に refresh_token は含まれない

### REQ-OAUTH2-022: 認可リクエストの nonce は ID トークンに伝播する
- Actor: RegisteredClient
- Then: "web-app" として scope "openid"、nonce "n-12345" で認可リクエストを送る
- Then: 発行された認可コードを verifier "v" で交換する
- Then: 応答の id_token の nonce クレームは "n-12345"

### REQ-OAUTH2-023: RP-Initiated Logout は登録済み post_logout_redirect_uri にだけ戻す
- Actor: ResourceOwner
- Given: confidential クライアント "web-app" が redirect_uri "https://app.example.com/cb" で登録済みである
- Then: "web-app" として post_logout_redirect_uri "https://app.example.com/cb" でログアウトする
- Then: state が post_logout_redirect_uri に伝播する
- Alternative (未登録の post_logout_redirect_uri を指定する): "web-app" として post_logout_redirect_uri "https://evil.example.com/cb" でログアウトする → エラー "InvalidRequestError"

### REQ-OAUTH2-024: RP-Initiated Logout はid_token_hintからsessionとclientを解決する
- Actor: ResourceOwner
- Given: ユーザー "alice" が "web-app" として認可コードを交換し sid 付きの ID Token を持つ
- Then: "alice" が発行済み ID Token を id_token_hint として /end_session を呼ぶ
- Then: id_token_hint の sid が示す LoginSession が失効する
- Then: 同じ sid を持つ全 client の RefreshTokenRecord が Revoked へ遷移する
- Alternative (id_token_hint の aud が指定 client_id と一致しない): client_id "other-app" と "web-app" 発行の ID Token を id_token_hint に付けて /end_session を呼ぶ → エラー "InvalidRequestError"
- Alternative (id_token_hint の署名がidmagicの署名鍵で検証できない): 他 issuer が署名した JWT を id_token_hint に付けて /end_session を呼ぶ → エラー "InvalidRequestError"
- Alternative (id_token_hint が期限切れ (exp 経過) である): 期限切れの発行済み ID Token を id_token_hint として /end_session を呼ぶ → exp 切れのみを理由にした拒否はされず sid による session 解決が成功する

### REQ-OAUTH2-025: session revokeはbackchannel_logout_uri登録済みRPへlogout tokenを配送する
- Actor: ResourceOwner
- Given: "web-app" が backchannel_logout_uri "https://app.example.com/backchannel_logout" を登録済みである
- Given: ユーザー "alice" が "web-app" とのブラウザ session を持つ
- Then: "alice" が /end_session でログアウトする
- Then: 対象 sid と "web-app" の LogoutNotification が作成される
- Then: 署名済み logout token が backchannel_logout_uri へ配送され Delivered になる
- Alternative (配送が一時的に失敗する (5xx / timeout)): LogoutNotification は Pending のまま attempts が増え再試行される → ローカルの session/refresh token 失効は取り消されない
- Alternative (max_attempts まで再試行しても配送が成功しない): LogoutNotification は Failed (dead-letter) に確定する → ローカルの session/refresh token 失効は取り消されない

### REQ-OAUTH2-026: client_credentials グラントで M2M トークンが発行される
- Actor: RegisteredClient
- Given: confidential クライアント "backend" が grant_types に "client_credentials" を含めて登録済みである
- Then: "backend" として client_credentials で scope "api:read" のトークンを取得する
- Then: 応答に access_token が含まれ refresh_token は含まれない
- Then: 発行された access_token の sub は client_id と一致する
- Then: "AccessTokenIssued" が発行される
- Alternative (public クライアントが client_credentials を登録しようとする): public クライアント "spa-app" を grant_types に "client_credentials" 含めて登録する → client_credentials は confidential 限定でありエラー "InvalidRequestError"

### REQ-OAUTH2-027: デバイス認可フローでアクセストークンを取得できる
- Actor: RegisteredClient
- Given: confidential クライアント "tv-app" が grant_types に "urn:ietf:params:oauth:grant-type:device_code" を含めて登録済みである
- Then: "tv-app" として scope "openid profile" でデバイス認可を開始する
- Then: 応答に device_code・user_code・verification_uri・interval が含まれる
- Then: ユーザー "alice" が verification_uri で user_code を入力し承認する
- Then: device_code "DC1" を交換する
- Then: 応答に access_token と id_token が含まれる
- Then: "DeviceAuthorizationRequested" が発行される
- Then: "DeviceAuthorizationApproved" が発行される
- Then: "AccessTokenIssued" が発行される
- Alternative (ユーザー承認前にポーリングする): Issued 状態の device_code "DC1" を交換する → エラー "AuthorizationPendingError"
- Alternative (ポーリング間隔より短い再試行をする): interval 5 秒の device_code "DC1" を Issued 状態で用意する → device_code "DC1" を交換し "2s" 経過後に再度交換する → 2 回目はエラー "SlowDownError"
- Alternative (device_code が expires_in を超えている): issued_at "2026-01-01T00:00:00Z"・expires_at "2026-01-01T00:10:00Z" の device_code "DC1" を時刻 "2026-01-01T00:11:00Z" で交換する → エラー "ExpiredTokenError"

### REQ-OAUTH2-028: 改ざんされた client_assertion は invalid_client で拒否される
- Actor: RegisteredClient
- Given: confidential クライアント "fapi-app" が token_endpoint_auth_method "private_key_jwt"・jwks 登録で存在する
- Then: 認可コード "AC1" を verifier "v"・改ざんされた client_assertion で交換する
- Then: エラー "InvalidClientError"

### REQ-OAUTH2-029: mTLS バインド AT は同じ証明書のリクエストでのみ受理される
- Actor: RegisteredClient
- Given: confidential クライアント "mtls-app" が token_endpoint_auth_method "tls_client_auth" で存在する
- Then: mTLS 証明書バインドされた access_token を発行する
- Then: 同じ証明書を提示して userinfo を取得すると 200 を返す
- Alternative (別の証明書を提示する): 別の証明書を提示すると invalid_token で拒否される

### REQ-OAUTH2-030: RFC 8414 メタデータ文書は OIDC Discovery と同等の内容を返す
- Actor: RegisteredClient
- Then: Authorization Server メタデータを取得する
- Then: 応答に issuer・authorization_endpoint・token_endpoint・jwks_uri・grant_types_supported が含まれる

### REQ-OAUTH2-031: 管理者は所属テナントの同意を参照・撤回できるが付与は代行できない
- Actor: TenantAdministrator
- Given: tenant_id "acme" の roles=["admin"] のユーザー "operator" が認証済みである
- Given: tenant_id "acme" のユーザー "alice" と client "portal" の Consent が Granted で存在する
- Then: 管理者 "operator" が Consent 一覧と単一 Consent を取得する
- Then: 管理者 "operator" がユーザー "alice" と client "portal" の Consent を撤回する
- Then: Consent の state は Revoked で revoked_at が記録される
- Then: "ConsentRevoked" が actorUserId "operator" で発行される
- Then: 管理者が Consent を作成または scope 拡張する interface は存在しない

### REQ-OAUTH2-032: ユーザーは接続済みアプリの同意を自分で撤回できる
- Actor: ResourceOwner
- Given: ユーザー "alice" が client "web-app" に scope "openid profile" を同意済みである
- Given: ユーザー "alice" が認証済みで接続済みアプリ画面を開いている
- Then: 一覧に "web-app" が表示される
- Then: ユーザー "alice" が "web-app" の同意を撤回する
- Then: Consent の state は Revoked になり一覧から消える

### REQ-OAUTH2-033: realm prefix 付き Discovery は同 prefix の issuer を返す
- Actor: RegisteredClient
- Given: tenant_id "acme" が Active で存在する
- Then: /realms/acme/.well-known/openid-configuration を取得する
- Then: 応答の issuer は base URL + /realms/acme
- Then: 応答の authorization_endpoint は base URL + /realms/acme/authorize

### REQ-OAUTH2-034: トークンエンドポイントはテナント境界を越えた資格情報を受理しない
- Actor: RegisteredClient
- Then: tenant_id "acme" で発行した認可コード "AC1" を "/realms/default/token" で交換する
- Then: エラー "InvalidGrantError"
- Alternative (他テナントの client_id を使う): tenant_id "acme" に登録した client_id "web-app" で "/realms/default/token" に交換を要求する → エラー "InvalidClientError"
- Alternative (他テナントのリフレッシュトークンを再発行する): tenant_id "acme" で発行した refresh token "RT1" を "/realms/default/token" で再発行する → エラー "InvalidGrantError"
- Alternative (保存層に別テナントの subject / client を持つ refresh token を書き込む): tenant_id "acme" の refresh token に tenant_id "default" の client_id または sub を指定する → 永続化層が参照整合性エラーで拒否する
- Alternative (他テナントの device_code を交換する): tenant_id "acme" で発行し承認した device_code "DC1" を "/realms/default/token" で交換する → エラー "InvalidGrantError"

### REQ-OAUTH2-035: 管理者は所属テナントのclientを作成・更新・削除できる
- Actor: TenantAdministrator
- Given: tenant_id "acme" の admin "operator" が認証済みである
- Then: "operator" が client "portal" を作成し client_secret を一度だけ受け取る
- Then: "operator" が client "portal" の redirect_uris を更新する
- Then: "operator" が client "portal" を削除する
- Then: "AdminOAuth2ClientCreated"、"AdminOAuth2ClientUpdated"、"AdminOAuth2ClientDeleted" が発行される
- Alternative (別テナントの client を取得する): 別テナントの管理者が client_id を指定して取得する → エラー "InvalidRequestError"

### REQ-OAUTH2-036: 管理者はApplicationから期限付きclient secretを追加発行して個別失効できる
- Actor: TenantAdministrator
- Given: tenant_id "acme" の Application "billing" は client_secret_basic の confidential OIDC client を binding として持つ
- Given: 期限なし legacy secret "S1" が Active である
- Then: 管理者が expires_in_days 90 で新 secret を追加発行する
- Then: 応答で新 secret を一度だけ受け取り、metadata は90日後の expires_at と Active 状態を持つ
- Then: 追加発行は既存 secret の期限と状態を変更しない
- Then: 新 secret と旧 secret の両方が token endpoint 認証へ成功する
- Then: 管理者が旧 credential だけを個別失効する
- Then: 旧 secret は InvalidClientError で拒否され、新 secret は引き続き認証へ成功する
- Then: ClientSecretIssued と ClientSecretRevoked が actor、client、credential、expiry の非機密 metadata だけを含んで発行される
- Alternative (expires_in_days が1..730の範囲外である): エラー "InvalidRequestError"
- Alternative (Active credential が既に2件存在する): 追加発行はエラー "ClientSecretLimitExceededError" で拒否され、既存 credential は変更されない
- Alternative (client が private_key_jwt、mTLS、または public client である): エラー "InvalidRequestError"
- Alternative (別 client の credential_id または存在しない credential_id を失効する): エラー "InvalidRequestError"
- Alternative (既に Revoked の credential を再び失効する): 冪等に成功し、ClientSecretRevoked は重複発行されない

### REQ-OAUTH2-037: 管理者は互換interfaceでclient secretを無停止rotationできる
- Actor: TenantAdministrator
- Given: tenant_id "acme" の Application "billing" は client_secret_basic の confidential OIDC client を binding として持つ
- Given: 旧 secret "S1" が有効である
- Then: 管理者が grace_days 7 で secret を rotation する
- Then: 応答で新 secret を一度だけ受け取る
- Then: 新 secret と旧 secret は grace_until より前に token endpoint 認証へ成功する
- Then: grace_until より後は旧 secret が InvalidClientError で拒否される
- Then: ClientSecretRotated が actor、client、grace_until だけを含んで発行される
- Alternative (grace_days が 0 以外で 1..30 の範囲外である): エラー "InvalidRequestError"
- Alternative (client が private_key_jwt、mTLS、または public client である): エラー "InvalidRequestError"

### REQ-OAUTH2-038: 管理consent APIは別テナントの同意を公開しない
- Actor: TenantAdministrator
- Given: tenant_id "acme" の user と client の Consent が存在する
- Then: tenant_id "default" の管理者が同じ user_id と client_id の Consent を取得する
- Then: エラー "InvalidRequestError"

### REQ-OAUTH2-039: KeyProvider障害時は新規token発行を拒否する
- Actor: RegisteredClient
- Given: tenant_id "acme" の KeyProvider が到達不能である
- Then: tenant_id "acme" の client が token 発行を要求する
- Then: 新規署名は行われずエラー "ServerError" で拒否される

### REQ-OAUTH2-040: protocol endpointは閾値超過リクエストをrate limitで拒否する
- Actor: RegisteredClient
- Given: client がある endpoint の EndpointRateLimitPolicy の window 内で許容 max_requests に到達している
- Then: 同一 window 内で追加リクエストを送る
- Then: エラー "RateLimitedError" (HTTP 429、Retry-After ヘッダ付き)
- Alternative (対象 endpoint が /token である): client_id と IP の組で閾値超過している状態で token を要求する → エラー "RateLimitedError"
- Alternative (対象 endpoint が /authorize または /par である): IP と client_id の組で閾値超過している状態で認可リクエストを送る → エラー "RateLimitedError"
- Alternative (対象 endpoint が /device_authorization である): client_id と IP の組で閾値超過している状態でデバイス認可を開始する → エラー "RateLimitedError"
- Alternative (共有カウンタストアに到達できない): リクエストは fail-closed で "RateLimitedError" として拒否される

### REQ-OAUTH2-041: RegisterClient
Dynamic Client Registration (RFC 7591)

### REQ-OAUTH2-042: Authorize
Authorization endpoint (RFC 6749 §4.1.1)。client_id は登録済み Client、または Client ID Metadata Document の URL を都度解決したものを受理する。
- Precondition: !context.rate_limited
- Precondition: every_authorization_parameter_has_single_value(input.params)
- Precondition: prompt_tokens_supported_and_unambiguous(input.params.prompt)
- Precondition: client_resolvable(input.params.client_id)
- Precondition: redirect_uri_exact_match(input.params.client_id, input.params.redirect_uri)
- Precondition: requested_scopes_allowed(input.params.client_id, input.params.scope)
- Precondition: pkce_s256_present(input.params)
- Precondition: par_requirement_satisfied(input.params.client_id, input.params.request_uri)
- Precondition: request_uri_is_the_only_authorization_request_source_when_present(input.params)
- Precondition: application_assignment_permits(context.tenant_id, subject.id, input.params.client_id)
- Precondition: application_sign_in_policy_satisfied(context.tenant_id, subject.id, input.params.client_id)
- Precondition: resource_indicator_registered_and_active(context.tenant_id, input.params.resource, input.params.scope)

### REQ-OAUTH2-043: ResumeFederatedAuthorization
Authentication context の inbound federation callback が認証済み session を確立した後、保存済み authorization transaction を再開して通常の application policy・consent・required-action gate を適用する。
- Precondition: valid_authorization_transaction(context.transaction_cookie)
- Precondition: application_assignment_permits(context.tenant_id, subject.id, context.client_id)
- Precondition: application_sign_in_policy_satisfied(context.tenant_id, subject.id, context.client_id)

### REQ-OAUTH2-044: GetBrowserTransaction
SPA が HttpOnly transaction cookie に対応するログイン・同意コンテキストを取得する。

### REQ-OAUTH2-045: SubmitBrowserConsent
SPA JSON API から authorization request の同意・拒否を確定する。
- Precondition: valid_authorization_transaction(context.transaction_cookie)

### REQ-OAUTH2-046: SubmitBrowserDevice
SPA JSON API から device user_code を承認または拒否する。

### REQ-OAUTH2-047: GetBrowserDeviceContext
SPA が device user_code と CSRF token を取得する。

### REQ-OAUTH2-048: EndSession
RP-Initiated Logout endpoint (OIDC RP-Initiated Logout 1.0)。redirect 検証は
OAuth2/OIDC bounded context が行い、session revocation は Authentication bounded
context の SessionManager に委譲する。id_token_hint が与えられれば署名・iss・aud・sub・
sid を検証して対象 session と client を hint から解決し、client_id と
矛盾する hint は拒否する。id_token_hint が無ければ client_id + browser cookie による
既存の解決方法を fallback として維持する。ローカル revoke 確定後、対象 sid に紐づく
ClientSession から前段の FrontChannelLogout target 一覧を応答へ含め、
BackChannelLogout 配送を非同期に起動する。
- Precondition: id_token_hint_signature_valid_or_absent(input.params.id_token_hint)
- Precondition: id_token_hint_client_matches(input.params.id_token_hint, input.params.client_id)
- Precondition: registered_post_logout_redirect_uri(input.params.client_id, input.params.post_logout_redirect_uri)

### REQ-OAUTH2-049: FrontChannelLogout
対象 sid に ClientSession を持つ RP のうち frontchannel_logout_uri を登録した
ものについて、EndSession 応答へ埋め込む iframe target 一覧を算出する
(OpenID Connect Front-Channel Logout 1.0)。frontchannel_logout_session_required=true
の client には iss / sid クエリパラメータを付与する。到達失敗 (RP 側 iframe load
エラー等) は許容し、ローカル revoke の成否に影響しない。

### REQ-OAUTH2-050: BackChannelLogout
1 件の LogoutNotification を配送する job handler (OpenID Connect Back-Channel
Logout 1.0)。署名済み logout token (iss, sub, aud, iat, jti, events, sid) を
target_uri へ POST し、2xx を成功、それ以外・timeout・接続失敗を再試行対象とする。
Jobs context の kind=backchannel_logout_delivery job から呼ばれ、max_attempts 到達で
LogoutNotification は Failed (dead-letter) に確定する。ローカル revoke は本 interface の
成否に関わらず既に確定済みである。

### REQ-OAUTH2-051: CheckSessionIframe
OP session の変化をポーリングする RP 用 hidden iframe (OIDC Session Management
1.0、Draft のまま Final に達していない仕様のため adoption は optional)。
静的な HTML/JS を返し、RP からの postMessage に対し現在の browser cookie が有効な
LoginSession に解決できるかどうかだけを応答する。session_state の salted hash 比較
など仕様の詳細な相関アルゴリズムは実装しない。

### REQ-OAUTH2-052: PushAuthorizationRequest
Pushed Authorization Request (RFC 9126)。client_id は登録済み Client、または Client ID Metadata Document の URL を都度解決したものを受理する。
- Precondition: !context.rate_limited
- Precondition: resource_indicator_registered_and_active(context.tenant_id, input.request.parameters.resource, input.request.parameters.scope)

### REQ-OAUTH2-053: Token
Token endpoint (RFC 6749 §3.2 + 9068 + 9449)
- Precondition: !context.rate_limited
- Precondition: signing_key_provider_healthy(context.tenant_id)
- Precondition: token_request_references_are_tenant_local(context.tenant_id, input.request)
- Precondition: dpop_proof_is_fresh_and_unique(context.dpop_proof)
- Precondition: input.request.grant_type != AuthorizationCode || resource_indicator_matches_authorization_request(context.tenant_id, input.request)
- Precondition: input.request.grant_type != ClientCredentials || resource_indicator_registered_and_active(context.tenant_id, input.request.resource, input.request.scope)
- Precondition: input.request.grant_type != DeviceCode || resource_indicator_registered_and_active(context.tenant_id, input.request.resource, input.request.scope)
- Precondition: input.request.grant_type != TokenExchange || resource_indicator_registered_and_active(context.tenant_id, input.request.resource, input.request.scope)
- Precondition: input.request.grant_type != TokenExchange || input.request.subject_token_type != "urn:ietf:params:oauth:token-type:jwt" || workload_attestation_verified(context.tenant_id, input.request.subject_token)
- Precondition: !contains_account_scope(input.request.scope) || grant_has_user_subject(input.request)
- Postcondition: output.response.access_token == null || access_token_audience_non_empty(output.response.access_token)
- Postcondition: input.request.grant_type != AuthorizationCode || consent_covers_issued_scopes(input.request)
- Postcondition: input.request.grant_type != AuthorizationCode || pkce_verifier_matches_authorization_code(input.request)
- Postcondition: input.request.grant_type != RefreshToken || rotated_refresh_preserves_parent_family_and_absolute_expiry(input.request.refresh_token, output.response.refresh_token)
- Postcondition: input.request.grant_type != RefreshToken || rotated_refresh_preserves_resource_binding(input.request.refresh_token, output.response.access_token)
- Postcondition: !refresh_token_reuse_detected(input.request) || refresh_family_is_revoked(input.request.refresh_token)
- Postcondition: client_authentication_failures_are_uniform()
- Postcondition: !contains_account_scope(output.response.scope) || access_token_subject_is_user(output.response.access_token)
- Postcondition: !contains_account_scope(output.response.scope) || access_token_audience_is_realm_api(output.response.access_token, context.issuer)

### REQ-OAUTH2-054: Introspect
通常 OAuth token と管理発行 JWT access token の Token Introspection (RFC 7662)。Agent 主体の token は SharedSignals の revocation epoch (kill-switch・所有者オフボード・CAEP/SSF 経由の失効) と issued_at を比較し、epoch 以前に発行された token は fail-closed で active=false とする。
- Postcondition: !access_token_subject_is_agent(input.request.token) || access_token_issued_after_revocation_epoch(input.request.token) || output.response.active == false

### REQ-OAUTH2-055: Revoke
通常 OAuth token と管理発行 JWT access token の Token Revocation (RFC 7009)

### REQ-OAUTH2-056: UserInfo
UserInfo (OIDC Core §5.3)。additional_claims は scope 主導の ClaimsForScopes と
client 単位の ClaimMapping claim_policy をマージする。

### REQ-OAUTH2-057: PostUserInfo
UserInfo の POST binding (OIDC Core §5.3)

### REQ-OAUTH2-058: DeviceAuthorization
Device Authorization Request (RFC 8628 §3.1)
- Precondition: !context.rate_limited

### REQ-OAUTH2-059: GetOpenidConfiguration
OIDC Discovery

### REQ-OAUTH2-060: GetOauthAuthorizationServer
OAuth 2.0 Authorization Server Metadata (RFC 8414)

### REQ-OAUTH2-061: GetProtectedResourceMetadata
Protected Resource Metadata (RFC 9728)。resource 未指定時は realm の IdMagic API と
ApiTokenScope を、resource 指定時は登録済み Active な McpResourceServer の metadata を返す。
未登録・Disabled は invalid_target として拒否する。

### REQ-OAUTH2-062: Health
Health check

### REQ-OAUTH2-063: ListAuthorizationDetailTypes
管理者が所属テナントの authorization_details type を一覧する (RFC 9396)。

### REQ-OAUTH2-064: GetAuthorizationDetailType
管理者が単一 type を取得する。別テナントの type は未存在として扱う。

### REQ-OAUTH2-065: CreateAuthorizationDetailType
管理者が所属テナントに authorization_details type を登録する。

### REQ-OAUTH2-066: UpdateAuthorizationDetailType
管理者が type の schema / display_template / description / state を更新する。

### REQ-OAUTH2-067: DeleteAuthorizationDetailType
管理者が type を削除する。発行済みトークンは失効まで有効。

### REQ-OAUTH2-068: ListAdminOAuth2Clients
管理者が所属テナントの client を client_id 昇順の双方向 keyset pagination で一覧する。
cursor は応答の Link response header (rel="prev" / rel="next") から取得する。

### REQ-OAUTH2-069: GetAdminOAuth2Client
管理者が所属テナントの client を取得する。別テナントの client は未存在として扱う。

### REQ-OAUTH2-070: CreateAdminOAuth2Client
管理者が所属テナントに client を作成する。

### REQ-OAUTH2-071: UpdateAdminOAuth2Client
管理者が所属テナントの client metadata を更新する。

### REQ-OAUTH2-072: DeleteAdminOAuth2Client
管理者が所属テナントの catalog 外 client を削除する。application_id が非 NULL の client は ApplicationOwnedProtocolError (HTTP 409) で拒否し、Application 削除へ誘導する。

### REQ-OAUTH2-073: ListAdminMcpResourceServers
管理者が所属テナントの McpResourceServer を一覧する (AdminMcpResourcesManage)。

### REQ-OAUTH2-074: GetAdminMcpResourceServer
管理者が所属テナントの McpResourceServer を取得する。別テナントの登録は未存在として扱う。

### REQ-OAUTH2-075: CreateAdminMcpResourceServer
管理者が所属テナントに McpResourceServer を登録する。resource は tenant 内で一意でなければならない。

### REQ-OAUTH2-076: UpdateAdminMcpResourceServer
管理者が McpResourceServer の name / scopes / state を更新する。resource (canonical URI) は不変。

### REQ-OAUTH2-077: DeleteAdminMcpResourceServer
管理者が McpResourceServer を削除する。以後この resource を指定する新規発行は invalid_target で拒否する (発行済みトークンは失効まで有効)。

### REQ-OAUTH2-078: ListAdminConsents
管理者が所属テナントの Consent を監査目的で user_id, client_id 昇順の双方向 keyset
pagination で一覧する。cursor は応答の Link response header (rel="prev" / rel="next") から取得する。

### REQ-OAUTH2-079: GetAdminConsent
管理者が所属テナントの Consent を subject と client_id で取得する。別テナントの Consent は未存在として扱う。

### REQ-OAUTH2-080: RevokeAdminConsent
管理者が所属テナントの Consent を論理撤回する。管理者による Consent の作成・scope 拡張は許可しない。

### REQ-OAUTH2-081: ListAdminRolePolicies
管理者が SCL authorization から導出された、自身に可視な role・permission・HTTP interface の対応を取得する。

### REQ-OAUTH2-082: ListMyConsents
認証済みユーザーが自身の接続済みアプリ (active な Consent) 一覧を取得する
(self-service)。対象は actor.id に固定し、他人の Consent は返さない。

### REQ-OAUTH2-083: RevokeMyConsent
認証済みユーザーが自身の Consent を client_id 指定で論理撤回する (self-service)。
actor.id == target.id に固定。次回その client の認可では consent が再要求される。

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

## State machines

### ClientSecretCredentialLifecycle

client secret credential は発行時に Active となり、期限到達で Expired、管理者の個別失効で Revoked となる。Revoked は期限より優先して表示する。

Initial: `Active`  
Terminal: `Expired`, `Revoked`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Expire | now() >= expires_at | Expired |  |
| Active | ClientSecretRevoked | "" | Revoked |  |

### AuthorizationCodeFlow

/authorize から /token に至る authorization request のライフサイクル。

Initial: `Received`  
Terminal: `Exchanged`, `Rejected`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Received | Validate | "" | AuthenticationPending |  |
| Received | Reject | "" | Rejected |  |
| AuthenticationPending | AuthenticateUser | "" | Authenticated |  |
| AuthenticationPending | Reject | "" | Rejected |  |
| AuthenticationPending | Expire | "" | Expired |  |
| Authenticated | RequestConsent | "" | ConsentPending |  |
| Authenticated | IssueCode | "" | CodeIssued |  |
| Authenticated | Reject | "" | Rejected |  |
| ConsentPending | GrantConsent | "" | Consented |  |
| ConsentPending | Reject | "" | Rejected |  |
| ConsentPending | Expire | "" | Expired |  |
| Consented | IssueCode | "" | CodeIssued |  |
| Consented | Reject | "" | Rejected |  |
| CodeIssued | RedeemCode | "" | Exchanged |  |
| CodeIssued | Expire | "" | Expired |  |

### DeviceCodeFlow

RFC 8628 デバイス認可グラントのライフサイクル。device_code と user_code がペアで進む。

Initial: `Issued`  
Terminal: `Exchanged`, `Denied`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Issued | EnterUserCode | "" | UserCodeEntered |  |
| Issued | Expire | "" | Expired |  |
| UserCodeEntered | Approve | "" | Approved |  |
| UserCodeEntered | Deny | "" | Denied |  |
| UserCodeEntered | Expire | "" | Expired |  |
| Approved | Exchange | "" | Exchanged |  |
| Approved | Expire | "" | Expired |  |

### RefreshTokenLifecycle

RefreshToken のライフサイクル。Rotate で子トークンに引き継がれ、Revoke で失効、Expire で期限切れ。Rotated 後も家族失効により Revoked へ遷移しうる（RFC 9700 §4.14）。

Initial: `Active`  
Terminal: `Revoked`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Rotate | now() < absolute_expires_at | Rotated |  |
| Active | RevokeToken | "" | Revoked |  |
| Active | Expire | "" | Expired |  |
| Rotated | RevokeToken | "" | Revoked |  |
| Rotated | Expire | "" | Expired |  |

### LogoutNotificationLifecycle

LogoutNotification のライフサイクル。Deliver で成功確定、Exhaust で
max_attempts 到達による最終失敗確定 (dead-letter)。Jobs 側の Retry は Pending のまま
attempts のみ増やす (状態遷移ではない)。

Initial: `Pending`  
Terminal: `Delivered`, `Failed`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Pending | Deliver | "" | Delivered |  |
| Pending | Exhaust | "" | Failed |  |

### AuthorizationCodeRecordLifecycle

発行された AuthorizationCode 本体のライフサイクル。AuthorizationCodeFlow（AuthorizationRequest 側）の Exchanged に対応するのが Redeemed。

Initial: `Issued`  
Terminal: `Redeemed`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Issued | RedeemCode | now() < expires_at | Redeemed |  |
| Issued | Expire | "" | Expired |  |

### ConsentLifecycle

同意レコードのライフサイクル。GDPR Art.7(3) により Granted → Revoked が可能。

Initial: `Granted`  
Terminal: `Revoked`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Granted | RevokeConsent | "" | Revoked |  |
| Granted | Expire | "" | Expired |  |

### PARRecordLifecycle

PAR で発行された request_uri のライフサイクル。/authorize から一度だけ参照可能（RFC 9126）。

Initial: `Stored`  
Terminal: `Used`, `Expired`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Stored | Use | now() < expires_at | Used |  |
| Stored | Expire | "" | Expired |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
