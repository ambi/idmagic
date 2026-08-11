namespace IdMagic.Contract {

@doc("EmailSender port 経由でメールを送信した。fail-open の port なので\n送信失敗時も use case 層には伝播しないが、本イベントの有無で\nSIEM 側からは観測できる。to は SHA-256 hash で記録する。")
model EmailSent {
  occurredAt: utcDateTime;
  @doc("宛先 email の SHA-256 hex")
  toHash: string;
  @doc("password_reset / email_change / email_verification など。")
  purpose: string;
  delivered: boolean;
}


@doc("upstream connection の wire protocol。social provider preset は Oidc の設定初期値として表す。")
enum IdentityProviderProtocol {
  Oidc: "Oidc",
  Saml: "Saml",
}


@doc("connection の lifecycle 状態。Active の connection だけを discovery と login routing に使用できる。作成直後の初期状態は Disabled。")
enum IdentityProviderConnectionStatus {
  Active: "Active",
  Disabled: "Disabled",
}


@doc("外部 identity と local User の相関方針。既定 None は既存 link 以外を自動相関しない。VerifiedEmail は tenant が明示した場合だけ verified email の一意一致を許可する。")
enum AccountLinkingPolicy {
  None: "None",
  VerifiedEmail: "VerifiedEmail",
}


@doc("検証済み upstream claim / attribute を local identity field へ写像する tenant 管理設定。JIT provisioning には subject と username の mapping が必須。")
model ClaimMapping {
  subject: string;
  username: string;
  email?: string;
  email_verified?: string;
  name?: string;
}


@doc("tenant-scoped upstream trust 設定。request 中に入力された任意 URL は取得せず、管理時に検証した metadata の last-known-good endpoint / key / certificate だけを使う。client secret は write-only の secret_reference で受け取り、DataKeys の per-tenant DEK でエンベロープ暗号化して保持する。")
model IdentityProviderConnection {
  id: string;
  tenant_id: string;
  display_name: string;
  protocol: IdentityProviderProtocol;
  status: IdentityProviderConnectionStatus;
  issuer: string;
  client_id?: string;
  @doc("client secret の実値を書き込み専用で受け取るフィールド。DB には DataKeys の per-tenant DEK でエンベロープ暗号化した ciphertext のみを保存し、API レスポンスには実値・ciphertext のいずれも含めない。フィールド名は移行元の env: 参照方式から維持している。")
  secret_reference?: string;
  @doc("secret_reference が設定済みかどうかを示す読み取り専用の派生フィールド。実値・ciphertext を含む書き込みには使わない。管理画面の詳細ビューが「設定済み/未設定」だけを示すために使う。")
  client_secret_configured: boolean;
  authorization_endpoint?: string;
  token_endpoint?: string;
  jwks_uri?: string;
  saml_sso_url?: string;
  saml_entity_id?: string;
  saml_signing_certificates: string[];
  claim_mapping: ClaimMapping;
  linking_policy: AccountLinkingPolicy;
  jit_provisioning: boolean;
  allowed_email_domains: string[];
  metadata_refreshed_at?: utcDateTime;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("接続テストの構造化結果。OIDC discovery/JWKS 到達性、SAML 証明書のパース可否、secret_reference の解決 (復号) 可能性を検証する。値そのもの (secret、token、証明書本文) は含まない。")
model IdentityProviderConnectionTestResult {
  success: boolean;
  failures: string[];
}


@doc("外部の不変 subject を local User に相関する link。同一 tenant/provider/subject は最大1 local user、同一 local user/provider も最大1 subject とする。")
model FederatedIdentity {
  tenant_id: string;
  provider_id: string;
  external_subject: string;
  local_user_id: string;
  linked_at: utcDateTime;
  last_login_at?: utcDateTime;
}


@doc("upstream browser round-trip を相関する単発 ephemeral record。state / nonce / PKCE verifier または RelayState / AuthnRequest ID を tenant と provider に束縛し、10分以内に一度だけ消費する。")
model FederatedLoginAttempt {
  state: string;
  tenant_id: string;
  provider_id: string;
  protocol: IdentityProviderProtocol;
  nonce?: string;
  pkce_verifier?: string;
  request_id?: string;
  return_to?: string;
  created_at: utcDateTime;
  expires_at: utcDateTime;
  consumed_at?: utcDateTime;
}


@doc("login UI に公開してよい active upstream provider。secret、内部 endpoint、certificate は含まない。")
model ExternalIdentityProviderSummary {
  id: string;
  display_name: string;
  protocol: IdentityProviderProtocol;
}


@doc("upstream 認証を開始するための検証済み redirect。")
model FederatedLoginStartResponse {
  redirect_to: string;
}


@doc("検証済み外部 identity を local User と LoginSession に相関した結果。")
model FederatedLoginCompletion {
  local_user_id: string;
  provider_id: string;
  session_id: string;
  return_to?: string;
}


@doc("管理者が mapping を確認するための正規化結果。token、assertion、secret は返さない。")
model ExternalIdentityClaimsPreview {
  subject: string;
  username: string;
  email?: string;
  email_verified: boolean;
  name?: string;
}


@doc("新しいパスワードが直近のパスワード履歴と一致する。")
model PasswordReuseError {

}


@doc("対象ユーザーが MFA enrollment bypass の対象条件を満たさない。")
model MfaEnrollmentNotAllowedError {

}


@doc("指定した reset target の組み合わせが対象ユーザーの現在の MFA 状態に対して無効である。")
model AuthenticatorResetNotAllowedError {

}


@doc("認証済み browser session の管理 shell bootstrap 情報。CSRF token と subject、現在 realm、effective role を返す。")
model AccountContextResponse {
  csrf_token: string;
  id: string;
  preferred_username?: string;
  tenant_id?: string;
  realm?: string;
  roles?: string[];
}


@doc("未認証 browser form が SameSite cookie と対で使う CSRF token bootstrap 情報。")
model BrowserCsrfContext {
  csrf_token: string;
}


@doc("第二要素 (MFA) の種別 (RFC 8176 amr 値の本アプリ向けサブセット)。Totp は RFC 6238、Webauthn は WebAuthn Level 3、Hwk / Swk は将来拡張のための予約。")
enum MfaFactorType {
  Totp: "Totp",
  Webauthn: "Webauthn",
  Hwk: "Hwk",
  Swk: "Swk",
}


@doc("ユーザーが登録した第二要素 (MFA) 1 件。Totp の場合 secret は base32 で保存し、認証時に RFC 6238 (T0=0, STEP=30s, digits 桁, window ±1 step) で評価する。algorithm は SHA1、secret は 20 バイトで生成する。User.mfa_enrolled の真値はこの集合の非空性と同義。")
model MfaFactor {
  @doc("所有者ユーザの sub。")
  user_id: string;
  type: MfaFactorType;
  @doc("Totp の場合は base32 シークレット。他要素は credential JSON 等を別フィールド化する余地として残す。DB 永続化時は DataKeys の per-tenant DEK でエンベロープ暗号化し、平文を保存しない。")
  secret?: string;
  @doc("ユーザが付ける表示名 (例 \"iPhone Authenticator\")。")
  label?: string;
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("LoginSession が未完了である理由。None は通常の認証済みセッション、Challenge は登録済み factor の検証待ち、Enrollment は管理者承認済みの新規 factor 登録待ち。")
enum LoginPendingPurpose {
  None: "None",
  Challenge: "Challenge",
  Enrollment: "Enrollment",
}


@doc("管理者が MFA 未登録 user に発行する短期・単発の登録承認。password 成功後に消費して enrollment pending へ進めるための trust anchor であり、MFA 必須ログインそのものを免除しない。期限切れ・取消・消費済みは fail closed で利用できない。")
model MfaEnrollmentBypass {
  id: unknown;
  tenant_id: string;
  user_id: string;
  issued_by: string;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  consumed_at?: utcDateTime;
  revoked_at?: utcDateTime;
  expired_at?: utcDateTime;
}


@doc("管理者による認証器リセットが対象にできる要素。Totp / Webauthn は MfaFactor / WebAuthnCredential の全件削除、RecoveryCode は backup recovery code 一式の削除を指す。")
enum AuthenticatorResetTarget {
  Totp: "Totp",
  Webauthn: "Webauthn",
  RecoveryCode: "RecoveryCode",
}


@doc("管理者が対象 user の認証器リセットで指定する削除対象の集合。")
model AuthenticatorResetRequest {
  targets: AuthenticatorResetTarget[];
}


@doc("認証器リセットの結果。reenrollment_required は TOTP / WebAuthn の両方が無くなり、次回ログインで登録専用 pending flow へ入る単発 bypass を自動発行したことを示す。bypass は reenrollment_required の場合だけ存在する。")
model AuthenticatorResetResponse {
  mfa_enrolled: boolean;
  reenrollment_required: boolean;
  bypass?: MfaEnrollmentBypass;
}


@doc("ブラウザログインセッション。PostgreSQL を単一正本として永続化し、\nプロセス再起動を跨いで有効な session を維持する。Cookie には不透明な\nsession id のみを置き、sub と auth_time はサーバー側ストアに保持する。amr / acr も\nここに残し、後続の /authorize で step-up 判定に使う。authentication_pending=true は\nサインインポリシーまたは step-up が追加 factor を要求した場合だけ同一 LoginSession を\n遷移させる中間状態で、MFA factor 登録済みという事実だけでは pending にしない。追加\nfactor 待ちへ進むときに別の完了済み pwd セッションを残してはならない。/authorize\n完了経路は pending セッションを未認証扱いとし対応する factor challenge に誘導する。\nCookie は http_only かつ same_site=Lax で発行する。tenant_id は認証解決クエリに\n必ず含め、session id が tenant をまたいで推測・再利用されても別 tenant の行を返さない\nfail-closed な分離を DB 層で担保する。失効は revoked_at / revoke_reason を設定する\ntombstone であり、行を物理削除しない (再失効は idempotent に成功する)。認証解決は\nrevoked_at IS NULL AND expires_at > now の行だけを有効とみなす。")
model LoginSession {
  id: unknown;
  tenant_id: string;
  user_id: string;
  @doc("認証が成立した Unix 秒。session の開始時刻として不変であり、一覧の並び替え (開始時刻の降順) にも使う。")
  auth_time: int64;
  @doc("認証成立時の RFC 8176 メソッド列 (RFC8176-AMR-VOCABULARY)。")
  amr: string[];
  @doc("認証コンテキストクラス参照 URN。urn:idmagic:acr:pwd (amr ⊇ [pwd] のみ) または urn:idmagic:acr:mfa (amr が otp / webauthn / hwk / swk / rc のいずれかを含む) のいずれか (OIDC Core §2 / RFC 8176)。")
  acr: string;
  @doc("追加 factor 検証を待っている中間状態か。完了で false。")
  authentication_pending: boolean = false;
  @doc("authentication_pending の目的。Enrollment の間は account/admin/application resource へアクセスできず、登録専用 API だけを許可する。")
  pending_purpose: LoginPendingPurpose = LoginPendingPurpose.None;
  @doc("pending_purpose=Enrollment の完了期限。期限後の登録確定は拒否し、セッションを認証完了へ昇格させない。")
  enrollment_deadline?: utcDateTime;
  @doc("Enrollment へ進むために消費した管理者発行 bypass。監査相関用。")
  enrollment_bypass_id?: unknown;
  @doc("直近で password / MFA による step-up 再認証が成立した Unix 秒 (未実施は 0)。高 sensitivity な self-service 操作の recency 判定に使う。")
  step_up_at: int64 = 0;
  expires_at: utcDateTime;
  @doc("セッションが認証結果として意味上利用された直近時刻。認証解決のたびに touch するが、直近の touch から LoginSessionTouchInterval 未満の再 touch は永続化しない粗粒度な値であり、idle timeout 判定はこの粒度で近似する。")
  last_seen_at: utcDateTime;
  @doc("失効時刻。未失効は null。失効は tombstone であり物理削除しない。")
  revoked_at?: utcDateTime;
  @doc("revoked_at と対で設定される失効理由。revoked_at が null なら null。")
  revoke_reason?: SessionEndReason;
}


@doc("Authentication bounded context が OAuth2/OIDC bounded context に渡す認証結果。OAuth2/OIDC は password、UserRepository、session cookie の詳細を参照しない。")
model AuthenticationContext {
  user_id: string;
  auth_time: int64;
  amr: string[];
  acr?: string;
  session_id?: string;
}


@doc("self-service セキュリティ画面で提示する MFA factor 1 件の射影。secret は\n返さず、種別・ラベル・作成/最終利用時刻のみ。現状 TOTP のみ。")
model AccountMfaFactor {
  type: MfaFactorType;
  label?: string;
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("`/api/account/security` の返り値 (end-user account portal のセキュリティ概要)。\nパスワード変更日時と MFA (TOTP / WebAuthn) の登録状態、backup recovery code の残数を\nまとめる。")
model AccountSecurityResponse {
  password_changed_at?: utcDateTime;
  totp_enrolled: boolean;
  factors: AccountMfaFactor[];
  webauthn_credentials: WebAuthnCredentialSummary[];
  recovery_codes: RecoveryCodeStatus;
}


@doc("TOTP 登録開始の返り値 (self-service)。secret はクライアントが確認まで保持し、\notpauth_uri は QR / 手動登録のために提示する。secret は確認 (ConfirmTotpEnrollment) で\n送り返されるまで永続化しない。")
model TOTPEnrollmentStart {
  secret: string;
  otpauth_uri: string;
  account_name: string;
  issuer: string;
}


@doc("管理者承認済みの login enrollment flow で TOTP 登録を確定する。secret と code は audit / log に残さない。")
model BrowserMfaEnrollmentConfirmRequest {
  secret: string;
  code: string;
  return_to?: string;
}


@doc("管理者が対象 user に単発 enrollment bypass を発行する入力。")
model MfaEnrollmentBypassIssueRequest {
  expires_in_seconds: int64 = 900;
}


@doc("enrollment bypass の管理表示。secret は存在せず、対象 user の password 成功時にサーバーが自動消費する。")
model MfaEnrollmentBypassResponse {
  bypass: MfaEnrollmentBypass;
}


@doc("TOTP 登録確認リクエスト (self-service)。secret は開始時の値、code はその secret\nから生成した現在のコード (所持証明)。")
model TOTPEnrollmentConfirmRequest {
  @minLength(1)
  secret: string;
  @minLength(1)
  code: string;
}


@doc("MFA factor 解除リクエスト (self-service)。code は所持証明としての現在の TOTP\nコード。解除は本人 + 所持証明を要求する。")
model MfaFactorRemoveRequest {
  @minLength(1)
  code: string;
}


@doc("WebAuthn authenticator の接続方式のヒント (WebAuthn Level 3 §5.8.4 AuthenticatorTransport)。")
enum WebAuthnTransport {
  Usb: "Usb",
  Nfc: "Nfc",
  Ble: "Ble",
  Internal: "Internal",
  Hybrid: "Hybrid",
}


@doc("ユーザーが登録した WebAuthn / Passkey credential 1 件。1 ユーザーが複数持てるため\nMfaFactor とは別集合とし、credential_id で一意識別する。public_key は COSE 公開鍵、\nsign_count は clone 検出のための署名カウンタ。origin / RP に束縛された所持要素として\nphishing-resistant MFA / step-up に使う。ceremony は RP ID / origin を deployment config\nから解決し、user_verification=preferred・attestation=none (enterprise attestation の厳格\nenforcement は行わない)・resident_key=discouraged (passwordless は対象外)・challenge は 32\nバイト・timeout は 120 秒とする (WebAuthn Level 3 §5 / §7 / §13)。")
model WebAuthnCredential {
  @doc("authenticator が返す credential ID (base64url)。RP 内で一意。")
  credential_id: string;
  @doc("所有者ユーザの sub。")
  user_id: string;
  @doc("COSE 形式の公開鍵 (base64url)。")
  public_key: string;
  @doc("直近 assertion の署名カウンタ。逆行は clone の疑いとして拒否する (sign_count_regression=reject)。")
  sign_count: int64;
  @doc("authenticator が申告した transport のヒント (usb / nfc / ble / internal / hybrid)。")
  transports: string[];
  @doc("authenticator の AAGUID (base64url)。機種識別のヒント。")
  aaguid?: string;
  @doc("ユーザが付ける表示名 (例 \"MacBook Touch ID\")。")
  label?: string;
  @doc("credential が複数デバイス間で同期可能か (WebAuthn BE flag)。")
  backup_eligible: boolean;
  @doc("credential が現在 backup 済みか (WebAuthn BS flag)。")
  backup_state: boolean;
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("self-service セキュリティ画面で提示する WebAuthn credential 1 件の射影。\n公開鍵や sign_count は返さず、識別と表示に必要な属性のみ。")
model WebAuthnCredentialSummary {
  credential_id: string;
  label?: string;
  transports: string[];
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("TOTP / WebAuthn 喪失時の backup recovery code 1 件。平文は生成時に一度だけ表示し、\nDB には code_hash (SHA-256 hex) のみ保存する。consumed_at が非 null なら使用済みで\n再利用不可。1 ユーザーの有効な code 集合を 1 set として扱い、再生成で全置換する\n(regenerate_replaces_all)。生成時は count=10 件、各 code_length=10 文字、\nalphabet 23456789abcdefghijkmnpqrstuvwxyz (混同しやすい文字を除いた base32 風) から\n採番し、単独では第二要素にならない backup 専用とする (NIST SP 800-63B §5.1.2 Look-up Secrets)。")
model RecoveryCode {
  user_id: string;
  @doc("recovery code 平文の SHA-256 hex。平文は保存しない。")
  code_hash: string;
  generated_at: utcDateTime;
  consumed_at?: utcDateTime;
}


@doc("self-service セキュリティ画面で提示する recovery code の状態。個々のコードは返さず、\n生成時刻と総数・残数のみ。")
model RecoveryCodeStatus {
  generated_at?: utcDateTime;
  total: int64;
  remaining: int64;
}


@doc("StartWebAuthnRegistration の返り値。navigator.credentials.create に渡す\nPublicKeyCredentialCreationOptions を JSON でそのまま返す。challenge は短命で\nサーバー側セッションに束縛する。")
model WebAuthnRegistrationOptions {
  @doc("PublicKeyCredentialCreationOptions (WebAuthn Level 3)。")
  public_key: unknown;
}


@doc("FinishWebAuthnRegistration の入力。navigator.credentials.create が返した\nattestation を JSON で送り返し、任意で表示ラベルを付ける。")
model WebAuthnRegistrationRequest {
  @doc("PublicKeyCredential (attestation response, WebAuthn Level 3)。")
  attestation: unknown;
  label?: string;
}


@doc("StartBrowserWebAuthn の返り値。navigator.credentials.get に渡す\nPublicKeyCredentialRequestOptions を JSON でそのまま返す。challenge は pending login\nsession に束縛する。")
model WebAuthnAssertionOptions {
  @doc("PublicKeyCredentialRequestOptions (WebAuthn Level 3)。")
  public_key: unknown;
}


@doc("SubmitBrowserWebAuthn の入力。navigator.credentials.get が返した assertion を\nJSON で送り返す。pending login session cookie が sub を、X-CSRF-Token が CSRF を解決する。")
model BrowserWebAuthnRequest {
  @doc("PublicKeyCredential (assertion response, WebAuthn Level 3)。")
  assertion: unknown;
  return_to?: string;
}


@doc("SubmitBrowserRecoveryCode の入力。第二要素として recovery code を 1 つ消費する。\ncode は audit / log に残さない。")
model BrowserRecoveryCodeRequest {
  code: string;
  return_to?: string;
}


@doc("RemoveWebAuthnCredential の入力。解除対象の credential を credential_id で指定する。")
model WebAuthnCredentialRemoveRequest {
  @minLength(1)
  credential_id: string;
}


@doc("GenerateRecoveryCodes の返り値。平文の recovery code 一覧を一度だけ返す。\n以降は残数 (RecoveryCodeStatus) のみ取得でき、codes は保存・再表示しない。")
model RecoveryCodesResponse {
  codes: string[];
  generated_at: utcDateTime;
}


@doc("ユーザー本人のサインイン履歴 1 件の射影。本スライスは既存の監査\nイベントストア (UserAuthenticated) から導出するため、保持する属性は発生時刻と\n認証手段 (amr) のみ。IP / UA / sessionId / clientId の付加は後続で\nイベント側を拡張してから足す。")
model UserSignInActivity {
  occurred_at: utcDateTime;
  @doc("RFC 8176 の技術値 (RFC8176-AMR-VOCABULARY)。UI は pwd=パスワード、otp=認証アプリ、webauthn=パスキー、rc=リカバリコードなど、利用者向けの認証方式名に変換して表示する。")
  amr: string[];
}


@doc("ListMySignInActivity / ListUserSignInActivity の返り値。発生時刻の降順 (新しい順) で\n直近 N 件を返す。")
model UserSignInActivityResponse {
  activities: UserSignInActivity[];
}


@doc("LoginSession 終了の理由。self_revoke / admin_revoke は本人 / admin の明示的な\n失効、logout / idle / absolute は自動失効、password_change / mfa_change は資格情報変更に\n伴う失効。")
enum SessionEndReason {
  logout: "logout",
  idle: "idle",
  absolute: "absolute",
  self_revoke: "self_revoke",
  admin_revoke: "admin_revoke",
  password_change: "password_change",
  mfa_change: "mfa_change",
  other: "other",
}


@doc("self-service セッション一覧の 1 件の射影。secret は持たず、識別子と\n認証情報のみ。current は呼び出し元の現在のセッションかどうか。started_at は LoginSession の\nauth_time、expires_at は失効予定時刻。")
model AccountSession {
  id: string;
  current: boolean;
  amr: string[];
  acr: string;
  started_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("ListMySessions の返り値。開始時刻の降順 (新しい順) で本人の有効なセッションを返す。")
model AccountSessionListResponse {
  sessions: AccountSession[];
}


@doc("admin 向けセッション一覧の1件の射影。secret は持たず、対象ユーザーの\n有効なセッションを識別・監査するための属性のみ。self-service の AccountSession と\n異なり current マーカーを持たない (操作者と対象ユーザーが別人のため)。")
model SessionRecord {
  id: string;
  user_id: string;
  amr: string[];
  acr: string;
  started_at: utcDateTime;
  last_seen_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("ListSessions (admin) の返り値。開始時刻の降順 (新しい順) で対象ユーザーの\n有効なセッションを返す。")
model SessionRecordListResponse {
  sessions: SessionRecord[];
}


@doc("認証イベント集約 (bucket) 1 件の admin 表示用射影。攻撃時に個別の\nログイン失敗を 5 分窓へ集約した結果で、window_start は窓の開始時刻、count はその窓の\n累計件数、first_seen / last_seen は窓内の初出・最終時刻。key_hash は SHA-256 hex で\n平文を含まない。tenant 境界内に閉じる。")
model AuthenticationEventBucket {
  @doc("failed_login | throttled | mfa_failed")
  kind: string;
  key_hash: string;
  window_start: utcDateTime;
  count: int64;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  first_seen: utcDateTime;
  last_seen: utcDateTime;
}


@doc("ListAuthenticationEventBuckets の返り値。window_start の降順 (新しい窓が先) で返す。")
model AuthenticationEventBucketListResponse {
  buckets: AuthenticationEventBucket[];
}


@doc("step-up 再認証に使える factor。password は常時、totp / webauthn は enrolled 時のみ、\nrecovery_code は有効な残数がある時のみ利用できる。")
enum StepUpMethod {
  password: "password",
  totp: "totp",
  webauthn: "webauthn",
  recovery_code: "recovery_code",
}


@doc("StartStepUpAuthentication の返り値。利用可能な再認証 method の一覧 (UI が再認証\nmodal に出す選択肢)。")
model StepUpStartResponse {
  methods: StepUpMethod[];
}


@doc("CompleteStepUpAuthentication の入力。method に応じて password または code を使う。\n検証材料は audit / log に残さない。")
model StepUpCompleteRequest {
  method: StepUpMethod;
  password?: string;
  code?: string;
}


@doc("Browser client が POST /api/auth/totp に送る TOTP コード。authorization request id は HttpOnly transaction cookie、CSRF は X-CSRF-Token header から解決する。他 factor (WebAuthn 等) はHTTP形が大きく異なるため、別endpointと別value objectとして独立に定義する。")
model BrowserTotpRequest {
  @doc("Authenticator アプリが生成した数字コード (6 桁、RFC 6238 §4 / RFC 4226 §4)。")
  code: string;
  @doc("管理画面への直接ログインを継続する場合の同一 realm 内遷移先。")
  return_to?: string;
}


@doc("SPA が POST /api/auth/change_password に送る JSON。認証済みセッション cookie が sub を解決し、CSRF は X-CSRF-Token header で検証する。")
model BrowserChangePasswordRequest {
  current_password: string;
  @minLength(12)
  @maxLength(128)
  new_password: string;
}


@doc("SPA が POST /api/auth/forgot_password に送る JSON。未認証経路。anti-enumeration のため email 存在有無に関わらず常に 204 を返す。")
model BrowserForgotPasswordRequest {
  email: string;
}


@doc("SPA が POST /api/auth/reset_password に送る JSON。email リンクに含まれた単発消費 token と新パスワードを受ける。")
model BrowserResetPasswordRequest {
  token: string;
  @minLength(12)
  @maxLength(128)
  new_password: string;
}


@doc("パスワードリセット用の単発消費トークン。\ntoken は base64url の 32 バイト乱数で、保存時は SHA-256 hash のみ。\n有効期間は発行から 1800 秒 (30 分) とし、単発消費 (consumed_at 設定後は再利用不可)。\n期限切れ / consume 済みの行は GC するが、INSERT 時に同 sub の未消費を\n失効させる軽い掃除も行う。")
model PasswordResetTokenRecord {
  user_id: string;
  token_hash: string;
  created_at: utcDateTime;
  @doc("created_at + 1800 秒 (30 分)。")
  expires_at: utcDateTime;
  @doc("非 null なら使用済みで再利用不可。")
  consumed_at?: utcDateTime;
}


@doc("ResourceOwner の認証が成功した。amr は実際に経由した認証手段の一覧 (OIDC Core §2)。")
model UserAuthenticated {
  occurredAt: utcDateTime;
  @doc("認証が起きたテナント ID。監査ビューのテナント絞り込みに使う。")
  tenantId: string;
  userId: string;
  amr?: string[];
  @doc("認証で開始した LoginSession の ID。")
  sessionId?: string;
  @doc("認証を起動した OAuth クライアント。")
  clientId?: string;
  acr?: string;
  @doc("接続元 IP。監査イベントの保持期間中は平文のまま保存する。")
  ip?: string;
  @doc("接続元 User-Agent。監査イベントの保持期間中は平文のまま保存する。")
  userAgent?: string;
  @doc("GeoIP 由来の国コード (未連携時は空)。")
  countryCode?: string;
  @doc("device fingerprint。監査イベントの保持期間中は平文のまま保存する。")
  deviceFingerprint?: string;
  @doc("リスクスコア (本 WI ではフィールド確保のみ、値は null)。")
  riskScore?: int64;
}


@doc("ResourceOwner の認証が失敗した。レートリミット・不正検知の入力イベント。\n実在しないユーザー名での試行も記録対象のため sub を持たず、username を平文のまま監査検索の\nactor.username 相関キーとして保持する。username を含む接続元属性は hash 化せず、他の failure\nイベントと同じ保持期間 (既定 30 日) で扱う。")
model AuthenticationFailed {
  occurredAt: utcDateTime;
  @doc("認証を試みたテナント ID (リクエストパス由来)。")
  tenantId: string;
  username?: string;
  reason?: string;
  sessionId?: string;
  clientId?: string;
  @doc("接続元 IP。監査イベントの保持期間中は平文のまま保存する。")
  ip?: string;
  @doc("接続元 User-Agent。監査イベントの保持期間中は平文のまま保存する。")
  userAgent?: string;
  countryCode?: string;
  @doc("device fingerprint。監査イベントの保持期間中は平文のまま保存する。")
  deviceFingerprint?: string;
  riskScore?: int64;
}


@doc("ResourceOwner が /api/auth/change_password で自身のパスワードを変更した。change-password 監査と通知の起点。")
model PasswordChanged {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
}


@doc("ログイン試行が per-account (10 回 / 900 秒窓 / 900 秒ロックアウト) または per-IP\n(30 回 / 900 秒窓 / 900 秒ロックアウト) のスロットリング閾値に到達してロックされた。\nSIEM の credential stuffing / brute-force 検知の入力イベント。keyHash は username\n(lowercase) または IP の SHA-256 で、平文を audit log に流さない。")
model LoginThrottled {
  occurredAt: utcDateTime;
  @doc("スロットリングが起きたテナント ID (リクエストパス由来)。")
  tenantId: string;
  @doc("account または ip")
  kind: string;
  @doc("username (lowercased) または IP の SHA-256 hex")
  keyHash: string;
  retryAfterSeconds: int64;
}


@doc("クレデンシャル試行洪水時に、個別の AuthenticationFailed を 1 行ずつ書かず\n(tenantId, kind, keyHash, 5 分窓) の bucket に集約したことを表す。1 つの窓につき最初の 1 件だけ\nemit し、以後の増分は bucket store の count に積む。したがって本 payload の\ncount は emit 時点の値で、実体の件数は ListAuthenticationEventBuckets で観測する。\nkeyHash は SHA-256 hex で平文を audit log に流さない (LoginThrottled と同方針)。")
model AuthenticationEventAggregated {
  occurredAt: utcDateTime;
  @doc("集約が起きたテナント ID。")
  tenantId: string;
  @doc("failed_login | throttled | mfa_failed")
  kind: string;
  @doc("kind:keyHash:windowStartUnix の合成キー")
  bucketKey: string;
  @doc("username (lowercased) または IP の SHA-256 hex")
  keyHash: string;
  @doc("emit 時点の窓内件数 (実体は bucket store で伸び続ける)")
  count: int64;
  firstSeen: utcDateTime;
  lastSeen: utcDateTime;
  @doc("窓内で頻出した keyHash (最大 5 件)")
  topKeys: string[];
}


@doc("多段認証フローの 1 ステップ (password / totp / webauthn 等) が成功した。acr 確定の入力。")
model AuthenticationStepCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  @doc("password | totp | webauthn | recovery_code 等")
  step: string;
  sessionId?: string;
}


@doc("多段認証フローの 1 ステップが失敗した。")
model AuthenticationStepFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  step: string;
  reason?: string;
}


@doc("MFA チャレンジ (TOTP / WebAuthn 等) を提示した。")
model MfaChallengeIssued {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
  sessionId?: string;
}


@doc("MFA チャレンジに成功した。")
model MfaChallengeSucceeded {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
  sessionId?: string;
}


@doc("MFA チャレンジに失敗した。bucket 集約 (kind=mfa_failed) の入力にもなる。")
model MfaChallengeFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId?: string;
  factorType: MfaFactorType;
  sessionId?: string;
}


@doc("MFA 必須ポリシー下の未登録 user が、有効な管理者承認を消費して登録専用 pending flow へ入った。password 認証だけでは通常セッションを成立させない。")
model MfaEnrollmentRequired {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
  bypassId: string;
  deadline: utcDateTime;
}


@doc("login enrollment flow で factor 登録と所持証明が完了し、同一 LoginSession が MFA 済みに昇格した。")
model MfaEnrollmentCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
  factorType: MfaFactorType;
}


@doc("管理者が MFA 未登録 user に短期・単発の enrollment bypass を発行した。")
model MfaEnrollmentBypassIssued {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  bypassId: string;
  expiresAt: utcDateTime;
}


@doc("password 認証済みの対象 user が enrollment bypass を単発消費し、登録専用 pending flow へ入った。")
model MfaEnrollmentBypassConsumed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  bypassId: string;
  sessionId: string;
}


@doc("管理者が未消費の enrollment bypass を取り消した。")
model MfaEnrollmentBypassRevoked {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  bypassId: string;
}


@doc("未消費の enrollment bypass が期限を過ぎて利用不能と判定された。期限切れを初めて観測した時に発行する。")
model MfaEnrollmentBypassExpired {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  bypassId: string;
}


@doc("管理者が対象 user の認証器リセットを要求した。削除実行前に発行し、要求時点の対象を監査に残す。")
model AuthenticatorResetRequested {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  targets: AuthenticatorResetTarget[];
}


@doc("管理者による認証器リセットが完了した。対象 factor を削除し mfa_enrolled を再計算した結果を残す。TOTP / WebAuthn が両方無くなった場合は reenrollment_required=true になり、別途 MfaEnrollmentBypassIssued が発行される。")
model AuthenticatorResetCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  targets: AuthenticatorResetTarget[];
  mfaEnrolled: boolean;
  reenrollmentRequired: boolean;
}


@doc("バックアップ/リカバリコードを 1 つ消費した (TOTP / WebAuthn 喪失時の復旧)。")
model BackupCodeConsumed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  remainingCount?: int64;
}


@doc("LoginSession を開始した。session 一覧・調査の起点。")
model SessionStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
  amr?: string[];
  acr?: string;
  @doc("セッション開始時の接続元 IP。監査イベントの保持期間中は平文のまま保存する。")
  ip?: string;
  @doc("セッション開始時の User-Agent。監査イベントの保持期間中は平文のまま保存する。")
  userAgent?: string;
}


