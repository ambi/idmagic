# Authentication Scenarios

### REQ-AUTHENTICATION-001: 外部 OIDC 認証は検証済みの subject を常に同じローカル User へ相関する
- ACTOR EndUser
- GIVEN リクエスト先のテナントで OIDC 接続が `Active` である
- GIVEN issuer、認可エンドポイント、トークンエンドポイント、JWKS は登録時に検証済みである
- WHEN EndUser が StartFederatedLogin を開始する
- THEN `state`、`nonce`、PKCE を単回限りのログイン試行として保存し、上流へ遷移する
- WHEN 上流のコールバックが認可コードと ID Token を返す
  - ALT 同じ `state` またはトークンレスポンスを再利用する → 単回限りの試行と再送防止によって拒否する
- THEN CompleteFederatedLogin は code と ID Token の署名、issuer、audience、時刻、nonce を検証する
  - ALT `state`、`nonce`、issuer、audience、署名、時刻のいずれかが一致しない → コールバックを拒否し、LoginSession も関連付けも作成しない → FederatedLoginRejected を発行する
- THEN 初回は、明示した JIT ポリシーとクレームの対応付けに従ってローカルの User と FederatedIdentity を作成する
- THEN 2 回目以降は、同じテナント・プロバイダー・外部 subject の既存の関連付けから同じローカル User を解決する
- THEN AMR に `federated` を持つ LoginSession を発行する

### REQ-AUTHENTICATION-002: 検証済みメールアドレスによる自動リンクは明示ポリシーと一意な一致を要求する
- ACTOR EndUser
- GIVEN その外部 subject に対する既存の関連付けはない
- GIVEN 同じメールアドレスを持つローカル User がテナント内に存在する
- GIVEN 接続の `linking_policy` が `VerifiedEmail` である
- GIVEN 上流の `email_verified` クレームが true で、メールアドレスがテナント内で一意に一致する
- WHEN EndUser が未連携の外部 subject でフェデレーションログインを完了する
  - ALT ポリシーが `None`、メールアドレスが未検証、または一致が一意でない → 自動リンクと LoginSession の発行を拒否する
- THEN 既存の User に対して FederatedIdentity を作成する

### REQ-AUTHENTICATION-003: 外部アイデンティティの明示的なリンクと解除はステップアップ認証を要求する
- ACTOR AuthenticatedSelf
- GIVEN ResourceOwner は対象テナントの有効な User である
- WHEN 直近 5 分以内にステップアップ認証を済ませたセッションで、外部プロバイダーの認証を完了する
  - ALT ステップアップ認証が古い、または行われていない → リンクと解除を AccessDeniedError で拒否する
- THEN その外部 subject が未使用であれば、自身へリンクする
- WHEN 直近 5 分以内にステップアップ認証を済ませたセッションでリンクの解除を要求する
  - ALT パスワード資格情報も他の外部アイデンティティのリンクも残らなくなる → 締め出しを防ぐため解除を拒否する
- THEN 対象の外部アイデンティティのリンクを解除する

### REQ-AUTHENTICATION-004: API トークンの発行者は機密操作のスコープで自身の認証情報だけを操作できる
- ACTOR SelfApiClient
- GIVEN クライアントは対象テナントの有効な User に固定された、有効な API アクセストークンを提示している
- WHEN クライアントがアカウントのセキュリティ設定、サインイン履歴、セッション、MFA 認証要素、復旧コード、またはパスワードの操作を要求する
  - ALT 対応しないスコープで機密操作の変更を要求する → 操作は AccessDeniedError で拒否される
  - ALT トークンのテナントまたは `user_id` が操作対象と一致しない → 操作は AccessDeniedError で拒否される
  - ALT API トークンでステップアップ認証のエンドポイントを要求する → 操作は AccessDeniedError で拒否される
- THEN `account:read` スコープは、自身のアカウント情報、セキュリティ設定、サインイン履歴、セッションの参照だけを許可する
- THEN `account:mfa:write` スコープは、自身の MFA 認証要素と復旧コードの変更だけを許可する
- THEN `account:sessions:write` スコープは、自身のセッションの失効だけを許可する
- THEN `account:password:write` スコープと現在のパスワードの提示は、自身のパスワードの変更だけを許可する

