# Authentication Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-AUTHENTICATION-001: 外部OIDC認証は検証済みsubjectを同じlocal Userへ相関する
- Actor: EndUser
- Given: request tenant で OIDC connection が Active である
- Given: issuer、authorization endpoint、token endpoint、JWKS は管理時に検証済みである
- Then: StartFederatedLogin は state、nonce、PKCE を単発 attempt に保存して upstream へ遷移する
- Then: CompleteFederatedLogin は code と ID Token の署名、issuer、audience、時刻、nonce を検証する
- Then: 初回は明示 JIT policy と claim mapping により local User と FederatedIdentity を作成する
- Then: 2回目は同じ tenant、provider、external subject の既存 link から同じ local User を解決する
- Then: federated AMR の LoginSession を発行する
- Alternative (state、nonce、issuer、audience、署名、時刻のいずれかが一致しない): callback を拒否し LoginSession と link を作成しない → FederatedLoginRejected を発行する
- Alternative (同じ state または token response を再利用する): single-use attempt / replay guard が拒否する

### REQ-AUTHENTICATION-002: verified emailによる自動linkは明示policyと一意一致を要求する
- Actor: EndUser
- Given: external subject の既存 link は無い
- Given: 同じ email の local User が tenant 内に存在する
- Then: provider の linking_policy が VerifiedEmail である
- Then: upstream email_verified claim が true である
- Then: email が tenant 内で一意に一致する
- Then: FederatedIdentity を既存 User に作成する
- Alternative (policy が None、email が未検証、または一致が曖昧である): 自動 link と LoginSession 発行を拒否する

### REQ-AUTHENTICATION-003: external identityの明示linkとunlinkはstep-upを要求する
- Actor: AuthenticatedSelf
- Given: ResourceOwner は対象 tenant の active User である
- Then: 直近5分以内の step-up session で provider の外部認証を完了する
- Then: external subject が未使用なら自身へ link する
- Then: 直近5分以内の step-up session で link を解除する
- Alternative (step-up が古い、または無い): link / unlink を AccessDeniedError で拒否する
- Alternative (password credential も他の external identity link も残らない): account lockout 防止のため unlink を拒否する

### REQ-AUTHENTICATION-004: API token発行者はsensitive facet scope内で自身のauthentication情報だけを操作できる
- Actor: SelfApiClient
- Given: client は対象 tenant の active User に固定された有効な API access token を提示している
- Then: account:read scope で account context、security、signin activity、session を参照できる
- Then: account:mfa:write scope で自身の MFA factor と recovery code を変更できる
- Then: account:sessions:write scope で自身の session を失効できる
- Then: account:password:write scope と current password で自身の password を変更できる
- Alternative (対応しない account scope で sensitive facet の変更を要求する): 操作は AccessDeniedError で拒否される
- Alternative (token の tenant または user_id が操作対象と一致しない): 操作は AccessDeniedError で拒否される
- Alternative (API token で step-up endpoint を要求する): 操作は AccessDeniedError で拒否される

### REQ-AUTHENTICATION-005: browser bootstrap contextは認証状態とCSRF境界を保持する
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が認証済み session または first-party portal の access token を持つ
- Then: 管理 portal は idmagic.admin scope で account context を取得できる
- Then: account portal は idmagic.account scope で同じ account context を取得できる
- Then: 自己管理 API client は account:read scope で同じ account context を取得できる
- Then: 応答は subject、realm、effective role、CSRF token を含む
- Then: 未認証のパスワードリセット画面は password reset context から CSRF token を取得できる
- Alternative (session が未認証または認証途中である): account context の取得は AccessDeniedError で拒否される
- Alternative (Bearer token が許可された portal scope または account:read scope を一つも持たない): account context の取得は AccessDeniedError で拒否される

### REQ-AUTHENTICATION-006: ユーザーはWebAuthnでstep-up challengeを開始できる
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が WebAuthn credential を登録済みで認証済み session を持つ
- Then: ユーザー "alice" が正しい CSRF token で step-up WebAuthn challenge を要求する
- Then: 応答の PublicKeyCredentialRequestOptions は現在 session に束縛される
- Alternative (CSRF token が一致しない、または WebAuthn が利用不能である): challenge は発行されず要求は拒否される

