# Feature: Authentication Scenarios

## Rule: REQ-AUTHENTICATION-001 外部 OIDC 認証は検証済みの subject を常に同じローカル User へ相関する

Primary actor: `EndUser`

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

## Rule: REQ-AUTHENTICATION-002 検証済みメールアドレスによる自動リンクは明示ポリシーと一意な一致を要求する

Primary actor: `EndUser`

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

## Rule: REQ-AUTHENTICATION-003 外部アイデンティティの明示的なリンクと解除はステップアップ認証を要求する

Primary actor: `AuthenticatedSelf`

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
- Then 締め出しを防ぐため解除を拒否する

## Rule: REQ-AUTHENTICATION-004 API トークンの発行者は機密操作のスコープで自身の認証情報だけを操作できる

Primary actor: `SelfApiClient`

### Example: EX-AUTHENTICATION-004-01 通常経路

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- Then `account:read` スコープは、自身のアカウント情報、セキュリティ設定、サインイン履歴、セッションの参照だけを許可する
- Then `account:mfa:write` スコープは、自身の MFA 認証要素と復旧コードの変更だけを許可する
- Then `account:sessions:write` スコープは、自身のセッションの失効だけを許可する
- Then `account:password:write` スコープと現在のパスワードの提示は、自身のパスワードの変更だけを許可する

### Example: EX-AUTHENTICATION-004-02 対応しないスコープで機密操作の変更を要求する

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- But 対応しないスコープで機密操作の変更を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-AUTHENTICATION-004-03 トークンのテナントまたは `user_id` が操作対象と一致しない

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- But トークンのテナントまたは `user_id` が操作対象と一致しない
- Then 操作は AccessDeniedError で拒否される

### Example: EX-AUTHENTICATION-004-04 API トークンでステップアップ認証のエンドポイントを要求する

- Given クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- When クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
- But API トークンでステップアップ認証のエンドポイントを要求する
- Then 操作は AccessDeniedError で拒否される

## Rule: REQ-AUTHENTICATION-005 ブラウザーの初期化情報は認証状態と CSRF 境界を保持する

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-005-01 通常経路

- Given ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- When ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
- Then 管理ポータルは `idmagic.admin`、アカウントポータルは `idmagic.account`、自己管理 API クライアントは `account:read` スコープで同じアカウントコンテキストを取得できる
- Then レスポンスは subject、realm、実効ロール、CSRF トークンを含む
- When 未認証のパスワードリセット画面がパスワードリセットコンテキストをリクエストする
- Then CSRF トークンを含むコンテキストが返る

### Example: EX-AUTHENTICATION-005-02 セッションが未認証または認証途中である

- Given ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- When ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
- But セッションが未認証または認証途中である
- Then アカウントコンテキストの取得を AccessDeniedError で拒否する

### Example: EX-AUTHENTICATION-005-03 Bearer トークンが許可されたポータルスコープまたは `account:read` スコープを 1 つも持たない

- Given ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- When ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
- But Bearer トークンが許可されたポータルスコープまたは `account:read` スコープを 1 つも持たない
- Then アカウントコンテキストの取得を AccessDeniedError で拒否する

## Rule: REQ-AUTHENTICATION-006 ユーザーは WebAuthn でステップアップ認証のチャレンジを開始できる

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-006-01 通常経路

- Given ユーザー "alice" が WebAuthn のクレデンシャルを登録済みで、認証済みセッションを持つ
- When ユーザー "alice" が正しい CSRF トークンでステップアップ認証の WebAuthn チャレンジを要求する
- Then レスポンスの `PublicKeyCredentialRequestOptions` は現在のセッションに束縛される

### Example: EX-AUTHENTICATION-006-02 CSRF トークンが一致しない、または WebAuthn を利用できない

- Given ユーザー "alice" が WebAuthn のクレデンシャルを登録済みで、認証済みセッションを持つ
- When ユーザー "alice" が正しい CSRF トークンでステップアップ認証の WebAuthn チャレンジを要求する
- But CSRF トークンが一致しない、または WebAuthn を利用できない
- Then チャレンジは発行されず、要求を拒否する

## Rule: REQ-AUTHENTICATION-007 ResourceOwner はブラウザーでパスワード認証し、認可を継続する

Primary actor: `ResourceOwner`

### Example: EX-AUTHENTICATION-007-01 通常経路

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- Then セッション Cookie が発行される
- Then 認可コードが redirect_uri に返る
- Then "UserAuthenticated" が発行される

### Example: EX-AUTHENTICATION-007-02 SameSite の Cookie とリクエストのトークンが一致しない

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- But SameSite の Cookie とリクエストのトークンが一致しない
- Then CSRF の値を改ざんしてログイン API を送信する
- And エラー "InvalidRequestError"

### Example: EX-AUTHENTICATION-007-03 直近 900 秒の時間枠で、アカウント単位の失敗回数が 10 回に達している

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- But 直近 900 秒の時間枠で、アカウント単位の失敗回数が 10 回に達している
- Then 正しいパスワードでログイン API を送信する
- And エラー "RateLimitedError"
- And "LoginThrottled" が発行される

