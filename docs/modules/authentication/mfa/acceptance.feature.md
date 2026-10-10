# Feature: 多要素認証の例

## Rule: REQ-AUTHENTICATION-018 MFA 未登録のユーザーは管理者が承認した登録を終えて同じ認可処理を継続できる

### Background:

- Given 対象 Application の実効ポリシーは MFA 必須かつ強制開始済みで、登録バイパスを許可し猶予期限内である
- And ユーザーは TOTP と WebAuthn のいずれの認証要素も持たない
- And 管理者が対象ユーザーへ有効な単回限りの登録バイパスを発行済みである

### Example: EX-AUTHENTICATION-018-01 通常経路

- When ユーザーが正しいパスワードを送信する
- Then バイパスを消費し、同じ LoginSession は `pending_purpose=Enrollment` の未完了状態になる
- Then `MfaEnrollmentRequired` と `MfaEnrollmentBypassConsumed` が発行され、登録専用画面へ進む
- When ユーザーが TOTP のシークレットに対する正しいコードで登録を確定する
- Then 認証要素が保存され、同じ LoginSession の `amr` に `otp` が追加されて保留状態が解除される
- Then `MfaEnrollmentCompleted` と `UserAuthenticated` が発行され、元の認可トランザクションが継続する

### Example: EX-AUTHENTICATION-018-02 登録バイパスがない、取り消し済み、消費済み、または期限切れである

- When ユーザーが正しいパスワードを送信する
- But 登録バイパスがない、取り消し済み、消費済み、または期限切れである
- Then パスワードが正しくてもログインを完了せずアクセスを拒否する
- And 認証要素の登録 API は MfaEnrollmentNotAllowedError で拒否し、認証要素を作らない

### Scenario Outline: 条件ごとの結果

- When ユーザーが正しいパスワードを送信する
- Then バイパスを消費し、同じ LoginSession は `pending_purpose=Enrollment` の未完了状態になる
- Then `MfaEnrollmentRequired` と `MfaEnrollmentBypassConsumed` が発行され、登録専用画面へ進む
- When ユーザーが TOTP のシークレットに対する正しいコードで登録を確定する
- But <condition>
- Then <result>
- And <result_2>

#### Examples:

  | example_id | condition | result | result_2 |
  | --- | --- | --- | --- |
  | EX-AUTHENTICATION-018-03 | 登録期限を過ぎている | 認証要素を保存せずアクセスを拒否する | LoginSession を認証完了へ昇格させない |
  | EX-AUTHENTICATION-018-04 | TOTP コードが不正である | 認証要素を保存せず InvalidRequestError を返す | LoginSession は `Enrollment` の保留状態のままである |

## Rule: REQ-AUTHENTICATION-019 MFA の強制開始前は、未登録のユーザーもログインできるが登録を促される

### Example: EX-AUTHENTICATION-019-01 通常経路

- Given テナントデフォルトポリシーは将来時刻から MFA 必須になる
- And ユーザーは MFA 認証要素を持たない
- When ユーザーが正しいパスワードでログインする
- Then 強制開始前なので、パスワードだけのセッションが成立する
- Then UI は強制開始日時と事前登録を促す警告を表示する
- Then ユーザーは通常のステップアップ認証を経たアカウントのセキュリティ設定画面から認証要素を事前登録できる

## Rule: REQ-AUTHENTICATION-015 MFA 登録済みでも、ポリシーが要求しない限り第二要素は求めない

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

## Rule: REQ-AUTHENTICATION-020 登録待ちのセッションは通常のリソースへアクセスできない

### Example: EX-AUTHENTICATION-020-01 通常経路

- Given `pending_purpose=Enrollment` の LoginSession が存在する
- When ユーザーがアカウント、管理、Application のいずれかのリソースを要求する
- Then 未認証として拒否する
- Then 登録の開始と確定の API、および元の認可トランザクションだけを許可する

## Rule: REQ-AUTHENTICATION-022 管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる

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

### Example: EX-AUTHENTICATION-023-01 通常経路

- Given ユーザー "bob" は TOTP 認証要素と WebAuthn クレデンシャルを両方持つ
- When 管理者がユーザー "bob" の ResetUserAuthenticators を targets=[Webauthn] で呼ぶ
- Then WebAuthn クレデンシャルだけが削除され、TOTP 認証要素は残るため `mfa_enrolled` は `true` のままである
- Then `reenrollment_required=false` のレスポンスが返り、登録バイパスは発行されない
- When "bob" が次回のログインで TOTP コードによる第二要素の検証を完了する
- Then ログインを完了できる