### REQ-AUTHENTICATION-007: ResourceOwnerはブラウザでパスワード認証し認可を継続する
- Actor: ResourceOwner
- Given: 未認証セッションで "web-app" として認可リクエストを送信済みである
- Then: browser login API に username "alice" と正しい password を送信する
- Then: セッション Cookie が発行される
- Then: 認可コードが redirect_uri に返る
- Then: "UserAuthenticated" が発行される
- Alternative (SameSite cookie と request token が一致しない): csrf 値を改ざんして browser login API を送信する → エラー "InvalidRequestError"
- Alternative (直近 900 秒窓で per-account の失敗回数が 10 回に達している): 正しい password で browser login API を送信する → エラー "RateLimitedError" → "LoginThrottled" が発行される
- Alternative (失敗回数に関わらず同一 IP からの login API リクエストが EndpointRateLimitPolicy の window 内で max_requests に達している): 正しい password で browser login API を送信する → エラー "RateLimitedError"

### REQ-AUTHENTICATION-008: パスワードリセット要求は識別子とIPの組でrate limitされる
- Actor: EndUser
- Given: 未認証である
- Then: "alice" 宛のパスワードリセットを要求する
- Then: user の存在有無に関わらず 204 を返す
- Then: "PasswordResetRequested" が発行される
- Alternative (同一 identifier と IP の組で EndpointRateLimitPolicy の window 内の max_requests に達している): "alice" 宛のパスワードリセットを再度要求する → エラー "RateLimitedError"

### REQ-AUTHENTICATION-009: 無効化されたユーザーは新規ログインも既存セッションも拒否される
- Actor: TenantAdministrator
- Given: ユーザー "alice" が認証済みセッションを持つ
- Then: 管理者がユーザー "alice" を無効化する
- Then: ユーザー "alice" が既存セッションで認証必須 API を呼ぶ
- Then: エラー "AccessDeniedError"
- Alternative (無効化済みユーザーが新規ログインを試みる): ユーザー "alice" が正しい password で browser login API を送信する → エラー "AccessDeniedError"

### REQ-AUTHENTICATION-010: ユーザーは現在のパスワードを確認して新しいパスワードへ変更できる
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が認証済みでパスワード変更画面を開いている
- Then: ユーザー "alice" が正しい現在のパスワードと新しいパスワードを送信する
- Then: パスワードが変更され password_changed_at が更新される
- Then: "PasswordChanged" が発行される
- Alternative (新しいパスワードが 12 文字未満である): ユーザー "alice" が 12 文字未満のパスワードを送信する → エラー "InvalidRequestError"
- Alternative (新しいパスワードが直近 5 件の履歴に一致する): ユーザー "alice" が直近使用した過去のパスワードを新パスワードとして送信する → エラー "InvalidRequestError"

### REQ-AUTHENTICATION-011: ユーザーはTOTP factorを登録して有効化できる
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が認証済みでセキュリティ画面を開いている
- Then: ユーザー "alice" が TOTP 登録を開始する
- Then: 応答に secret と account_name が含まれる
- Then: ユーザー "alice" がその secret に対する正しいコードで登録を確認する
- Then: セキュリティ概要の MFA 状態が登録済みになる
- Then: "MfaFactorEnrolled" が発行される

### REQ-AUTHENTICATION-012: ユーザーはstep-up再認証のうえでTOTP factorを解除する
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が登録済み TOTP factor を持ち認証済みである
- Then: ユーザー "alice" が step-up を成立させ現在の TOTP コードで解除する
- Then: TOTP factor が解除される
- Then: "MfaFactorRemoved" が発行される
- Alternative (step-up なしで解除を試みる): ユーザー "alice" が step-up なしで TOTP factor の解除を試みる → step-up 再認証が要求される

### REQ-AUTHENTICATION-013: ユーザーは自分の有効なセッションを一覧して失効できる
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が複数の有効なセッションを持ち認証済みである
- Then: ユーザー "alice" がアクティビティ画面でセッション一覧を取得する
- Then: ユーザー "alice" が現在以外のセッションを 1 件失効させる
- Then: 失効したセッションは一覧から消える
- Then: ユーザー "alice" が現在以外のすべてのセッションを一括失効させる
- Then: 現在のセッションだけが残る
- Alternative (既に失効済みのセッションへ同じ失効操作を再送する): ユーザー "alice" が直前に失効させた同じセッション id へ再度失効を要求する → 要求は成功として扱われ、最初の失効時刻が保持される
- Alternative (process 再起動を挟んでセッション一覧を取得する): サーバープロセスを再起動する → ユーザー "alice" が同じ session cookie でアクティビティ画面を開く → セッションは再起動前と同じ内容で解決できる

### REQ-AUTHENTICATION-014: ユーザーは自分のサインイン履歴を確認できる
- Actor: AuthenticatedSelf
- Given: ユーザー "alice" が認証済みでアクティビティ画面を開いている
- Then: ユーザー "alice" が自分のサインイン履歴を取得する
- Then: 応答に自分のサインインイベントだけが含まれる
- Then: 第二要素を使ったサインインは、pwd と第二要素の amr を含む完了後の UserAuthenticated として表示される
- Alternative (認証手段に WebAuthn が含まれる): UI は webauthn という技術名ではなく「パスキー」と表示する