### Example: EX-AUTHENTICATION-007-04 失敗回数によらず、同一 IP からのログイン API リクエストが `EndpointRateLimitPolicy` の時間枠内で上限に達している

- Given 未認証セッションで "web-app" として認可リクエストを送信済みである
- When ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
- But 失敗回数によらず、同一 IP からのログイン API リクエストが `EndpointRateLimitPolicy` の時間枠内で上限に達している
- Then 正しいパスワードでログイン API を送信する
- And エラー "RateLimitedError"

## Rule: REQ-AUTHENTICATION-008 パスワードリセットの要求は識別子と IP の組で流量制限される

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-008-01 通常経路

- Given 未認証である
- When "alice" 宛のパスワードリセットを要求する
- Then ユーザーの存在にかかわらず 204 を返す
- Then "PasswordResetRequested" が発行される

### Example: EX-AUTHENTICATION-008-02 同じ識別子と IP の組で、`EndpointRateLimitPolicy` の時間枠内の上限に達している

- Given 未認証である
- When "alice" 宛のパスワードリセットを要求する
- But 同じ識別子と IP の組で、`EndpointRateLimitPolicy` の時間枠内の上限に達している
- Then "alice" 宛のパスワードリセットを再度要求する
- And エラー "RateLimitedError"

## Rule: REQ-AUTHENTICATION-009 無効なユーザーは新規ログインも既存セッションも拒否される

無効化そのものは IdManagement の操作であり、無効化から到達経路が閉じるまでの連鎖は REQ-PLATFORM-001 が持つ。ここは、無効な主体を Authentication が単独で拒否することだけを述べる。

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-009-01 通常経路

- Given ユーザー "alice" は無効状態であり、無効化の前に取得した認証済みセッションを持つ
- When ユーザー "alice" が既存セッションで認証必須 API を呼ぶ
- Then エラー "AccessDeniedError"
- When ユーザー "alice" が正しいパスワードで新規ログインを試みる
- Then エラー "AccessDeniedError"

## Rule: REQ-AUTHENTICATION-010 ユーザーは現在のパスワードを確認して新しいパスワードへ変更できる

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-010-01 通常経路

- Given ユーザー "alice" が認証済みでパスワード変更画面を開いている
- When ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
- Then パスワードが変更され、`password_changed_at` が更新される
- Then "PasswordChanged" が発行される

### Example: EX-AUTHENTICATION-010-02 新しいパスワードが 12 文字未満である

- Given ユーザー "alice" が認証済みでパスワード変更画面を開いている
- When ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
- But 新しいパスワードが 12 文字未満である
- Then ユーザー "alice" が 12 文字未満のパスワードを送信する
- And エラー "InvalidRequestError"

### Example: EX-AUTHENTICATION-010-03 新しいパスワードが直近 5 件の履歴に一致する

- Given ユーザー "alice" が認証済みでパスワード変更画面を開いている
- When ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
- But 新しいパスワードが直近 5 件の履歴に一致する
- Then ユーザー "alice" が直近使用した過去のパスワードを新パスワードとして送信する
- And エラー "InvalidRequestError"

## Rule: REQ-AUTHENTICATION-011 ユーザーは TOTP 認証要素を登録して有効化できる

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-011-01 通常経路

- Given ユーザー "alice" が認証済みでセキュリティ画面を開いている
- When ユーザー "alice" が TOTP 登録を開始する
- Then レスポンスにシークレットとアカウント名が含まれる
- When ユーザー "alice" がそのシークレットに対する正しいコードで登録を確認する
- Then セキュリティ概要の MFA 状態が登録済みになる
- Then "MfaFactorEnrolled" が発行される

## Rule: REQ-AUTHENTICATION-012 ユーザーはステップアップ再認証のうえで TOTP 認証要素を解除する

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-012-01 通常経路

- Given ユーザー "alice" が登録済みの TOTP 認証要素を持ち認証済みである
- When ユーザー "alice" がステップアップ認証を成立させ現在の TOTP コードで解除する
- Then TOTP 認証要素が解除される
- Then "MfaFactorRemoved" が発行される

### Example: EX-AUTHENTICATION-012-02 ステップアップ認証なしで解除を試みる

- Given ユーザー "alice" が登録済みの TOTP 認証要素を持ち認証済みである
- When ユーザー "alice" がステップアップ認証を成立させ現在の TOTP コードで解除する
- But ステップアップ認証なしで解除を試みる
- Then ユーザー "alice" がステップアップ認証なしで TOTP 認証要素の解除を試みる
- And ステップアップ認証による再認証が要求される

## Rule: REQ-AUTHENTICATION-013 ユーザーは自分の有効なセッションを一覧して失効できる

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-013-01 通常経路

- Given ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- When ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
- Then 自分の有効なセッションが返る
- When ユーザー "alice" が現在以外のセッションを 1 件失効させる
- Then 失効したセッションは一覧から消える
- When ユーザー "alice" が現在以外のすべてのセッションを一括失効させる
- Then 現在のセッションだけが残る