### REQ-AUTHENTICATION-005: ブラウザーの初期化情報は認証状態と CSRF 境界を保持する
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みセッション、またはファーストパーティーのポータルのアクセストークンを持つ
- WHEN ブラウザーまたは API クライアントがアカウントコンテキストをリクエストする
  - ALT セッションが未認証または認証途中である → アカウントコンテキストの取得を AccessDeniedError で拒否する
  - ALT Bearer トークンが許可されたポータルスコープまたは `account:read` スコープを 1 つも持たない → アカウントコンテキストの取得を AccessDeniedError で拒否する
- THEN 管理ポータルは `idmagic.admin`、アカウントポータルは `idmagic.account`、自己管理 API クライアントは `account:read` スコープで同じアカウントコンテキストを取得できる
- THEN レスポンスは subject、realm、実効ロール、CSRF トークンを含む
- WHEN 未認証のパスワードリセット画面がパスワードリセットコンテキストをリクエストする
- THEN CSRF トークンを含むコンテキストが返る

### REQ-AUTHENTICATION-006: ユーザーは WebAuthn でステップアップ認証のチャレンジを開始できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が WebAuthn のクレデンシャルを登録済みで、認証済みセッションを持つ
- WHEN ユーザー "alice" が正しい CSRF トークンでステップアップ認証の WebAuthn チャレンジを要求する
  - ALT CSRF トークンが一致しない、または WebAuthn を利用できない → チャレンジは発行されず、要求を拒否する
- THEN レスポンスの `PublicKeyCredentialRequestOptions` は現在のセッションに束縛される

### REQ-AUTHENTICATION-007: ResourceOwner はブラウザーでパスワード認証し、認可を継続する
- ACTOR ResourceOwner
- GIVEN 未認証セッションで "web-app" として認可リクエストを送信済みである
- WHEN ブラウザーのログイン API にユーザー名 "alice" と正しいパスワードを送信する
  - ALT SameSite の Cookie とリクエストのトークンが一致しない → CSRF の値を改ざんしてログイン API を送信する → エラー "InvalidRequestError"
  - ALT 直近 900 秒の時間枠で、アカウント単位の失敗回数が 10 回に達している → 正しいパスワードでログイン API を送信する → エラー "RateLimitedError" → "LoginThrottled" が発行される
  - ALT 失敗回数によらず、同一 IP からのログイン API リクエストが `EndpointRateLimitPolicy` の時間枠内で上限に達している → 正しいパスワードでログイン API を送信する → エラー "RateLimitedError"
- THEN セッション Cookie が発行される
- THEN 認可コードが redirect_uri に返る
- THEN "UserAuthenticated" が発行される

### REQ-AUTHENTICATION-008: パスワードリセットの要求は識別子と IP の組で流量制限される
- ACTOR EndUser
- GIVEN 未認証である
- WHEN "alice" 宛のパスワードリセットを要求する
  - ALT 同じ識別子と IP の組で、`EndpointRateLimitPolicy` の時間枠内の上限に達している → "alice" 宛のパスワードリセットを再度要求する → エラー "RateLimitedError"
- THEN ユーザーの存在にかかわらず 204 を返す
- THEN "PasswordResetRequested" が発行される

### REQ-AUTHENTICATION-009: 無効なユーザーは新規ログインも既存セッションも拒否される
- ACTOR EndUser
- GIVEN ユーザー "alice" は無効状態であり、無効化の前に取得した認証済みセッションを持つ
- WHEN ユーザー "alice" が既存セッションで認証必須 API を呼ぶ
- THEN エラー "AccessDeniedError"
- WHEN ユーザー "alice" が正しいパスワードで新規ログインを試みる
- THEN エラー "AccessDeniedError"

無効化そのものは IdManagement の操作であり、無効化から到達経路が閉じるまでの連鎖は REQ-PLATFORM-001 が持つ。ここは、無効な主体を Authentication が単独で拒否することだけを述べる。