### REQ-AUTHENTICATION-015: MFA登録済みユーザーでもポリシーが要求しない限り第二要素を求められない
- Actor: EndUser
- Given: ユーザー "alice" は TOTP または WebAuthn credential を登録済みである
- Given: 対象 Application の実効サインインポリシーは Password である
- Then: ユーザー "alice" がユーザー名とパスワードを送信する
- Then: LoginSession は authentication_pending=false で作られる
- Then: 認可フローは第二要素画面に進まず、同意または認可コード発行へ進む
- Alternative (対象 Application の実効サインインポリシーが Mfa である): LoginSession は authentication_pending=true へ切り替わる → 利用可能な第二要素 (TOTP / パスキー / リカバリコード) の選択画面へ進む

### REQ-AUTHENTICATION-016: ユーザーはメールのリセットリンクでパスワードを再設定する
- Actor: EndUser
- Given: ユーザー "alice" 宛に有効なパスワードリセットトークンが発行されている
- Then: ユーザー "alice" がそのトークンと新しいパスワードを送信する
- Then: パスワードが更新される
- Then: ユーザー "alice" が新しいパスワードで browser login API に成功する
- Alternative (トークンが期限切れまたは不正である): 無効なパスワードリセットトークンで新しいパスワードを送信する → エラー "InvalidRequestError"
- Alternative (リセット要求はアカウントの存在を漏らさない): パスワード再設定画面で未登録のメールアドレスを送信する → 応答は登録済みアドレスと区別できない → パスワード再設定画面で登録済みのメールアドレスを送信する → 登録済みアドレスへリセットリンクが送られる

### REQ-AUTHENTICATION-017: TOTP必須ユーザーは正しいコードで認証を継続できる
- Actor: EndUser
- Given: TOTP factor が登録された authentication_pending の LoginSession が存在する
- Then: browser TOTP API に正しいコードを送信する
- Then: 認証が成立し認可フローが継続する
- Then: "UserAuthenticated" が発行される
- Alternative (誤った TOTP コードを送信する): browser TOTP API に誤ったコードを送信する → エラー "InvalidRequestError" → LoginSession は authentication_pending のままである

### REQ-AUTHENTICATION-018: MFA未登録ユーザーは管理者承認済みオンボーディングを完了して同じ認可処理を継続できる
- Actor: EndUser
- Given: 対象 Application の実効ポリシーは MFA 必須で強制開始済み、enrollment bypass を許可し猶予期限内である
- Given: user は TOTP / WebAuthn factor を持たない
- Given: 管理者が対象 user に有効な単発 enrollment bypass を発行済みである
- Then: user が正しい password を送信する
- Then: bypass は消費され、同一 LoginSession は pending_purpose=Enrollment の未完了状態になる
- Then: MfaEnrollmentRequired と MfaEnrollmentBypassConsumed が発行され、登録専用画面へ進む
- Then: user が TOTP secret に対する正しい code で登録を確定する
- Then: factor が保存され、同一 LoginSession に otp が追加されて pending が解除される
- Then: MfaEnrollmentCompleted と UserAuthenticated が発行され、元の authorization transaction が継続する
- Alternative (enrollment bypass が無い、取消済み、消費済み、または期限切れである): password が正しくてもログインを完了せず access denied にする → factor 登録 API は利用できない
- Alternative (enrollment deadline を過ぎている): factor を保存せず access denied にする → LoginSession を認証完了へ昇格させない
- Alternative (TOTP code が不正である): factor を保存せず InvalidRequestError を返す → LoginSession は Enrollment pending のままである

### REQ-AUTHENTICATION-019: MFA強制開始前の未登録ユーザーはログインできるが登録を促される
- Actor: EndUser
- Given: テナントデフォルトポリシーは将来時刻から MFA 必須になる
- Given: user は MFA factor を持たない
- Then: user が正しい password でログインする
- Then: 強制開始前なので password session は成立する
- Then: UI は強制開始日時と事前登録を促す警告を表示する
- Then: user は通常の step-up を経た account security から factor を事前登録できる

### REQ-AUTHENTICATION-020: Enrollment pendingセッションは通常リソースへアクセスできない
- Actor: EndUser
- Given: pending_purpose=Enrollment の LoginSession が存在する
- Then: user が account、admin、Application の resource を要求する
- Then: システムは未認証として拒否する
- Then: 登録専用 start / confirm API と元の auth transaction だけを許可する