### Example: EX-AUTHENTICATION-013-02 プロセスの再起動を挟んでセッション一覧を取得する

- Given ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- When ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
- But プロセスの再起動を挟んでセッション一覧を取得する
- Then サーバープロセスを再起動する
- And ユーザー "alice" が同じセッション Cookie でアクティビティ画面を開く
- And セッションは再起動前と同じ内容で解決できる

### Example: EX-AUTHENTICATION-013-03 既に失効済みのセッションへ同じ失効操作を再送する

- Given ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- When ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
- Then 自分の有効なセッションが返る
- When ユーザー "alice" が現在以外のセッションを 1 件失効させる
- But 既に失効済みのセッションへ同じ失効操作を再送する
- Then ユーザー "alice" が直前に失効させた同じセッション ID へ再度失効を要求する
- And 要求は成功として扱われ、最初の失効時刻を保持する

## Rule: REQ-AUTHENTICATION-014 ユーザーは自分のサインイン履歴を確認できる

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-014-01 通常経路

- Given ユーザー "alice" が認証済みでアクティビティ画面を開いている
- When ユーザー "alice" が自分のサインイン履歴を取得する
- Then レスポンスに自分のサインインイベントだけが含まれる
- Then 第二要素を使ったサインインは、`pwd` と第二要素の `amr` を持つ完了後の `UserAuthenticated` として表示される

### Example: EX-AUTHENTICATION-014-02 認証手段に WebAuthn が含まれる

- Given ユーザー "alice" が認証済みでアクティビティ画面を開いている
- When ユーザー "alice" が自分のサインイン履歴を取得する
- But 認証手段に WebAuthn が含まれる
- Then UI は `webauthn` という技術名ではなく「パスキー」と表示する

## Rule: REQ-AUTHENTICATION-015 MFA 登録済みでも、ポリシーが要求しない限り第二要素は求めない

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-015-01 通常経路

- Given ユーザー "alice" は TOTP または WebAuthn のクレデンシャルを登録済みである
- And 対象 Application の実効サインインポリシーは `Password` である
- When ユーザー "alice" がユーザー名とパスワードを送信する
- Then LoginSession は `authentication_pending=false` で作られる
- Then 認可フローは第二要素画面に進まず、同意または認可コード発行へ進む

### Example: EX-AUTHENTICATION-015-02 対象 Application の実効サインインポリシーが `Mfa` である

- Given ユーザー "alice" は TOTP または WebAuthn のクレデンシャルを登録済みである
- And 対象 Application の実効サインインポリシーは `Password` である
- When ユーザー "alice" がユーザー名とパスワードを送信する
- But 対象 Application の実効サインインポリシーが `Mfa` である
- Then LoginSession は `authentication_pending=true` へ切り替わる
- And 利用できる第二要素 (TOTP / パスキー / 復旧コード) の選択画面へ進む

## Rule: REQ-AUTHENTICATION-016 ユーザーはメールのリセットリンクでパスワードを再設定する

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-016-01 通常経路

- Given ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- When ユーザー "alice" がそのトークンと新しいパスワードを送信する
- Then パスワードが更新される
- When ユーザー "alice" が新しいパスワードをブラウザーのログイン API へ送信する
- Then ログインに成功する
- When EndUser が未登録のメールアドレスでパスワードリセットを要求する
- Then レスポンスは登録済みアドレスに対するものと区別できない
- When EndUser が登録済みのメールアドレスでパスワードリセットを要求する
- Then 登録済みアドレスへリセットリンクが送られる

### Example: EX-AUTHENTICATION-016-02 トークンが期限切れまたは不正である

- Given ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- When ユーザー "alice" がそのトークンと新しいパスワードを送信する
- But トークンが期限切れまたは不正である
- Then 無効なパスワードリセットトークンで新しいパスワードを送信する
- And エラー "InvalidRequestError"

## Rule: REQ-AUTHENTICATION-017 TOTP が必須のユーザーは正しいコードで認証を継続できる

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-017-01 通常経路

- Given TOTP 認証要素が登録された `authentication_pending` の LoginSession が存在する
- When ブラウザーの TOTP API に正しいコードを送信する
- Then 認証が成立し認可フローが継続する
- Then "UserAuthenticated" が発行される

### Example: EX-AUTHENTICATION-017-02 誤った TOTP コードを送信する

- Given TOTP 認証要素が登録された `authentication_pending` の LoginSession が存在する
- When ブラウザーの TOTP API に正しいコードを送信する
- But 誤った TOTP コードを送信する
- Then ブラウザーの TOTP API に誤ったコードを送信する
- And エラー "InvalidRequestError"
- And LoginSession は `authentication_pending` のままである

## Rule: REQ-AUTHENTICATION-018 MFA 未登録のユーザーは管理者が承認した登録を終えて同じ認可処理を継続できる

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-018-01 通常経路