### REQ-AUTHENTICATION-010: ユーザーは現在のパスワードを確認して新しいパスワードへ変更できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでパスワード変更画面を開いている
- WHEN ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
  - ALT 新しいパスワードが 12 文字未満である → ユーザー "alice" が 12 文字未満のパスワードを送信する → エラー "InvalidRequestError"
  - ALT 新しいパスワードが直近 5 件の履歴に一致する → ユーザー "alice" が直近使用した過去のパスワードを新パスワードとして送信する → エラー "InvalidRequestError"
- THEN パスワードが変更され、`password_changed_at` が更新される
- THEN "PasswordChanged" が発行される

### REQ-AUTHENTICATION-011: ユーザーは TOTP 認証要素を登録して有効化できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでセキュリティ画面を開いている
- WHEN ユーザー "alice" が TOTP 登録を開始する
- THEN レスポンスにシークレットとアカウント名が含まれる
- WHEN ユーザー "alice" がそのシークレットに対する正しいコードで登録を確認する
- THEN セキュリティ概要の MFA 状態が登録済みになる
- THEN "MfaFactorEnrolled" が発行される

### REQ-AUTHENTICATION-012: ユーザーはステップアップ再認証のうえで TOTP 認証要素を解除する
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が登録済みの TOTP 認証要素を持ち認証済みである
- WHEN ユーザー "alice" がステップアップ認証を成立させ現在の TOTP コードで解除する
  - ALT ステップアップ認証なしで解除を試みる → ユーザー "alice" がステップアップ認証なしで TOTP 認証要素の解除を試みる → ステップアップ認証による再認証が要求される
- THEN TOTP 認証要素が解除される
- THEN "MfaFactorRemoved" が発行される

### REQ-AUTHENTICATION-013: ユーザーは自分の有効なセッションを一覧して失効できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- WHEN ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
  - ALT プロセスの再起動を挟んでセッション一覧を取得する → サーバープロセスを再起動する → ユーザー "alice" が同じセッション Cookie でアクティビティ画面を開く → セッションは再起動前と同じ内容で解決できる
- THEN 自分の有効なセッションが返る
- WHEN ユーザー "alice" が現在以外のセッションを 1 件失効させる
  - ALT 既に失効済みのセッションへ同じ失効操作を再送する → ユーザー "alice" が直前に失効させた同じセッション ID へ再度失効を要求する → 要求は成功として扱われ、最初の失効時刻を保持する
- THEN 失効したセッションは一覧から消える
- WHEN ユーザー "alice" が現在以外のすべてのセッションを一括失効させる
- THEN 現在のセッションだけが残る

### REQ-AUTHENTICATION-014: ユーザーは自分のサインイン履歴を確認できる
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" が認証済みでアクティビティ画面を開いている
- WHEN ユーザー "alice" が自分のサインイン履歴を取得する
  - ALT 認証手段に WebAuthn が含まれる → UI は `webauthn` という技術名ではなく「パスキー」と表示する
- THEN レスポンスに自分のサインインイベントだけが含まれる
- THEN 第二要素を使ったサインインは、`pwd` と第二要素の `amr` を持つ完了後の `UserAuthenticated` として表示される

### REQ-AUTHENTICATION-015: MFA 登録済みでも、ポリシーが要求しない限り第二要素は求めない
- ACTOR EndUser
- GIVEN ユーザー "alice" は TOTP または WebAuthn のクレデンシャルを登録済みである
- GIVEN 対象 Application の実効サインインポリシーは `Password` である
- WHEN ユーザー "alice" がユーザー名とパスワードを送信する
  - ALT 対象 Application の実効サインインポリシーが `Mfa` である → LoginSession は `authentication_pending=true` へ切り替わる → 利用できる第二要素 (TOTP / パスキー / 復旧コード) の選択画面へ進む
- THEN LoginSession は `authentication_pending=false` で作られる
- THEN 認可フローは第二要素画面に進まず、同意または認可コード発行へ進む