### REQ-AUTHENTICATION-021: 管理者は対象ユーザーのセッションを一覧・個別失効・全失効できる
- Actor: TenantAdministrator
- Given: ユーザー "alice" が複数の有効な LoginSession を持つ
- Then: 管理者がユーザー "alice" の ListSessions を呼ぶ
- Then: 開始時刻の降順で有効なセッション一覧が返る
- Then: 管理者がそのうち1件の RevokeSession を呼ぶ
- Then: 対象セッションは revoke_reason=admin_revoke で失効し "SessionEnded" が発行される
- Then: 管理者がユーザー "alice" の RevokeUserSessions を呼ぶ
- Then: 残り全セッションが失効する
- Alternative (他テナントの管理者が呼び出す): エラー "AccessDeniedError"
- Alternative (既に失効済みのセッションへ再度 RevokeSession を呼ぶ): 204 が返り revoked_at は初回の値を保持する

### REQ-AUTHENTICATION-022: 管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる
- Actor: TenantAdministrator
- Given: ユーザー "alice" は TOTP factor を持ち、recovery code も生成済みである
- Then: 管理者がユーザー "alice" の ResetUserAuthenticators を targets=[Totp, RecoveryCode] で呼ぶ
- Then: "AuthenticatorResetRequested" が発行される
- Then: TOTP factor と recovery code が削除され、他に WebAuthn credential も無いため mfa_enrolled が false になる
- Then: reenrollment_required=true の応答が返り、単発 enrollment bypass が自動発行される
- Then: "AuthenticatorResetCompleted" と "MfaEnrollmentBypassIssued" が発行される
- Then: alice が正しい password で次にログインすると、有効な bypass により同一 LoginSession が pending_purpose=Enrollment になる
- Then: alice が新しい TOTP factor の登録を確定すると、同一 LoginSession が MFA 済みに昇格し元の authorization transaction が継続する
- Alternative (他テナントの管理者、または admin ロールを持たない操作者が呼び出す): エラー "AccessDeniedError" → 対象ユーザーの認証器は変更されない

### REQ-AUTHENTICATION-023: 管理者が一部の認証器のみリセットした場合は残存要素でログインを継続できる
- Actor: TenantAdministrator
- Given: ユーザー "bob" は TOTP factor と WebAuthn credential を両方持つ
- Then: 管理者がユーザー "bob" の ResetUserAuthenticators を targets=[Webauthn] で呼ぶ
- Then: WebAuthn credential のみ削除され、TOTP factor は残るため mfa_enrolled は true のままである
- Then: reenrollment_required=false の応答が返り、enrollment bypass は発行されない
- Then: bob は次回ログインで引き続き TOTP コードによる第二要素検証を経てログインを完了できる

### REQ-AUTHENTICATION-024: GetAccountContext
認証済み browser session の portal 共通 bootstrap 情報と CSRF token を取得する。 Bearer token では管理 portal の idmagic.admin、account portal の idmagic.account、 または自己管理 API client の account:read scope のいずれかを要求する。認証途中の session と、それ以外の scope だけを持つ Bearer token は拒否する。

### REQ-AUTHENTICATION-025: GetPasswordResetContext
未認証のパスワードリセット画面が POST を保護する CSRF token を取得する。

### REQ-AUTHENTICATION-026: StartStepUpWebAuthnChallenge
認証済み session に束縛した step-up 用 WebAuthn assertion challenge を発行する。CSRF を検証し、WebAuthn 無効時は fail closed とする。

### REQ-AUTHENTICATION-027: SubmitBrowserLogin
SPA JSON API から browser password step を処理する。MFA 必須かつ factor 未登録の場合、強制開始前は password session でログインを許可して事前登録を促し、開始後は有効な単発 enrollment bypass と猶予期限の両方を満たすときだけ同一 LoginSession を Enrollment pending にする。それ以外は fail closed でログインを完了しない。per-account (10 回 / 900 秒窓、900 秒ロックアウト) と per-IP (30 回 / 900 秒窓、900 秒ロックアウト) のスロットリングを cluster_wide カウンタで適用し、共有カウンタストア到達不能時も fail_closed とする。この失敗回数ベースの login throttle とは別に、成功/失敗を問わず全リクエストを消費する EndpointRateLimitPolicy (IP+tenant ベース) も適用する。
- Precondition: !context.login_throttled
- Precondition: !context.rate_limited

### REQ-AUTHENTICATION-028: StartBrowserMfaEnrollment
pending_purpose=Enrollment かつ期限内の LoginSession だけが TOTP 登録を開始する。通常の未認証 session、Challenge pending、期限切れ、既登録 user は拒否する。secret は確定まで永続化しない。