- Given 対象 Application の実効ポリシーは MFA 必須かつ強制開始済みで、登録バイパスを許可し猶予期限内である
- And ユーザーは TOTP と WebAuthn のいずれの認証要素も持たない
- And 管理者が対象ユーザーへ有効な単回限りの登録バイパスを発行済みである
- When ユーザーが正しいパスワードを送信する
- Then バイパスを消費し、同じ LoginSession は `pending_purpose=Enrollment` の未完了状態になる
- Then `MfaEnrollmentRequired` と `MfaEnrollmentBypassConsumed` が発行され、登録専用画面へ進む
- When ユーザーが TOTP のシークレットに対する正しいコードで登録を確定する
- Then 認証要素が保存され、同じ LoginSession の `amr` に `otp` が追加されて保留状態が解除される
- Then `MfaEnrollmentCompleted` と `UserAuthenticated` が発行され、元の認可トランザクションが継続する

### Example: EX-AUTHENTICATION-018-02 登録バイパスがない、取り消し済み、消費済み、または期限切れである

- Given 対象 Application の実効ポリシーは MFA 必須かつ強制開始済みで、登録バイパスを許可し猶予期限内である
- And ユーザーは TOTP と WebAuthn のいずれの認証要素も持たない
- And 管理者が対象ユーザーへ有効な単回限りの登録バイパスを発行済みである
- When ユーザーが正しいパスワードを送信する
- But 登録バイパスがない、取り消し済み、消費済み、または期限切れである
- Then パスワードが正しくてもログインを完了せずアクセスを拒否する
- And 認証要素の登録 API は MfaEnrollmentNotAllowedError で拒否し、認証要素を作らない

### Example: EX-AUTHENTICATION-018-03 登録期限を過ぎている

- Given 対象 Application の実効ポリシーは MFA 必須かつ強制開始済みで、登録バイパスを許可し猶予期限内である
- And ユーザーは TOTP と WebAuthn のいずれの認証要素も持たない
- And 管理者が対象ユーザーへ有効な単回限りの登録バイパスを発行済みである
- When ユーザーが正しいパスワードを送信する
- Then バイパスを消費し、同じ LoginSession は `pending_purpose=Enrollment` の未完了状態になる
- Then `MfaEnrollmentRequired` と `MfaEnrollmentBypassConsumed` が発行され、登録専用画面へ進む
- When ユーザーが TOTP のシークレットに対する正しいコードで登録を確定する
- But 登録期限を過ぎている
- Then 認証要素を保存せずアクセスを拒否する
- And LoginSession を認証完了へ昇格させない

### Example: EX-AUTHENTICATION-018-04 TOTP コードが不正である

- Given 対象 Application の実効ポリシーは MFA 必須かつ強制開始済みで、登録バイパスを許可し猶予期限内である
- And ユーザーは TOTP と WebAuthn のいずれの認証要素も持たない
- And 管理者が対象ユーザーへ有効な単回限りの登録バイパスを発行済みである
- When ユーザーが正しいパスワードを送信する
- Then バイパスを消費し、同じ LoginSession は `pending_purpose=Enrollment` の未完了状態になる
- Then `MfaEnrollmentRequired` と `MfaEnrollmentBypassConsumed` が発行され、登録専用画面へ進む
- When ユーザーが TOTP のシークレットに対する正しいコードで登録を確定する
- But TOTP コードが不正である
- Then 認証要素を保存せず InvalidRequestError を返す
- And LoginSession は `Enrollment` の保留状態のままである

## Rule: REQ-AUTHENTICATION-019 MFA の強制開始前は、未登録のユーザーもログインできるが登録を促される

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-019-01 通常経路

- Given テナントデフォルトポリシーは将来時刻から MFA 必須になる
- And ユーザーは MFA 認証要素を持たない
- When ユーザーが正しいパスワードでログインする
- Then 強制開始前なので、パスワードだけのセッションが成立する
- Then UI は強制開始日時と事前登録を促す警告を表示する
- Then ユーザーは通常のステップアップ認証を経たアカウントのセキュリティ設定画面から認証要素を事前登録できる

## Rule: REQ-AUTHENTICATION-020 登録待ちのセッションは通常のリソースへアクセスできない

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-020-01 通常経路

- Given `pending_purpose=Enrollment` の LoginSession が存在する
- When ユーザーがアカウント、管理、Application のいずれかのリソースを要求する
- Then 未認証として拒否する
- Then 登録の開始と確定の API、および元の認可トランザクションだけを許可する

## Rule: REQ-AUTHENTICATION-021 管理者は対象ユーザーのセッションを一覧・個別失効・全失効できる

Primary actor: `TenantAdministrator`

### Example: EX-AUTHENTICATION-021-01 通常経路

- Given ユーザー "alice" が複数の有効な LoginSession を持つ
- When 管理者がユーザー "alice" の ListSessions を呼ぶ
- Then 開始時刻の降順で有効なセッション一覧が返る
- When 管理者がそのうち 1 件の `RevokeSession` を呼ぶ
- Then 対象セッションは `revoke_reason=admin_revoke` で失効し、"SessionEnded" が発行される
- When 管理者がユーザー "alice" の RevokeUserSessions を呼ぶ
- Then 残り全セッションが失効する

### Example: EX-AUTHENTICATION-021-02 他テナントの管理者が呼び出す