@doc("LoginSession の有効期限を延伸した (アクティブ継続)。")
model SessionRefreshed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
}


@doc("外部 IdP 経由の federated login が成功した。")
model FederatedAuthenticated {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  @doc("外部 IdP 識別子 (例: google / github / saml:acme)")
  provider: string;
  sessionId?: string;
}


@doc("tenant administrator が upstream connection を Draft として作成した。secret / token / certificate 本文は含めない。")
model IdentityProviderConnectionCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  protocol: IdentityProviderProtocol;
}


@doc("metadata / trust validation を通過した connection を login routing で利用可能にした。")
model IdentityProviderConnectionActivated {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
}


@doc("connection を login routing から除外した。")
model IdentityProviderConnectionDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
}


@doc("検証済み external subject と local User の link を作成した。external token / assertion と PII claim は含めない。")
model FederatedIdentityLinked {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  userId: string;
  @doc("explicit | verified_email | jit")
  linkingMethod: string;
}


@doc("step-up 済み ResourceOwner が external identity link を解除した。")
model FederatedIdentityUnlinked {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  userId: string;
}


@doc("upstream callback の trust / correlation / policy 検証を fail closed で拒否した。token、assertion、state、nonce は含めない。")
model FederatedLoginRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  reason: string;
}


@doc("admin が user として操作するセッションを開始した (impersonation)。本人通知を前提とする\n監査事象で、retention 短縮の対象外。")
model SessionImpersonationStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  @doc("impersonate した admin の sub")
  actorUserId: string;
  @doc("対象 user の sub")
  targetUserId: string;
  sessionId: string;
}