### REQ-AUTHENTICATION-029: ConfirmBrowserMfaEnrollment
pending_purpose=Enrollment かつ期限内の LoginSession で TOTP 所持証明を検証して factor を保存し、同一 LoginSession に otp を加えて MFA 済みに昇格させ、元の authorization transaction を継続する。通常 self-service 登録 endpoint と権限を共有しない。

### REQ-AUTHENTICATION-030: SubmitBrowserTotp
SPA JSON API から TOTP コードを検証する。成功すれば LoginSession を認証完了状態に昇格させ、後続フロー (consent / authorization code 発行) に進む。他 factor は別 endpoint として定義する。未認証の pending login session cookie が対象ユーザーを解決するため認証は要求しない。

### REQ-AUTHENTICATION-031: ChangePassword
認証済みユーザーが自身のパスワードを変更する。current_password を verify し、new_password は 12〜128 文字の長さ制約と直近 history_depth (5) 件のハッシュとの不一致 (PasswordHistoryNoReuse) を満たさなければならない。
- Precondition: size(input.request.new_password) >= 12
- Precondition: size(input.request.new_password) <= 128
- Precondition: !passwordHistoryContains(subject.id, input.request.new_password)

### REQ-AUTHENTICATION-032: GetAccountSecurity
認証済みユーザーが自身のセキュリティ概要を取得する (self-service)。対象は subject.id に
固定。パスワード変更日時と MFA (TOTP) 登録状態を返す。

### REQ-AUTHENTICATION-033: StartTotpEnrollment
認証済みユーザーが TOTP 登録を開始する (self-service)。新しい secret (20 バイト) と
otpauth URI を返すだけで永続化はしない。既に TOTP factor がある場合は拒否する。

### REQ-AUTHENTICATION-034: ConfirmTotpEnrollment
secret に対するコードの所持証明を検証し、TOTP factor を永続化する (self-service)。
subject.id に固定し、確定時に User.mfa_enrolled を true にする。

### REQ-AUTHENTICATION-035: RemoveTotpFactor
有効な TOTP コード (所持証明) を検証してから TOTP factor を解除する (self-service)。
subject.id に固定し、解除時に User.mfa_enrolled を false に戻す。

### REQ-AUTHENTICATION-036: StartWebAuthnRegistration
認証済みユーザーが WebAuthn / Passkey 登録を開始する (self-service)。
PublicKeyCredentialCreationOptions を返し、challenge をサーバー側セッションに束縛する。
永続化はまだ行わない。

### REQ-AUTHENTICATION-037: FinishWebAuthnRegistration
attestation を検証して WebAuthn credential を永続化する (self-service)。subject.id に
固定し、最初の 1 件で User.mfa_enrolled を true にする。RP ID / origin / challenge を検証する。

### REQ-AUTHENTICATION-038: RemoveWebAuthnCredential
指定した WebAuthn credential を解除する (self-service, step-up 必須)。subject.id に固定し、
残る第二要素が無くなれば User.mfa_enrolled を false に戻す。

### REQ-AUTHENTICATION-039: StartBrowserWebAuthn
SPA JSON API から WebAuthn assertion challenge を発行する。pending login session が
sub を解決し、PublicKeyCredentialRequestOptions を返す。challenge はそのセッションに束縛する。

### REQ-AUTHENTICATION-040: SubmitBrowserWebAuthn
SPA JSON API から WebAuthn assertion を検証する。成功すれば LoginSession を認証完了
状態に昇格させ、後続フロー (consent / authorization code 発行) に進む。sign_count 逆行は
clone として拒否する (sign_count_regression=reject)。

### REQ-AUTHENTICATION-041: SubmitBrowserRecoveryCode
SPA JSON API から backup recovery code を第二要素として検証する。成功すれば
LoginSession を認証完了状態に昇格させ、消費した code は再利用不可にする。

### REQ-AUTHENTICATION-042: GenerateRecoveryCodes
認証済みユーザーが backup recovery code を生成 / 再生成する (self-service, step-up 必須)。
平文を一度だけ返し、DB には hash のみ保存する。既存 set は全置換する (count=10 件)。

### REQ-AUTHENTICATION-043: RevokeRecoveryCodes
認証済みユーザーが backup recovery code を明示的に失効する (self-service, step-up 必須)。
subject.id に固定し、有効な code をすべて削除する。

### REQ-AUTHENTICATION-044: ListMySignInActivity
認証済みユーザーが自身の直近サインイン履歴を取得する (self-service)。対象は
subject.id に固定。既存の監査イベントストア (UserAuthenticated) から発生時刻の降順で
最大 limit 件を返す。limit 既定 10 / 上限 50。

### REQ-AUTHENTICATION-045: ListUserSignInActivity
admin が対象ユーザーのサインイン履歴を取得する。tenant 境界内に限定し、
パス sub のユーザーの UserAuthenticated を発生時刻の降順で最大 limit 件返す。
limit 既定 10 / 上限 50。