- Given ユーザー "alice" が複数の有効な LoginSession を持つ
- When 管理者がユーザー "alice" の ListSessions を呼ぶ
- But 他テナントの管理者が呼び出す
- Then エラー "AccessDeniedError"

### Example: EX-AUTHENTICATION-021-03 既に失効済みのセッションへ再度 `RevokeSession` を呼ぶ

- Given ユーザー "alice" が複数の有効な LoginSession を持つ
- When 管理者がユーザー "alice" の ListSessions を呼ぶ
- Then 開始時刻の降順で有効なセッション一覧が返る
- When 管理者がそのうち 1 件の `RevokeSession` を呼ぶ
- But 既に失効済みのセッションへ再度 `RevokeSession` を呼ぶ
- Then 204 が返り、`revoked_at` は初回の値を保持する

## Rule: REQ-AUTHENTICATION-022 管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる

Primary actor: `TenantAdministrator`

### Example: EX-AUTHENTICATION-022-01 通常経路

- Given ユーザー "alice" は TOTP 認証要素を持ち、復旧コードも生成済みである
- When 管理者がユーザー "alice" の ResetUserAuthenticators を targets=[Totp, RecoveryCode] で呼ぶ
- Then "AuthenticatorResetRequested" が発行される
- Then TOTP 認証要素と復旧コードが削除され、他に WebAuthn クレデンシャルもないため `mfa_enrolled` が `false` になる
- Then `reenrollment_required=true` のレスポンスが返り、単回限りの登録バイパスを自動発行する
- Then "AuthenticatorResetCompleted" と "MfaEnrollmentBypassIssued" が発行される
- When "alice" が正しいパスワードで次にログインする
- Then 有効なバイパスにより、同じ LoginSession が `pending_purpose=Enrollment` になる
- When "alice" が新しい TOTP 認証要素の登録を確定する
- Then 同じ LoginSession が MFA 済みへ昇格し、元の認可トランザクションが継続する

### Example: EX-AUTHENTICATION-022-02 他テナントの管理者、または `admin` ロールを持たない操作者が呼び出す

- Given ユーザー "alice" は TOTP 認証要素を持ち、復旧コードも生成済みである
- When 管理者がユーザー "alice" の ResetUserAuthenticators を targets=[Totp, RecoveryCode] で呼ぶ
- But 他テナントの管理者、または `admin` ロールを持たない操作者が呼び出す
- Then エラー "AccessDeniedError"
- And 対象ユーザーの認証器は変更されない

## Rule: REQ-AUTHENTICATION-023 管理者が一部の認証器のみリセットした場合は残存要素でログインを継続できる

Primary actor: `TenantAdministrator`

### Example: EX-AUTHENTICATION-023-01 通常経路

- Given ユーザー "bob" は TOTP 認証要素と WebAuthn クレデンシャルを両方持つ
- When 管理者がユーザー "bob" の ResetUserAuthenticators を targets=[Webauthn] で呼ぶ
- Then WebAuthn クレデンシャルだけが削除され、TOTP 認証要素は残るため `mfa_enrolled` は `true` のままである
- Then `reenrollment_required=false` のレスポンスが返り、登録バイパスは発行されない
- When "bob" が次回のログインで TOTP コードによる第二要素の検証を完了する
- Then ログインを完了できる

## Rule: REQ-AUTHENTICATION-024 有効期限を過ぎたパスワードのユーザーは次回ログイン後にパスワード変更を強制される

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-024-01 通常経路

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- Then ログイン自体は成功する
- Then ユーザー "alice" に必須操作 `update_password` が付与される
- Then ユーザー "alice" はパスワード変更画面へ誘導され、変更完了までフローを継続できない
- When ユーザー "alice" がポリシーを満たす新しいパスワードへ変更する
- Then `update_password` が解除され、"PasswordChanged" が発行される

### Example: EX-AUTHENTICATION-024-02 `password_changed_at` が 89 日前である

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- But `password_changed_at` が 89 日前である
- Then ログインはそのまま完了し、`update_password` は付与されない

### Example: EX-AUTHENTICATION-024-03 `max_age_days` が未設定である

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- But `max_age_days` が未設定である
- Then 経過日数によらず `update_password` は付与されない

### Example: EX-AUTHENTICATION-024-04 ポリシーの更新から 90 日が経過していない

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- But ポリシーの更新から 90 日が経過していない
- Then 猶予期間内なので `update_password` は付与されない

### Example: EX-AUTHENTICATION-024-05 ユーザーがパスワード資格情報を持たない (フェデレーションまたはパスワードレス)

- Given テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- And ユーザー "alice" の `password_changed_at` は 91 日前である
- When ユーザー "alice" が正しいパスワードでログインする
- But ユーザーがパスワード資格情報を持たない (フェデレーションまたはパスワードレス)
- Then `update_password` は付与されない

## Rule: REQ-AUTHENTICATION-025 外部 IdP 接続の管理は対話セッションに限る

Primary actor: `ManagementApiClient`

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