@doc("impersonation セッションを終了した。retention 短縮の対象外。")
model SessionImpersonationEnded {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  sessionId: string;
}


@doc("パスワードリセットリンクを発行した。anti-enumeration のため\n「リクエストが届いた」ことのみ記録し、発行有無 (verified email を持つ\nuser が存在したか) は別途 EmailSent イベントで観測する。\nemailHash は SHA-256 で、平文を audit log に流さない。")
model PasswordResetRequested {
  occurredAt: utcDateTime;
  @doc("リセットを要求したテナント ID (リクエストパス由来)。")
  tenantId: string;
  @doc("送信元 email の SHA-256 hex")
  emailHash: string;
}


@doc("本人が self-service で MFA factor (現状 TOTP) を登録した。secret は audit に\n流さず種別だけを残す。User.mfa_enrolled を true にする。")
model MfaFactorEnrolled {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
}


@doc("本人が self-service で MFA factor を解除した。解除は有効な TOTP コードによる\n所持証明を伴う。User.mfa_enrolled を false に戻す。")
model MfaFactorRemoved {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
}


@doc("本人が self-service で WebAuthn / Passkey credential を登録した。公開鍵や\ncredential_id は audit に流さず、種別のみ残す。最初の 1 件で User.mfa_enrolled を true にする。")
model WebAuthnCredentialRegistered {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
}


