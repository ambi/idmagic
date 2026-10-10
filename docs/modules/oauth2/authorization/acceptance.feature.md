# Feature: 認可の例

## Rule: REQ-OAUTH2-005 認可コードフローでアクセストークンと ID トークンを取得できる

### Background:

- Given "web-app" は confidential クライアントで redirect_uri "https://app.example.com/callback" を登録済みである
- And ユーザー "alice" は "web-app" に scope "openid プロファイル offline_access" を同意済みである

### Example: EX-OAUTH2-005-01 通常経路

- When "web-app" として scope "openid プロファイル offline_access" で認可リクエストを送る
- When クライアントが発行された認可コードを正しい PKCE verifier で交換する
- Then レスポンスに `access_token`、`id_token`、`refresh_token` が含まれ、`token_type` は `Bearer`
- Then "UserAuthenticated" が発行される
- Then "AuthorizationCodeIssued" が発行される
- Then "AuthorizationCodeRedeemed" が発行される
- Then "AccessTokenIssued" が発行される
- Then "RefreshTokenIssued" が発行される

### Scenario Outline: 条件ごとの結果

- When "web-app" として scope "openid プロファイル offline_access" で認可リクエストを送る
- But <condition>
- Then <result>
- And エラー "InvalidRequestError"

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-OAUTH2-005-02 | 認可リクエストの redirect_uri が未登録である | リダイレクトは行われず IdP がエラーページを表示する |
  | EX-OAUTH2-005-04 | `request_uri` と、併用を許可しないフロントチャネル認可パラメーターが混在する | 認可コードは発行されない |

### Example: EX-OAUTH2-005-03 単一値の認可パラメーターが重複する、`prompt` に重複または未対応のトークンがある、または `none` がほかの `prompt` トークンと併用される

- When "web-app" として scope "openid プロファイル offline_access" で認可リクエストを送る
- But 単一値の認可パラメーターが重複する、`prompt` に重複または未対応のトークンがある、または `none` がほかの `prompt` トークンと併用される
- Then 認可コードは発行されない
- And 安全に確定した登録済みの `redirect_uri` があれば `state` と発行者の識別子を含む `invalid_request` を返す
- And それ以外はリダイレクトせず IdP がエラーページを表示する

### Example: EX-OAUTH2-005-05 `prompt=none` で既存セッションまたは必要な同意がない

- When "web-app" として scope "openid プロファイル offline_access" で認可リクエストを送る
- But `prompt=none` で既存セッションまたは必要な同意がない
- Then UI とログインへのリダイレクトは発生しない
- And 既存セッションがなければ `state` と発行者識別子を含む `login_required` を登録済みの `redirect_uri` へ返す
- And 同意がなければ `state` と発行者識別子を含む `consent_required` を登録済みの `redirect_uri` へ返す

### Example: EX-OAUTH2-005-06 PKCE verifier が一致しない

- When "web-app" として scope "openid プロファイル offline_access" で認可リクエストを送る
- When クライアントが発行された認可コードを正しい PKCE verifier で交換する
- But PKCE verifier が一致しない
- Then 認可コードを誤った code_verifier で交換する
- And エラー "InvalidGrantError"
- And トークンは発行されない

### Example: EX-OAUTH2-005-07 同じ認可コードを 2 回交換する

- When "web-app" として scope "openid プロファイル offline_access" で認可リクエストを送る
- When クライアントが発行された認可コードを正しい PKCE verifier で交換する
- But 同じ認可コードを 2 回交換する
- Then 1 回目のレスポンスには access_token が含まれる
- And 2 回目はエラー "InvalidGrantError"
- And 発行ファミリーのトークンがすべて失効する
- And "RefreshTokenReuseDetected" が発行される
- And "TokenRevoked" が発行される

### Example: EX-OAUTH2-005-08 認可コードが発行から 60 秒を超えている

- When "web-app" として scope "openid プロファイル offline_access" で認可リクエストを送る
- When クライアントが発行された認可コードを正しい PKCE verifier で交換する
- But 認可コードが発行から 60 秒を超えている
- Then 認可コードの交換はエラー "InvalidGrantError"
- And 認可コードの状態は Expired になる

## Rule: REQ-OAUTH2-009 PAR で送信した認可リクエストを request_uri 経由で実行する

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

## Rule: REQ-OAUTH2-015 認可コードの並行交換はちょうど一方だけ成功する

### Example: EX-OAUTH2-015-01 通常経路

- Given 発行済み認可コード "AC1"（family_id "F1"）が存在する
- When 認可コード "AC1" を verifier "v" で並行に 2 回交換する
- Then ちょうど一方が成功し、もう一方はエラー "InvalidGrantError"
- Then family_id "F1" のトークンがすべて失効する
- Then "AuthorizationCodeRedeemed" が発行される
- Then "RefreshTokenReuseDetected" が発行される
- Then "TokenRevoked" が発行される

## Rule: REQ-OAUTH2-022 認可リクエストの nonce は ID トークンに伝播する

### Example: EX-OAUTH2-022-01 通常経路

- When "web-app" として scope "openid"、nonce "n-12345" で認可リクエストを送る
- When クライアントが発行された認可コードを verifier "v" で交換する
- Then レスポンスの id_token の nonce クレームは "n-12345"