### REQ-AUTHENTICATION-016: ユーザーはメールのリセットリンクでパスワードを再設定する
- ACTOR EndUser
- GIVEN ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- WHEN ユーザー "alice" がそのトークンと新しいパスワードを送信する
  - ALT トークンが期限切れまたは不正である → 無効なパスワードリセットトークンで新しいパスワードを送信する → エラー "InvalidRequestError"
- THEN パスワードが更新される
- WHEN ユーザー "alice" が新しいパスワードをブラウザーのログイン API へ送信する
- THEN ログインに成功する
- WHEN EndUser が未登録のメールアドレスでパスワードリセットを要求する
- THEN レスポンスは登録済みアドレスに対するものと区別できない
- WHEN EndUser が登録済みのメールアドレスでパスワードリセットを要求する
- THEN 登録済みアドレスへリセットリンクが送られる

### REQ-AUTHENTICATION-017: TOTP が必須のユーザーは正しいコードで認証を継続できる
- ACTOR EndUser
- GIVEN TOTP 認証要素が登録された `authentication_pending` の LoginSession が存在する
- WHEN ブラウザーの TOTP API に正しいコードを送信する
  - ALT 誤った TOTP コードを送信する → ブラウザーの TOTP API に誤ったコードを送信する → エラー "InvalidRequestError" → LoginSession は `authentication_pending` のままである
- THEN 認証が成立し認可フローが継続する
- THEN "UserAuthenticated" が発行される

### REQ-AUTHENTICATION-018: MFA 未登録のユーザーは管理者が承認した登録を終えて同じ認可処理を継続できる
- ACTOR EndUser
- GIVEN 対象 Application の実効ポリシーは MFA 必須かつ強制開始済みで、登録バイパスを許可し猶予期限内である
- GIVEN ユーザーは TOTP と WebAuthn のいずれの認証要素も持たない
- GIVEN 管理者が対象ユーザーへ有効な単回限りの登録バイパスを発行済みである
- WHEN ユーザーが正しいパスワードを送信する
  - ALT 登録バイパスがない、取り消し済み、消費済み、または期限切れである → パスワードが正しくてもログインを完了せずアクセスを拒否する → 認証要素の登録 API は MfaEnrollmentNotAllowedError で拒否し、認証要素を作らない
- THEN バイパスを消費し、同じ LoginSession は `pending_purpose=Enrollment` の未完了状態になる
- THEN `MfaEnrollmentRequired` と `MfaEnrollmentBypassConsumed` が発行され、登録専用画面へ進む
- WHEN ユーザーが TOTP のシークレットに対する正しいコードで登録を確定する
  - ALT 登録期限を過ぎている → 認証要素を保存せずアクセスを拒否する → LoginSession を認証完了へ昇格させない
  - ALT TOTP コードが不正である → 認証要素を保存せず InvalidRequestError を返す → LoginSession は `Enrollment` の保留状態のままである
- THEN 認証要素が保存され、同じ LoginSession の `amr` に `otp` が追加されて保留状態が解除される
- THEN `MfaEnrollmentCompleted` と `UserAuthenticated` が発行され、元の認可トランザクションが継続する

### REQ-AUTHENTICATION-019: MFA の強制開始前は、未登録のユーザーもログインできるが登録を促される
- ACTOR EndUser
- GIVEN テナントデフォルトポリシーは将来時刻から MFA 必須になる
- GIVEN ユーザーは MFA 認証要素を持たない
- WHEN ユーザーが正しいパスワードでログインする
- THEN 強制開始前なので、パスワードだけのセッションが成立する
- THEN UI は強制開始日時と事前登録を促す警告を表示する
- THEN ユーザーは通常のステップアップ認証を経たアカウントのセキュリティ設定画面から認証要素を事前登録できる

### REQ-AUTHENTICATION-020: 登録待ちのセッションは通常のリソースへアクセスできない
- ACTOR EndUser
- GIVEN `pending_purpose=Enrollment` の LoginSession が存在する
- WHEN ユーザーがアカウント、管理、Application のいずれかのリソースを要求する
- THEN 未認証として拒否する
- THEN 登録の開始と確定の API、および元の認可トランザクションだけを許可する