### REQ-AUTHENTICATION-046: ListAuthenticationEventBuckets
admin が認証イベント集約 (bucket) を window_start の降順で一覧する。攻撃時に個別行へ
落とさず 5 分窓へ集約したログイン失敗などを、所属テナント境界内で双方向 keyset pagination により
取得する。cursor は応答の Link response header (rel="prev" / rel="next") から取得する。limit 既定
50 / 上限 200。

### REQ-AUTHENTICATION-047: ListMySessions
認証済みユーザーが自身の有効なセッションを取得する (self-service)。対象は
subject.id に固定。現在のセッションを current=true でマークし、開始時刻の降順で返す。

### REQ-AUTHENTICATION-048: RevokeMySession
認証済みユーザーが自身のセッション 1 件を失効する (self-service)。対象が本人の
ものでなければ拒否する。失効は LoginSession に revoked_at / revoke_reason=self_revoke を
設定する tombstone であり、物理削除しない。既に失効済みの対象への再失効は idempotent に
成功する (revoked_at は初回の値を保持する)。

### REQ-AUTHENTICATION-049: RevokeMyOtherSessions
認証済みユーザーが現在のセッションを除く自身の全セッションを失効する (self-service)。
"他のセッションを全て終了" 操作。各対象は revoked_at / revoke_reason=self_revoke を設定する
tombstone であり、既に失効済みの対象は idempotent にスキップする。

### REQ-AUTHENTICATION-050: ListSessions
admin が対象ユーザーの有効なセッションを一覧する。tenant 境界内に
限定し、開始時刻の降順で返す。ListMySessions と異なり対象は任意の user_id で、
current マーカーは持たない。

### REQ-AUTHENTICATION-051: RevokeSession
admin が対象ユーザーのセッション1件を失効する。RevokeMySession と同じ
tombstone 契約 (revoked_at / revoke_reason=admin_revoke を設定し物理削除しない)。
既に失効済みの対象への再失効は idempotent に成功する。

### REQ-AUTHENTICATION-052: RevokeUserSessions
admin が対象ユーザーの全セッションを失効する。"全セッションを終了"
操作。RevokeMyOtherSessions と異なり操作者自身のセッションではないため除外対象は
無く、対象ユーザーの有効な全セッションを tombstone にする。各対象は idempotent に
失効する。

### REQ-AUTHENTICATION-053: StartStepUpAuthentication
高 sensitivity 操作の前段として step-up 再認証を開始する (self-service)。subject 本人のセッションに固定し、利用可能な factor (password 常時 / totp は
enrolled 時) を返す。本 interface 自体には step-up を要求しない (再認証の入口のため)。

### REQ-AUTHENTICATION-054: CompleteStepUpAuthentication
提示された factor (password または totp コード) を検証し、成立すればセッションに
step_up_at を刻む。以降 recency 窓 (5 分) 内は step_up: required の
操作を許可する。検証失敗は AccessDeniedError。

### REQ-AUTHENTICATION-055: IssueMfaEnrollmentBypass
テナント管理者が同一テナントの MFA 未登録 active user に短期・単発の enrollment bypass を発行する。同一 user の既存未消費 bypass は取り消して置換する。ログイン免除ではなく登録専用 pending flow への承認である。

### REQ-AUTHENTICATION-056: RevokeMfaEnrollmentBypass
テナント管理者が対象 user の未消費 enrollment bypass を取り消す。既に消費・期限切れの場合もログインを許可せず冪等に終了する。

### REQ-AUTHENTICATION-057: ResetUserAuthenticators
テナント管理者が対象 user の認証器 (TOTP factor / WebAuthn credential /
recovery code) から選択した種別を削除し、mfa_enrolled を再計算する (緊急
backstop)。管理者は新しい factor を代わりに登録できず、削除と再登録要求に限定する。
TOTP と WebAuthn が両方無くなった場合は既存の管理者承認 enrollment bypass
(IssueMfaEnrollmentBypass と同じ機構) を自動発行して置換し、次回ログインを
fail-closed な enrollment-required flow へ誘導する。片方の要素のみ削除するなど mfa_enrolled
が true のまま残る場合は bypass を発行せず、残存要素で通常ログインを継続できる。

### REQ-AUTHENTICATION-058: RequestPasswordReset
未認証ユーザーがパスワードリセットリンクの送信を要求する。
anti-enumeration のため、email の存在有無に関わらず常に 204 を返し、
verified email を持つ user が見つかった場合のみ token を発行して送信する。
文面は Tenancy の通知テンプレートカタログ (template_key=PasswordReset) から
解決する。locale は受信者 User の locale 属性 → テナントの default_locale →
システム既定 locale の順に決め、テナント上書きがあれば組込み既定より優先する。
プレーンテキストと HTML の両方を必ず生成し、リセットリンクは
リクエストの発行元 URL から組み立てる (テンプレート側に URL 結合をさせない)。
- Precondition: !context.rate_limited