## Rule: REQ-AUTHENTICATION-026 第二要素の成立時に本人が同意した端末は次回以降の第二要素を省略できる

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-026-01 通常経路

- Given テナントの `trusted_device_max_age_seconds` は正の値である
- And 対象 Application の実効サインインポリシーは `Mfa` で `allow_trusted_device=true` である
- And ユーザー "alice" は TOTP 認証要素を登録済みである
- When ユーザー "alice" が正しいパスワードに続けて正しい TOTP コードを送信し、このデバイスを記憶することに同意する
- Then 認証が成立し、realm scope の HttpOnly cookie として信頼済みデバイスの資格情報が発行される
- Then "TrustedDeviceRegistered" が発行される
- When 同じブラウザーでユーザー "alice" が正しいパスワードを送信する
- Then 第二要素の画面へ進まずに認証が成立し、`amr` に `tdev` が加わって `acr` が `urn:idmagic:acr:mfa` になる
- Then 信頼済みデバイスの verifier が回転し、更新された cookie が再発行される

### Example: EX-AUTHENTICATION-026-02 テナントの `trusted_device_max_age_seconds` が 0 または未設定である

- Given テナントの `trusted_device_max_age_seconds` は正の値である
- And 対象 Application の実効サインインポリシーは `Mfa` で `allow_trusted_device=true` である
- And ユーザー "alice" は TOTP 認証要素を登録済みである
- When ユーザー "alice" が正しいパスワードに続けて正しい TOTP コードを送信し、このデバイスを記憶することに同意する
- But テナントの `trusted_device_max_age_seconds` が 0 または未設定である
- Then 同意は無視され、デバイスは記憶されない

### Example: EX-AUTHENTICATION-026-03 第二要素として復旧コードを消費した

- Given テナントの `trusted_device_max_age_seconds` は正の値である
- And 対象 Application の実効サインインポリシーは `Mfa` で `allow_trusted_device=true` である
- And ユーザー "alice" は TOTP 認証要素を登録済みである
- When ユーザー "alice" が正しいパスワードに続けて正しい TOTP コードを送信し、このデバイスを記憶することに同意する
- But 第二要素として復旧コードを消費した
- Then デバイスは記憶されない

### Example: EX-AUTHENTICATION-026-04 パスワードだけで認証が完了した (ポリシーが MFA を要求していない)

- Given テナントの `trusted_device_max_age_seconds` は正の値である
- And 対象 Application の実効サインインポリシーは `Mfa` で `allow_trusted_device=true` である
- And ユーザー "alice" は TOTP 認証要素を登録済みである
- When ユーザー "alice" が正しいパスワードに続けて正しい TOTP コードを送信し、このデバイスを記憶することに同意する
- But パスワードだけで認証が完了した (ポリシーが MFA を要求していない)
- Then デバイスは記憶されない

## Rule: REQ-AUTHENTICATION-027 期限切れ・盗難・別テナントの信頼済みデバイス cookie は第二要素を省略できない

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-027-01 通常経路

- Given ユーザー "alice" は 1 つの信頼済みデバイスを持ち、対象 Application の実効サインインポリシーは `Mfa` である
- When ユーザー "alice" が絶対期限を過ぎた cookie を提示して正しいパスワードを送信する
- Then LoginSession は `authentication_pending=true` になり、第二要素の選択画面へ進む
- Then `amr` に `tdev` は加わらない

### Example: EX-AUTHENTICATION-027-02 直近利用から idle 期限を過ぎた cookie を提示する

- Given ユーザー "alice" は 1 つの信頼済みデバイスを持ち、対象 Application の実効サインインポリシーは `Mfa` である
- When ユーザー "alice" が絶対期限を過ぎた cookie を提示して正しいパスワードを送信する
- But 直近利用から idle 期限を過ぎた cookie を提示する
- Then 第二要素を要求する

### Example: EX-AUTHENTICATION-027-03 回転前の古い cookie を提示する

- Given ユーザー "alice" は 1 つの信頼済みデバイスを持ち、対象 Application の実効サインインポリシーは `Mfa` である
- When ユーザー "alice" が絶対期限を過ぎた cookie を提示して正しいパスワードを送信する
- But 回転前の古い cookie を提示する
- Then 第二要素を要求する

### Example: EX-AUTHENTICATION-027-04 別テナントの realm で発行された cookie を提示する

- Given ユーザー "alice" は 1 つの信頼済みデバイスを持ち、対象 Application の実効サインインポリシーは `Mfa` である
- When ユーザー "alice" が絶対期限を過ぎた cookie を提示して正しいパスワードを送信する
- But 別テナントの realm で発行された cookie を提示する
- Then 第二要素を要求する

### Example: EX-AUTHENTICATION-027-05 selector は正しいが verifier が一致しない cookie を提示する

- Given ユーザー "alice" は 1 つの信頼済みデバイスを持ち、対象 Application の実効サインインポリシーは `Mfa` である
- When ユーザー "alice" が絶対期限を過ぎた cookie を提示して正しいパスワードを送信する
- But selector は正しいが verifier が一致しない cookie を提示する
- Then 第二要素を要求する