### REQ-AUTHENTICATION-021: 管理者は対象ユーザーのセッションを一覧・個別失効・全失効できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" が複数の有効な LoginSession を持つ
- WHEN 管理者がユーザー "alice" の ListSessions を呼ぶ
  - ALT 他テナントの管理者が呼び出す → エラー "AccessDeniedError"
- THEN 開始時刻の降順で有効なセッション一覧が返る
- WHEN 管理者がそのうち 1 件の `RevokeSession` を呼ぶ
  - ALT 既に失効済みのセッションへ再度 `RevokeSession` を呼ぶ → 204 が返り、`revoked_at` は初回の値を保持する
- THEN 対象セッションは `revoke_reason=admin_revoke` で失効し、"SessionEnded" が発行される
- WHEN 管理者がユーザー "alice" の RevokeUserSessions を呼ぶ
- THEN 残り全セッションが失効する

### REQ-AUTHENTICATION-022: 管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は TOTP 認証要素を持ち、復旧コードも生成済みである
- WHEN 管理者がユーザー "alice" の ResetUserAuthenticators を targets=[Totp, RecoveryCode] で呼ぶ
  - ALT 他テナントの管理者、または `admin` ロールを持たない操作者が呼び出す → エラー "AccessDeniedError" → 対象ユーザーの認証器は変更されない
- THEN "AuthenticatorResetRequested" が発行される
- THEN TOTP 認証要素と復旧コードが削除され、他に WebAuthn クレデンシャルもないため `mfa_enrolled` が `false` になる
- THEN `reenrollment_required=true` のレスポンスが返り、単回限りの登録バイパスを自動発行する
- THEN "AuthenticatorResetCompleted" と "MfaEnrollmentBypassIssued" が発行される
- WHEN "alice" が正しいパスワードで次にログインする
- THEN 有効なバイパスにより、同じ LoginSession が `pending_purpose=Enrollment` になる
- WHEN "alice" が新しい TOTP 認証要素の登録を確定する
- THEN 同じ LoginSession が MFA 済みへ昇格し、元の認可トランザクションが継続する

### REQ-AUTHENTICATION-023: 管理者が一部の認証器のみリセットした場合は残存要素でログインを継続できる
- ACTOR TenantAdministrator
- GIVEN ユーザー "bob" は TOTP 認証要素と WebAuthn クレデンシャルを両方持つ
- WHEN 管理者がユーザー "bob" の ResetUserAuthenticators を targets=[Webauthn] で呼ぶ
- THEN WebAuthn クレデンシャルだけが削除され、TOTP 認証要素は残るため `mfa_enrolled` は `true` のままである
- THEN `reenrollment_required=false` のレスポンスが返り、登録バイパスは発行されない
- WHEN "bob" が次回のログインで TOTP コードによる第二要素の検証を完了する
- THEN ログインを完了できる

### REQ-AUTHENTICATION-024: 有効期限を過ぎたパスワードのユーザーは次回ログイン後にパスワード変更を強制される
- ACTOR EndUser
- GIVEN テナントのパスワードポリシーは `max_age_days=90` で、ポリシーの更新から 90 日以上が経過している
- GIVEN ユーザー "alice" の `password_changed_at` は 91 日前である
- WHEN ユーザー "alice" が正しいパスワードでログインする
  - ALT `password_changed_at` が 89 日前である → ログインはそのまま完了し、`update_password` は付与されない
  - ALT `max_age_days` が未設定である → 経過日数によらず `update_password` は付与されない
  - ALT ポリシーの更新から 90 日が経過していない → 猶予期間内なので `update_password` は付与されない
  - ALT ユーザーがパスワード資格情報を持たない (フェデレーションまたはパスワードレス) → `update_password` は付与されない
- THEN ログイン自体は成功する
- THEN ユーザー "alice" に必須操作 `update_password` が付与される
- THEN ユーザー "alice" はパスワード変更画面へ誘導され、変更完了までフローを継続できない
- WHEN ユーザー "alice" がポリシーを満たす新しいパスワードへ変更する
- THEN `update_password` が解除され、"PasswordChanged" が発行される