### REQ-AUTHENTICATION-059: ResetPasswordWithToken
reset リンクから戻ってきたユーザーが新パスワードを設定する。
token を単発消費し (期限 1800 秒、consumed_at 設定済みは再利用不可)、12〜128 文字の
長さ制約と直近 history_depth (5) 件のハッシュとの不一致 (PasswordHistoryNoReuse) を
通過した new_password で User.password_hash を更新する。成功時 PasswordChanged を emit。
- Precondition: size(input.request.new_password) >= 12
- Precondition: size(input.request.new_password) <= 128
- Precondition: !passwordResetTokenConsumed(input.request.token)
- Precondition: !passwordHistoryContainsForToken(input.request.token, input.request.new_password)

### REQ-AUTHENTICATION-060: DiscoverIdentityProviders
login UI が request tenant で Active な upstream provider の表示用情報だけを取得する。secret、endpoint、certificate、link policy は公開しない。optional email/domain hint は管理時に検証済み routing rule だけに適用する。

### REQ-AUTHENTICATION-061: StartFederatedLogin
Active connection を tenant 内で解決し、OIDC state / nonce / PKCE または SAML RelayState / AuthnRequest ID を単発 FederatedLoginAttempt に保存して、管理時検証済み upstream endpoint への redirect を返す。任意 URL は受け取らない。

### REQ-AUTHENTICATION-062: CompleteFederatedLogin
callback を single-use login attempt と相関し、OIDC token または SAML response の trust、issuer、audience、nonce/InResponseTo、time、replay を検証する。既存 link、明示 policy 下の verified-email link、明示 JIT の順で local User を解決し、LoginSession 発行へ渡す。

### REQ-AUTHENTICATION-063: ListLinkedExternalIdentities
ResourceOwner が自身に結び付く external identity の provider 表示名、protocol、linked_at、last_login_at を取得する。external subject と token は返さない。

### REQ-AUTHENTICATION-064: LinkExternalIdentity
直近5分以内に step-up 済みの ResourceOwner が provider の外部認証を開始し、callback で得た subject を自身へ明示 link する。既に他 user に link 済みの subject は付け替えない。

### REQ-AUTHENTICATION-065: UnlinkExternalIdentity
直近5分以内に step-up 済みの ResourceOwner が自身の provider link を解除する。利用可能な password credential も他の link も無い場合は account lockout 防止のため拒否する。

### REQ-AUTHENTICATION-066: ListIdentityProviderConnections
tenant administrator が connection 一覧を取得する。secret_reference と secret は返さない。

### REQ-AUTHENTICATION-067: CreateIdentityProviderConnection
tenant administrator が connection を作る。作成直後の初期状態は Disabled。OIDC issuer または SAML metadata を管理時に検証し last-known-good trust material を保存する。client secret は secret_reference で受け取り、書き込み専用のエンベロープ暗号化保存にまわす。

### REQ-AUTHENTICATION-068: UpdateIdentityProviderConnection
tenant administrator が connection の display、mapping、policy、secret reference、trust source を更新する。issuer / endpoint / certificate / protocol などの trust source を変更したときだけ Active を Disabled に落とし、display_name や claim_mapping など非 trust フィールドの変更では状態を保持する。trust source 変更時に再検証が失敗した場合は last-known-good を維持する。

### REQ-AUTHENTICATION-069: DeleteIdentityProviderConnection
tenant administrator が connection を状態を問わずいつでも削除する。link が残る場合は削除を拒否し、先に明示 cleanup を要求する。

### REQ-AUTHENTICATION-070: ActivateIdentityProviderConnection
trust validation と接続 test に成功した Disabled connection を Active にする。

### REQ-AUTHENTICATION-071: DisableIdentityProviderConnection
Active connection を Disabled にし、新規 routing と callback completion を直ちに拒否する。

### REQ-AUTHENTICATION-072: RefreshIdentityProviderMetadata
tenant administrator が固定済み issuer / metadata source から discovery/JWKS または SAML metadata を再取得し、SSRF と trust validation 成功時だけ last-known-good を更新する。

### REQ-AUTHENTICATION-073: TestIdentityProviderConnection
保存済み last-known-good trust source に対して実際に到達性・解決可能性を検査する。OIDC は authorization_endpoint/token_endpoint/jwks_uri への到達性と secret_reference の解決可能性を、SAML は保存済み証明書の X.509 パース可否と有効期限を確認する。secret の実値、token、certificate 本文は返さない。