@doc("本人が self-service で WebAuthn / Passkey credential を解除した (step-up 済み)。\n残る第二要素が無くなった場合は User.mfa_enrolled を false に戻す。")
model WebAuthnCredentialRemoved {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
}


@doc("本人が backup recovery code を生成 / 再生成した (step-up 済み)。平文や hash は\naudit に流さず、発行件数のみ残す。再生成は既存 set を全置換する。")
model RecoveryCodesGenerated {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
  @doc("発行した recovery code の件数。")
  count: int64;
}


@doc("本人が backup recovery code を明示的に失効した (step-up 済み)。以降 recovery code は\n第二要素として使えない。")
model RecoveryCodesRevoked {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
}


@doc("LoginSession が終了した。本人 / admin の明示的な失効では actorUserId が操作者、\nreason が self_revoke / admin_revoke になる。sessionId は OIDC sid と同値であり、\nOAuth2 bounded context はこの sessionId をキーに RefreshTokenRecord.sid が一致する\nfamily を失効し、登録済み RP へ front/back-channel logout を通知する。\nAccess Token 自体は本イベントで即時失効しない (短命 TTL に委ねる)。")
model SessionEnded {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
  sessionId: string;
  @doc("失効を実行した主体の sub (self は本人)。")
  actorUserId: string;
  reason: SessionEndReason;
}


@doc("本人が高 sensitivity 操作のための step-up 再認証を開始した。\n利用可能な factor の提示要求であり、まだ再認証は成立していない。")
model StepUpRequested {
  occurredAt: utcDateTime;
  @doc("本人の所属テナント ID。")
  tenantId: string;
  userId: string;
  sessionId: string;
}


@doc("step-up 再認証が成立した。method は再認証に使った factor\n(password | totp)。検証材料 (パスワード / コード) は payload に含めない。")
model StepUpCompleted {
  occurredAt: utcDateTime;
  @doc("本人の所属テナント ID。")
  tenantId: string;
  userId: string;
  sessionId: string;
  @doc("password | totp")
  method: string;
}

}