## Rule: REQ-AUTHENTICATION-028 資格情報が変わると信頼済みデバイスはすべて失効する

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-028-01 通常経路

- Given ユーザー "alice" は有効な信頼済みデバイスを持つ
- When ユーザー "alice" が自身のパスワードを変更する
- Then ユーザー "alice" の信頼済みデバイスはすべて失効し、"TrustedDeviceRevoked" が発行される
- When 失効した端末でユーザー "alice" が再びログインする
- Then 第二要素が再び要求される

### Example: EX-AUTHENTICATION-028-02 メールのリセットリンクでパスワードを再設定する

- Given ユーザー "alice" は有効な信頼済みデバイスを持つ
- When ユーザー "alice" が自身のパスワードを変更する
- But メールのリセットリンクでパスワードを再設定する
- Then 同じく全デバイスが失効する

### Example: EX-AUTHENTICATION-028-03 ユーザー "alice" が TOTP 認証要素を登録または解除する

- Given ユーザー "alice" は有効な信頼済みデバイスを持つ
- When ユーザー "alice" が自身のパスワードを変更する
- But ユーザー "alice" が TOTP 認証要素を登録または解除する
- Then 同じく全デバイスが失効する

### Example: EX-AUTHENTICATION-028-04 管理者がユーザー "alice" の認証器をリセットする

- Given ユーザー "alice" は有効な信頼済みデバイスを持つ
- When ユーザー "alice" が自身のパスワードを変更する
- But 管理者がユーザー "alice" の認証器をリセットする
- Then 同じく全デバイスが失効する

### Example: EX-AUTHENTICATION-028-05 管理者がユーザー "alice" を無効化する

- Given ユーザー "alice" は有効な信頼済みデバイスを持つ
- When ユーザー "alice" が自身のパスワードを変更する
- But 管理者がユーザー "alice" を無効化する
- Then 同じく全デバイスが失効する

### Example: EX-AUTHENTICATION-028-06 ユーザー "alice" が他のセッションを一括失効させる

- Given ユーザー "alice" は有効な信頼済みデバイスを持つ
- When ユーザー "alice" が自身のパスワードを変更する
- But ユーザー "alice" が他のセッションを一括失効させる
- Then 同じく全デバイスが失効する

## Rule: REQ-AUTHENTICATION-029 信頼済みデバイスは機微操作の再認証を肩代わりしない

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-029-01 通常経路

- Given ユーザー "alice" は信頼済みデバイスによって `amr` に `tdev` を持つセッションで認証済みである
- And そのセッションはステップアップ認証を行っていない
- When ユーザー "alice" がパスワードの変更、TOTP 認証要素の解除、または他セッションの一括失効を要求する
- Then ステップアップ認証による再認証が要求される
- When ユーザー "alice" が自身の信頼済みデバイスを一覧する
- Then selector と verifier を含まない一覧が最終利用時刻の降順で返り、現在の端末が current として示される
- When ユーザー "alice" がステップアップ認証を成立させて信頼済みデバイスを失効させる
- Then 対象は一覧から消え、"TrustedDeviceRevoked" が発行される

### Example: EX-AUTHENTICATION-029-02 ステップアップ認証なしで信頼済みデバイスの失効を要求する

- Given ユーザー "alice" は信頼済みデバイスによって `amr` に `tdev` を持つセッションで認証済みである
- And そのセッションはステップアップ認証を行っていない
- When ユーザー "alice" がパスワードの変更、TOTP 認証要素の解除、または他セッションの一括失効を要求する
- Then ステップアップ認証による再認証が要求される
- When ユーザー "alice" が自身の信頼済みデバイスを一覧する
- But ステップアップ認証なしで信頼済みデバイスの失効を要求する
- Then ステップアップ認証による再認証が要求される

### Example: EX-AUTHENTICATION-029-03 既に失効済みのデバイスへ同じ失効操作を再送する

- Given ユーザー "alice" は信頼済みデバイスによって `amr` に `tdev` を持つセッションで認証済みである
- And そのセッションはステップアップ認証を行っていない
- When ユーザー "alice" がパスワードの変更、TOTP 認証要素の解除、または他セッションの一括失効を要求する
- Then ステップアップ認証による再認証が要求される
- When ユーザー "alice" が自身の信頼済みデバイスを一覧する
- Then selector と verifier を含まない一覧が最終利用時刻の降順で返り、現在の端末が current として示される
- When ユーザー "alice" がステップアップ認証を成立させて信頼済みデバイスを失効させる
- But 既に失効済みのデバイスへ同じ失効操作を再送する
- Then 要求は成功として扱われ、最初の失効時刻を保持する

## Rule: REQ-AUTHENTICATION-030 既知でない端末からのサインインだけがセキュリティ通知を生む

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-030-01 通常経路

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- And ユーザー "alice" はこれまで一度もサインインしていないブラウザーを使っている
- When ユーザー "alice" がそのブラウザーで認証に成功する
- Then そのブラウザーは既知の端末として記録される
- Then "alice" の検証済みアドレスへセキュリティ通知が送られ、"AccountSecurityNotificationSent" が発行される
- When ユーザー "alice" が同じブラウザーで再び認証に成功する
- Then 通知は送られず、その端末の最終利用時刻だけが更新される