### REQ-AUTHENTICATION-025: 外部 IdP 接続の管理は対話セッションに限る
- ACTOR ManagementApiClient
- GIVEN クライアントは対象テナントの有効な API アクセストークンを提示している
- GIVEN トークンの発行者は `admin` ロールを持つ
- WHEN クライアントが外部 IdP 接続の参照または変更をリクエストする
  - ALT トークンが `ApiTokenScope` のどのスコープを持っていても → 操作は `insufficient_scope` で拒否され、必要な資格として対話セッションを提示する
- THEN 外部 IdP 接続の管理は、ブラウザーのログインセッションまたは管理ポータルのアクセストークンからのみ行える
- WHEN 同じクライアントがセッションと認証情報の管理 API をリクエストする
- THEN `sessions:read` は利用者のセッションとサインイン履歴の参照を、`sessions:write` はセッションの失効を、`users:write` は MFA 登録の一時免除と認証器のリセットを許可する

### REQ-AUTHENTICATION-026: 第二要素の成立時に本人が同意した端末は次回以降の第二要素を省略できる
- ACTOR EndUser
- GIVEN テナントの `trusted_device_max_age_seconds` は正の値である
- GIVEN 対象 Application の実効サインインポリシーは `Mfa` で `allow_trusted_device=true` である
- GIVEN ユーザー "alice" は TOTP 認証要素を登録済みである
- WHEN ユーザー "alice" が正しいパスワードに続けて正しい TOTP コードを送信し、このデバイスを記憶することに同意する
  - ALT テナントの `trusted_device_max_age_seconds` が 0 または未設定である → 同意は無視され、デバイスは記憶されない
  - ALT 第二要素として復旧コードを消費した → デバイスは記憶されない
  - ALT パスワードだけで認証が完了した (ポリシーが MFA を要求していない) → デバイスは記憶されない
- THEN 認証が成立し、realm scope の HttpOnly cookie として信頼済みデバイスの資格情報が発行される
- THEN "TrustedDeviceRegistered" が発行される
- WHEN 同じブラウザーでユーザー "alice" が正しいパスワードを送信する
- THEN 第二要素の画面へ進まずに認証が成立し、`amr` に `tdev` が加わって `acr` が `urn:idmagic:acr:mfa` になる
- THEN 信頼済みデバイスの verifier が回転し、更新された cookie が再発行される

### REQ-AUTHENTICATION-027: 期限切れ・盗難・別テナントの信頼済みデバイス cookie は第二要素を省略できない
- ACTOR EndUser
- GIVEN ユーザー "alice" は 1 つの信頼済みデバイスを持ち、対象 Application の実効サインインポリシーは `Mfa` である
- WHEN ユーザー "alice" が絶対期限を過ぎた cookie を提示して正しいパスワードを送信する
  - ALT 直近利用から idle 期限を過ぎた cookie を提示する → 第二要素を要求する
  - ALT 回転前の古い cookie を提示する → 第二要素を要求する
  - ALT 別テナントの realm で発行された cookie を提示する → 第二要素を要求する
  - ALT selector は正しいが verifier が一致しない cookie を提示する → 第二要素を要求する
- THEN LoginSession は `authentication_pending=true` になり、第二要素の選択画面へ進む
- THEN `amr` に `tdev` は加わらない

### REQ-AUTHENTICATION-028: 資格情報が変わると信頼済みデバイスはすべて失効する
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" は有効な信頼済みデバイスを持つ
- WHEN ユーザー "alice" が自身のパスワードを変更する
  - ALT メールのリセットリンクでパスワードを再設定する → 同じく全デバイスが失効する
  - ALT ユーザー "alice" が TOTP 認証要素を登録または解除する → 同じく全デバイスが失効する
  - ALT 管理者がユーザー "alice" の認証器をリセットする → 同じく全デバイスが失効する
  - ALT 管理者がユーザー "alice" を無効化する → 同じく全デバイスが失効する
  - ALT ユーザー "alice" が他のセッションを一括失効させる → 同じく全デバイスが失効する
- THEN ユーザー "alice" の信頼済みデバイスはすべて失効し、"TrustedDeviceRevoked" が発行される
- WHEN 失効した端末でユーザー "alice" が再びログインする
- THEN 第二要素が再び要求される

