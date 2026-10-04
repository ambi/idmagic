# Feature: 外部 IdP との連携の例

## Rule: REQ-AUTHENTICATION-002 検証済みメールアドレスによる自動リンクは明示ポリシーと一意な一致を要求する

### Example: EX-AUTHENTICATION-002-01 通常経路

- Given その外部 subject に対する既存の関連付けはない
- And 同じメールアドレスを持つローカル User がテナント内に存在する
- And 接続の `linking_policy` が `VerifiedEmail` である
- And 上流の `email_verified` クレームが true で、メールアドレスがテナント内で一意に一致する
- When EndUser が未連携の外部 subject でフェデレーションログインを完了する
- Then 既存の User に対して FederatedIdentity を作成する

### Example: EX-AUTHENTICATION-002-02 ポリシーが `None`、メールアドレスが未検証、または一致が一意でない

- Given その外部 subject に対する既存の関連付けはない
- And 同じメールアドレスを持つローカル User がテナント内に存在する
- And 接続の `linking_policy` が `VerifiedEmail` である
- And 上流の `email_verified` クレームが true で、メールアドレスがテナント内で一意に一致する
- When EndUser が未連携の外部 subject でフェデレーションログインを完了する
- But ポリシーが `None`、メールアドレスが未検証、または一致が一意でない
- Then 自動リンクと LoginSession の発行を拒否する

## Rule: REQ-AUTHENTICATION-025 外部 IdP 接続の管理は対話セッションに限る

### Example: EX-AUTHENTICATION-025-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は `admin` ロールを持つ
- When クライアントが外部 IdP 接続の参照または変更をリクエストする
- Then 外部 IdP 接続の管理は、ブラウザーのログインセッションまたは管理ポータルのアクセストークンからのみ行える
- When 同じクライアントがセッションと認証情報の管理 API をリクエストする
- Then `sessions:read` は利用者のセッションとサインイン履歴の参照を、`sessions:write` はセッションの失効を、`users:write` は MFA 登録の一時免除と認証器のリセットを許可する

### Example: EX-AUTHENTICATION-025-02 トークンが `ApiTokenScope` のどのスコープを持っていても

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は `admin` ロールを持つ
- When クライアントが外部 IdP 接続の参照または変更をリクエストする
- But トークンが `ApiTokenScope` のどのスコープを持っていても
- Then 操作は `insufficient_scope` で拒否され、必要な資格として対話セッションを提示する

## Rule: REQ-AUTHENTICATION-001 外部 OIDC 認証は検証済みの subject を常に同じローカル User へ相関する

### Example: EX-AUTHENTICATION-001-01 通常経路

- Given リクエスト先のテナントで OIDC 接続が `Active` である
- And issuer、認可エンドポイント、トークンエンドポイント、JWKS は登録時に検証済みである
- When EndUser が StartFederatedLogin を開始する
- Then `state`、`nonce`、PKCE を単回限りのログイン試行として保存し、上流へ遷移する
- When 上流のコールバックが認可コードと ID Token を返す
- Then CompleteFederatedLogin は code と ID Token の署名、issuer、audience、時刻、nonce を検証する
- Then 初回は、明示した JIT ポリシーとクレームの対応付けに従ってローカルの User と FederatedIdentity を作成する
- Then 2 回目以降は、同じテナント・プロバイダー・外部 subject の既存の関連付けから同じローカル User を解決する
- Then AMR に `federated` を持つ LoginSession を発行する

### Example: EX-AUTHENTICATION-001-02 同じ `state` またはトークンレスポンスを再利用する

- Given リクエスト先のテナントで OIDC 接続が `Active` である
- And issuer、認可エンドポイント、トークンエンドポイント、JWKS は登録時に検証済みである
- When EndUser が StartFederatedLogin を開始する
- Then `state`、`nonce`、PKCE を単回限りのログイン試行として保存し、上流へ遷移する
- When 上流のコールバックが認可コードと ID Token を返す
- But 同じ `state` またはトークンレスポンスを再利用する
- Then 単回限りの試行と再送防止によって拒否する

### Example: EX-AUTHENTICATION-001-03 `state`、`nonce`、issuer、audience、署名、時刻のいずれかが一致しない

- Given リクエスト先のテナントで OIDC 接続が `Active` である
- And issuer、認可エンドポイント、トークンエンドポイント、JWKS は登録時に検証済みである
- When EndUser が StartFederatedLogin を開始する
- Then `state`、`nonce`、PKCE を単回限りのログイン試行として保存し、上流へ遷移する
- When 上流のコールバックが認可コードと ID Token を返す
- Then `state`、`nonce`、issuer、audience、署名、時刻のいずれかが一致しない
- Then コールバックを拒否し、LoginSession も関連付けも作成しない
- And FederatedLoginRejected を発行する

## Rule: REQ-AUTHENTICATION-003 外部アイデンティティの明示的なリンクと解除はステップアップ認証を要求する

### Example: EX-AUTHENTICATION-003-01 通常経路

- Given ResourceOwner は対象テナントの有効な User である
- When 直近 5 分以内にステップアップ認証を済ませたセッションで、外部プロバイダーの認証を完了する
- Then その外部 subject が未使用であれば、自身へリンクする
- When 直近 5 分以内にステップアップ認証を済ませたセッションでリンクの解除を要求する
- Then 対象の外部アイデンティティのリンクを解除する

### Example: EX-AUTHENTICATION-003-02 ステップアップ認証が古い、または行われていない

- Given ResourceOwner は対象テナントの有効な User である
- When 直近 5 分以内にステップアップ認証を済ませたセッションで、外部プロバイダーの認証を完了する
- But ステップアップ認証が古い、または行われていない
- Then リンクと解除を AccessDeniedError で拒否する

### Example: EX-AUTHENTICATION-003-03 パスワード資格情報も他の外部アイデンティティのリンクも残らなくなる

- Given ResourceOwner は対象テナントの有効な User である
- When 直近 5 分以内にステップアップ認証を済ませたセッションで、外部プロバイダーの認証を完了する
- Then その外部 subject が未使用であれば、自身へリンクする
- When 直近 5 分以内にステップアップ認証を済ませたセッションでリンクの解除を要求する
- But パスワード資格情報も他の外部アイデンティティのリンクも残らなくなる
- Then 締め出しを防ぐため UnlinkDeniedError で解除を拒否し、外部アイデンティティのリンクは残る

## Rule: REQ-AUTHENTICATION-037 外部 IdP 接続は、利用者との連携が残っている間は削除できない

### Example: EX-AUTHENTICATION-037-01 通常経路

- Given 管理者として認証済みである
- And テナントに外部 IdP 接続がある
- When 管理者がその接続を DeleteIdentityProviderConnection で削除する
- Then 接続は削除され、以後の一覧に現れない

### Example: EX-AUTHENTICATION-037-02 接続に連携した外部アイデンティティが残っている

- Given 管理者として認証済みである
- And テナントに外部 IdP 接続がある
- When 管理者がその接続を DeleteIdentityProviderConnection で削除する
- But 接続に連携した外部アイデンティティが残っている
- Then エラー "IdentityProviderConnectionInUseError"
- And 接続と外部アイデンティティの連携はどちらも残る