### Example: EX-AUTHENTICATION-030-02 "alice" が検証済みのメールアドレスを持たない

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- And ユーザー "alice" はこれまで一度もサインインしていないブラウザーを使っている
- When ユーザー "alice" がそのブラウザーで認証に成功する
- Then そのブラウザーは既知の端末として記録される
- Then "alice" が検証済みのメールアドレスを持たない
- Then 通知は送られず、認証は成功したままである

## Rule: REQ-AUTHENTICATION-031 資格情報の変更は本人へ通知され、通知の失敗は変更を巻き戻さない

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-031-01 通常経路

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- When ユーザー "alice" のパスワード、認証要素、復旧コード、または信頼済みデバイスが増減する
- Then "alice" の検証済みアドレスへセキュリティ通知が送られる
- Then 通知の本文には生の IP アドレス、生の User-Agent、トークン、資格情報のいずれも含まれない

### Example: EX-AUTHENTICATION-031-02 メールの配送に失敗する

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- When ユーザー "alice" のパスワード、認証要素、復旧コード、または信頼済みデバイスが増減する
- Then "alice" の検証済みアドレスへセキュリティ通知が送られる
- Then メールの配送に失敗する
- Then 資格情報の変更は成立したままで、配送の失敗は呼び出し元へ伝播しない

## Rule: REQ-AUTHENTICATION-032 メールアドレスの変更は変更前のアドレスへ通知される

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-032-01 通常経路

- Given ユーザー "alice" の検証済みのメールアドレスは "old@example.test" である
- When ユーザー "alice" がメールアドレスの "new@example.test" への変更を要求する
- Then セキュリティ通知は "old@example.test" へ送られる
- When その変更が確定する
- Then セキュリティ通知は "new@example.test" へ送られる

## Rule: REQ-AUTHENTICATION-033 必須の種別の通知は本人が止められない

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-033-01 通常経路

- Given ユーザー "alice" はステップアップ認証を成立させたセッションで認証済みである
- When ユーザー "alice" が自身の通知設定を取得する
- Then 全種別が返り、資格情報・認証要素・連絡先・なりすましの各種別は mandatory として示される
- When ユーザー "alice" が必須の種別を含めて受信の停止を要求する
- Then 要求は拒否され、設定はいずれの種別についても変更されない

### Example: EX-AUTHENTICATION-033-02 ステップアップ認証を成立させていないセッションで更新を要求する

- Given ユーザー "alice" はステップアップ認証を成立させたセッションで認証済みである
- When ユーザー "alice" が自身の通知設定を取得する
- Then 全種別が返り、資格情報・認証要素・連絡先・なりすましの各種別は mandatory として示される
- When ユーザー "alice" が必須の種別を含めて受信の停止を要求する
- Then ステップアップ認証を成立させていないセッションで更新を要求する
- Then ステップアップ認証による再認証が要求される

## Rule: REQ-AUTHENTICATION-034 停止した種別の通知は送られない

Primary actor: `AuthenticatedSelf`

### Example: EX-AUTHENTICATION-034-01 通常経路

- Given ユーザー "alice" は検証済みのメールアドレスを持つ
- When ユーザー "alice" がステップアップ認証を成立させ、既知でない端末からのサインイン通知の受信を停止する
- Then 以後、既知でない端末から認証しても通知は送られない
- Then 資格情報の変更に対する通知は引き続き送られる

## Rule: REQ-AUTHENTICATION-035 サインアウトはテナントのエンドポイント形式によらずサーバー側のセッションを失効させる

Primary actor: `EndUser`

### Example: EX-AUTHENTICATION-035-01 通常経路

- Given ユーザー "alice" がサブドメイン形式のテナントでログインし、ブラウザは `__Host-` 接頭辞つきのセッション Cookie を保持している
- When ユーザー "alice" が SAML シングルログアウトでサインアウトする
- Then そのセッションはサーバー側で失効し、以後の認証解決は未認証として扱われる
- Then ブラウザの Cookie を復元して同じセッション ID を再提示しても認証されない

### Example: EX-AUTHENTICATION-035-02 WS-Federation のサインアウトを使う

- Given ユーザー "alice" がサブドメイン形式のテナントでログインし、ブラウザは `__Host-` 接頭辞つきのセッション Cookie を保持している
- When ユーザー "alice" が SAML シングルログアウトでサインアウトする
- But WS-Federation のサインアウトを使う
- Then 同じくサーバー側のセッションが失効する

### Example: EX-AUTHENTICATION-035-03 パス形式のテナントで接頭辞のない Cookie を送る

- Given ユーザー "alice" がサブドメイン形式のテナントでログインし、ブラウザは `__Host-` 接頭辞つきのセッション Cookie を保持している
- When ユーザー "alice" が SAML シングルログアウトでサインアウトする
- But パス形式のテナントで接頭辞のない Cookie を送る
- Then 同じくサーバー側のセッションが失効する