### REQ-AUTHENTICATION-074: PreviewIdentityProviderMapping
tenant administrator が保存済み test response の検証済み claim を mapping し、token/assertionを含まない正規化 preview を取得する。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| IdentityBroker | 外部 identity provider の認証結果を検証し、tenant 内の local User と安全に相関して LoginSession 発行へ渡す Authentication capability。 |  |
| ExternalIdentityProvider | idmagic に対して upstream authentication authority となる OIDC Provider または SAML Identity Provider。 | upstream IdP, social login provider |
| FederatedIdentity | tenant、provider、外部の不変 subject の組を local User へ一意に結び付ける identity link。 |  |
| JitProvisioning | 検証済み外部 claim と tenant の明示 policy / claim mapping に基づき、初回 federated login 中に local User を作成すること。 | JIT provisioning |
| Totp | RFC 6238 に基づく time-based one-time password。 | totp, otp |
| Webauthn | WebAuthn credential による認証。 | webauthn |
| RecoveryCode | TOTP / WebAuthn 喪失時に使う backup の使い捨て復旧コード。 | recovery_code |
| EndUser | 認証済みまたは認証を試みる一般利用者。ログイン・MFA継続・パスワードリセットなど、認証が未完了の操作の主体を指す。 |  |
| ResourceOwner | OAuth2/OIDC 認可フローでリソースの所有者として認可判断を行う利用者。EndUser と同一人物を OAuth2 文脈で指す呼称。 |  |

## Standards

### OpenID Connect Core 1.0

1.0 incorporating errata set 2 — https://openid.net/specs/openid-connect-core-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-CORE-CODE-FLOW | required | MUST | 外部 OIDC 認証は authorization code flow を使い、ID Token の署名、issuer、audience、有効時間、nonce を検証する。 |
| OIDC-CORE-CSRF | required | SHOULD | callback は login attempt に束縛された単発 state を照合し、不一致または再利用を拒否する。 |

### OpenID Connect Discovery 1.0

1.0 incorporating errata set 2 — https://openid.net/specs/openid-connect-discovery-1_0.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| OIDC-DISCOVERY-ISSUER | required | MUST | discovery 文書の issuer は設定した issuer と完全一致し、endpoint と JWKS URI は事前に許可された HTTPS authority に限定する。 |

### TOTP Time-Based One-Time Password Algorithm

RFC 6238 — https://www.rfc-editor.org/rfc/rfc6238.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC6238-TOTP | optional | MUST | TOTP factor利用時は共有秘密と時間ステップからOTPを生成・検証する。 |

### Digital Identity Guidelines — Authentication and Authenticator Management

NIST SP 800-63B-4 — https://pages.nist.gov/800-63-4/sp800-63b.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| NIST63B4-PASSWORD-MINIMUM | excluded | MUST | 単一要素認証に使用するPasswordへ15文字以上の最小長を要求する。 |
| NIST63B4-NO-COMPOSITION | required | MUST NOT | 文字種混在などPassword composition ruleを課さない。 |
| NIST63B4-PASSWORD-STORAGE | required | MUST | Passwordをsaltとcost factorを持つoffline attack耐性のあるhashとして保存する。 |

### Web Authentication — An API for accessing Public Key Credentials Level 3

Candidate Recommendation Snapshot — https://www.w3.org/TR/webauthn-3/

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| WEBAUTHN3-AUTHENTICATION | required | MAY | WebAuthn factor利用時はoriginとRelying Partyにscopeされた公開鍵Credentialを検証する。 |
| WEBAUTHN3-REGISTRATION | required | MUST | WebAuthn credential登録時はattestationのchallenge / RP ID / originを検証し、COSE公開鍵とsign countを保存する。 |

### Authentication Method Reference Values

RFC 8176 — https://www.rfc-editor.org/rfc/rfc8176.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC8176-AMR-VOCABULARY | required | MUST | LoginSession.amr は RFC 8176 登録値 (pwd, otp, webauthn, hwk, swk) のサブセットに、本アプリ固有の非 IANA 拡張値 rc (recovery code) を加えた語彙のみを許可する。 |

## State machines

### IdentityProviderConnectionLifecycle

upstream connection は利用可能 Active と routing 停止 Disabled の2状態だけを遷移する。作成直後の初期状態は Disabled。metadata refresh 失敗や trust source 以外のフィールド更新は状態を変えず last-known-good を保持する。

Initial: `Disabled`  
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | IdentityProviderConnectionDisabled | "" | Disabled |  |
| Disabled | IdentityProviderConnectionActivated | "" | Active |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