### REQ-AUTHENTICATION-029: 信頼済みデバイスは機微操作の再認証を肩代わりしない
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" は信頼済みデバイスによって `amr` に `tdev` を持つセッションで認証済みである
- GIVEN そのセッションはステップアップ認証を行っていない
- WHEN ユーザー "alice" がパスワードの変更、TOTP 認証要素の解除、または他セッションの一括失効を要求する
- THEN ステップアップ認証による再認証が要求される
- WHEN ユーザー "alice" が自身の信頼済みデバイスを一覧する
  - ALT ステップアップ認証なしで信頼済みデバイスの失効を要求する → ステップアップ認証による再認証が要求される
- THEN selector と verifier を含まない一覧が最終利用時刻の降順で返り、現在の端末が current として示される
- WHEN ユーザー "alice" がステップアップ認証を成立させて信頼済みデバイスを失効させる
  - ALT 既に失効済みのデバイスへ同じ失効操作を再送する → 要求は成功として扱われ、最初の失効時刻を保持する
- THEN 対象は一覧から消え、"TrustedDeviceRevoked" が発行される

### REQ-AUTHENTICATION-030: 既知でない端末からのサインインだけがセキュリティ通知を生む
- ACTOR EndUser
- GIVEN ユーザー "alice" は検証済みのメールアドレスを持つ
- GIVEN ユーザー "alice" はこれまで一度もサインインしていないブラウザーを使っている
- WHEN ユーザー "alice" がそのブラウザーで認証に成功する
- THEN そのブラウザーは既知の端末として記録される
- THEN "alice" の検証済みアドレスへセキュリティ通知が送られ、"AccountSecurityNotificationSent" が発行される
  - ALT "alice" が検証済みのメールアドレスを持たない → 通知は送られず、認証は成功したままである
- WHEN ユーザー "alice" が同じブラウザーで再び認証に成功する
- THEN 通知は送られず、その端末の最終利用時刻だけが更新される

### REQ-AUTHENTICATION-031: 資格情報の変更は本人へ通知され、通知の失敗は変更を巻き戻さない
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" は検証済みのメールアドレスを持つ
- WHEN ユーザー "alice" のパスワード、認証要素、復旧コード、または信頼済みデバイスが増減する
- THEN "alice" の検証済みアドレスへセキュリティ通知が送られる
- THEN 通知の本文には生の IP アドレス、生の User-Agent、トークン、資格情報のいずれも含まれない
  - ALT メールの配送に失敗する → 資格情報の変更は成立したままで、配送の失敗は呼び出し元へ伝播しない

### REQ-AUTHENTICATION-032: メールアドレスの変更は変更前のアドレスへ通知される
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" の検証済みのメールアドレスは "old@example.test" である
- WHEN ユーザー "alice" がメールアドレスの "new@example.test" への変更を要求する
- THEN セキュリティ通知は "old@example.test" へ送られる
- WHEN その変更が確定する
- THEN セキュリティ通知は "new@example.test" へ送られる

### REQ-AUTHENTICATION-033: 必須の種別の通知は本人が止められない
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" はステップアップ認証を成立させたセッションで認証済みである
- WHEN ユーザー "alice" が自身の通知設定を取得する
- THEN 全種別が返り、資格情報・認証要素・連絡先・なりすましの各種別は mandatory として示される
- WHEN ユーザー "alice" が必須の種別を含めて受信の停止を要求する
- THEN 要求は拒否され、設定はいずれの種別についても変更されない
  - ALT ステップアップ認証を成立させていないセッションで更新を要求する → ステップアップ認証による再認証が要求される

### REQ-AUTHENTICATION-034: 停止した種別の通知は送られない
- ACTOR AuthenticatedSelf
- GIVEN ユーザー "alice" は検証済みのメールアドレスを持つ
- WHEN ユーザー "alice" がステップアップ認証を成立させ、既知でない端末からのサインイン通知の受信を停止する
- THEN 以後、既知でない端末から認証しても通知は送られない
- THEN 資格情報の変更に対する通知は引き続き送られる
