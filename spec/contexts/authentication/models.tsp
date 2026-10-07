namespace IdMagic.Contract {

@doc("EmailSender のポートでメッセージを送った。ポートはフェイルオープンなので、送信の失敗はユースケースの層へ伝わらず、SIEM はこのイベントがあるかどうかでそれを観測する。`to` は SHA-256 のハッシュとして記録する。")
model EmailSent {
  occurredAt: utcDateTime;
  @doc("受信者のメールアドレスの SHA-256 の 16 進数。")
  toHash: string;
  @doc("`password_reset`、`email_change`、`email_verification` など。")
  purpose: string;
  delivered: boolean;
}


@doc("上流の接続の送受信のプロトコル。ソーシャルのプロバイダーのプリセットは、`oidc` の初期の設定として現れる。")
enum IdentityProviderProtocol {
  Oidc: "oidc",
  Saml: "saml",
}


@doc("接続のライフサイクルの状態。ディスカバリーとログインの振り分けに使えるのは Active の接続だけである。新たに作成した接続は Disabled で始まる。")
enum IdentityProviderConnectionStatus {
  Active: "active",
  Disabled: "disabled",
}


@doc("外部の ID をローカルの User とどう関連付けるか。デフォルトの `none` は、既存の連携のほかは何も自動で関連付けない。`verified_email` は、確認済みのメールアドレスでの一意な一致を、テナントが明示したときに限って許す。")
enum AccountLinkingPolicy {
  None: "none",
  VerifiedEmail: "verified_email",
}


@doc("検証済みの上流のクレームと属性を、ローカルの ID のフィールドへ射影する、テナントが管理する設定。JIT のプロビジョニングには、サブジェクトとユーザー名の対応付けが必要である。")
model ClaimMapping {
  subject: string;
  username: string;
  email?: string;
  email_verified?: string;
  name?: string;
}


@doc("テナントに属する上流の信頼の設定。リクエスト中に与えられた URL を取得することはなく、設定時に検証したメタデータから得た、最後に正常だったエンドポイント、鍵、証明書だけを使う。クライアントシークレットは書き込み専用の `secret_reference` として届き、DataKeys のテナントごとの DEK でエンベロープ暗号化して保持する。")
model IdentityProviderConnection {
  id: string;
  tenant_id: string;
  display_name: string;
  protocol: IdentityProviderProtocol;
  status: IdentityProviderConnectionStatus;
  issuer: string;
  client_id?: string;
  @doc("クライアントシークレットそのものを運ぶ書き込み専用のフィールド。データベースには DataKeys のテナントごとの DEK でエンベロープ暗号化した暗号文だけを保存し、値も暗号文も API のレスポンスには現れない。フィールド名は、置き換えた `env:` の参照方式から引き継いでいる。")
  secret_reference?: string;
  @doc("`secret_reference` を設定しているかどうかを示す、読み取り専用の導出したフィールド。値や暗号文を運ぶ書き込みには使わず、管理コンソールの詳細の表示が、設定済みか未設定かだけを示すのに使う。")
  client_secret_configured: boolean;
  authorization_endpoint?: string;
  token_endpoint?: string;
  jwks_uri?: string;
  saml_sso_url?: string;
  @maxLength(2048)
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


@doc("接続の試験の構造化された結果。OIDC のディスカバリーと JWKS に到達できること、SAML の証明書が解析できること、`secret_reference` が解決する、すなわち復号できることを確かめる。値そのものは含まず、シークレットもトークンも証明書の本体も含まない。")
model IdentityProviderConnectionTestResult {
  success: boolean;
  failures: string[];
}


@doc("変わらない外部のサブジェクトをローカルの User と関連付ける連携。一つのテナント、プロバイダー、サブジェクトが対応するローカルのユーザーは一人までで、一人のローカルのユーザーとプロバイダーが対応するサブジェクトも一つまでである。")
model FederatedIdentity {
  tenant_id: string;
  provider_id: string;
  external_subject: string;
  local_user_id: string;
  linked_at: utcDateTime;
  last_login_at?: utcDateTime;
}


@doc("上流とのブラウザーの往復を関連付ける、一回限りの一時的な記録。state、nonce、PKCE の検証子、または RelayState と AuthnRequest の ID を、テナントとプロバイダーに結び付け、10 分以内に一度だけ使う。")
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


@doc("サインインの UI に公開してよい形の、有効な上流のプロバイダー。シークレット、内部のエンドポイント、証明書は含まない。")
model ExternalIdentityProviderSummary {
  id: string;
  display_name: string;
  protocol: IdentityProviderProtocol;
}


@doc("検証済みの外部の ID を、ローカルの User と LoginSession に関連付けた結果。")
model FederatedLoginCompletion {
  local_user_id: string;
  provider_id: string;
  session_id: string;
  return_to?: string;
}


@doc("管理者が対応付けを確かめるために読む、正規化した結果。トークン、アサーション、シークレットは返さない。")
model ExternalIdentityClaimsPreview {
  subject: string;
  username: string;
  email?: string;
  email_verified: boolean;
  name?: string;
}


@doc("新しいパスワードが最近のパスワードの履歴の項目と一致する。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:password_reuse` である。")
model PasswordReuseError is ProblemDetails;


@doc("対象のユーザーが、MFA の登録の回避の条件を満たさない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:mfa_enrollment_not_allowed` である。")
model MfaEnrollmentNotAllowedError is ProblemDetails;


@doc("ユーザーは求めた種類の要素をすでに保持しているので、登録するものがない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:mfa_already_enrolled` である。これは呼び出し元についての判断ではなく現在の状態との衝突であり、そこが MfaEnrollmentNotAllowedError との違いである。前者はユーザーに何もしなくてよいことを、後者は管理者に連絡することを求める。")
model MfaAlreadyEnrolledError is ProblemDetails;


@doc("求めたリセットの対象の組み合わせは、対象のユーザーの現在の MFA の状態に対して有効でない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:authenticator_reset_not_allowed` である。")
model AuthenticatorResetNotAllowedError is ProblemDetails;


@doc("認証済みのブラウザーのセッションの、管理シェルの起動情報。CSRF トークン、サブジェクト、現在のレルム、有効ロールを返す。")
model AccountContextResponse {
  csrf_token: string;
  id: string;
  preferred_username?: string;
  tenant_id?: string;
  realm?: string;
  roles?: string[];
}


@doc("未認証のブラウザーのフォームが SameSite の Cookie とともに使う、CSRF トークンの起動情報。")
model BrowserCsrfContext {
  csrf_token: string;
}


@doc("第二要素（MFA）の種類。このアプリケーションのために選んだ RFC 8176 の amr の値の部分集合である。`totp` は RFC 6238、`webauthn` は WebAuthn Level 3 で、`hwk` と `swk` は将来のために予約している。")
enum MfaFactorType {
  Totp: "totp",
  Webauthn: "webauthn",
  Hwk: "hwk",
  Swk: "swk",
}


@doc("ユーザーが登録した一つの第二要素（MFA）。`totp` では、シークレットを base32 で保存し、認証時に RFC 6238（T0=0、STEP=30s、`digits` 桁、前後一ステップのウィンドウ）で評価する。アルゴリズムは SHA1 で、シークレットは 20 バイトで生成する。`User.mfa_enrolled` が true であることは、この集合が空でないことにちょうど等しい。")
model MfaFactor {
  @doc("所有するユーザーの `sub`。")
  user_id: string;
  type: MfaFactorType;
  @doc("`totp` では base32 のシークレット。別の要素が自身の資格情報の JSON を独自のフィールドに入れられるように、開いたままにしている。永続化するときは DataKeys のテナントごとの DEK でエンベロープ暗号化し、平文は保存しない。")
  secret?: string;
  @doc("ユーザーが付ける表示名。たとえば `iPhone Authenticator`。")
  label?: string;
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("LoginSession がまだ完了していない理由。`none` は通常の認証済みのセッション、`challenge` は登録済みの要素の検証待ち、`enrollment` は新しい要素の、管理者の承認による登録待ちである。")
enum LoginPendingPurpose {
  None: "None",
  Challenge: "Challenge",
  Enrollment: "Enrollment",
}


@doc("MFA を登録していないユーザーに管理者が発行する、短命で一回限りの登録の承認。パスワードのステップが成功した後に使う信頼の起点で、セッションを登録の保留へ移せるようにする。サインインの MFA の要求を誰かに免除するものではない。期限切れ、失効済み、使用済みのものは、フェイルクローズで使えない。")
model MfaEnrollmentBypass {
  id: string;
  tenant_id: string;
  user_id: string;
  issued_by: string;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  consumed_at?: utcDateTime;
  revoked_at?: utcDateTime;
  expired_at?: utcDateTime;
}


@doc("管理者による認証器のリセットの対象。`totp` と `webauthn` はすべての MfaFactor または WebAuthnCredential の削除を、`recovery_code` はバックアップのリカバリーコードの組全体の削除を意味する。")
enum AuthenticatorResetTarget {
  Totp: "totp",
  Webauthn: "webauthn",
  RecoveryCode: "recovery_code",
}


@doc("対象のユーザーの認証器をリセットするときに、管理者が削除の対象として指定する集合。")
model AuthenticatorResetRequest {
  targets: AuthenticatorResetTarget[];
}


@doc("認証器のリセットの結果。`reenrollment_required` は、TOTP と WebAuthn がどちらもなくなり、一回限りの回避を自動で発行したので、次のサインインが登録専用の保留のフローに入ることを示す。回避は `reenrollment_required` が true のときだけ含む。")
model AuthenticatorResetResponse {
  mfa_enrolled: boolean;
  reenrollment_required: boolean;
  bypass?: MfaEnrollmentBypass;
}


@doc("ブラウザーのログインセッション。PostgreSQL が唯一の情報源なので、プロセスを再起動してもセッションは有効なままである。Cookie は不透明なセッションの id だけを運び、`sub` と `auth_time` はサーバー側のストアにある。amr と acr もここに残り、後の `/authorize` でステップアップを決める。`authentication_pending=true` は、サインインのポリシーまたはステップアップが別の要素を求めるときにだけ、同じ LoginSession が移る中間の状態である。MFA の要素を登録しているだけでは保留にならない。別の要素の待ちへ移るときに、完了した別の `pwd` のセッションを残してはならない。`/authorize` の完了の経路は、保留中のセッションを未認証として扱い、対応する要素のチャレンジへ向かわせる。Cookie は `http_only` と `same_site=Lax` で発行する。`tenant_id` は常に認証の解決のクエリの一部なので、セッションの id が推測されたりテナントをまたいで再送されたりしても、データベースの層は他のテナントの行を返さない。隔離はフェイルクローズである。失効は `revoked_at` と `revoke_reason` を墓標として設定し、行を削除しないので、再度の失効は冪等に成功する。認証の解決が有効とみなすのは `revoked_at IS NULL AND expires_at > now` の行だけである。")
model LoginSession {
  id: string;
  tenant_id: string;
  user_id: string;
  @doc("認証が成功した Unix 秒。セッションの開始時刻として変わらず、一覧はこれで新しい順に並べる。")
  auth_time: int64;
  @doc("認証が成功した方式。閉じた語彙から取り、RFC 8176 の登録済みの値の pwd、otp、webauthn、hwk、swk と、このアプリケーションの IANA に登録していない拡張の rc（リカバリーコード）、tdev（記憶した信頼済みデバイス）、federated（上流の IdP に委ねた認証）からなる。それ以外を含む LoginSession は保存しない（RFC8176-AMR-VOCABULARY）。")
  amr: string[];
  @doc("認証コンテキストのクラスの参照の URN。amr が otp、webauthn、hwk、swk、rc、tdev のいずれかを含むときは `urn:idmagic:acr:mfa`、それ以外は `urn:idmagic:acr:pwd` である。フェデレーションによるサインインは、上流の IdP が実際に何を検証したかをブローカーが判別できないので、pwd のままにする（OpenID Connect Core の 2 節と RFC 8176）。")
  acr: string;
  @doc("別の要素の検証を待つ中間の状態かどうか。完了すると false になる。")
  authentication_pending: boolean = false;
  @doc("`authentication_pending` の目的。`enrollment` の間、セッションはアカウント、管理、アプリケーションのどのリソースにも届かず、登録の API だけを許す。")
  pending_purpose: LoginPendingPurpose = LoginPendingPurpose.None;
  @doc("`pending_purpose=enrollment` を完了する期限。これを過ぎてからの登録の確定は拒否し、セッションを認証済みへ引き上げない。")
  enrollment_deadline?: utcDateTime;
  @doc("Enrollment へ移るために使った、管理者が発行した回避。監査での関連付けのために残す。")
  enrollment_bypass_id?: string;
  @doc("パスワードまたは MFA によるステップアップの再認証が最後に成功した Unix 秒。成功したことがなければ 0 である。機微性の高いセルフサービスの操作の新しさを判断する。")
  step_up_at: int64 = 0;
  expires_at: utcDateTime;
  @doc("認証の結果として最後に使われた時刻。認証の解決のたびに更新するが、前回から LoginSessionTouchInterval 以内の更新は永続化しないので、値は粗い粒度になり、アイドルのタイムアウトもその粒度で近似する。")
  last_seen_at: utcDateTime;
  @doc("失効した時刻。失効していない間は null である。失効は墓標であり、行を削除しない。")
  revoked_at?: utcDateTime;
  @doc("失効の理由。`revoked_at` と一緒に設定する。`revoked_at` が null の間は null である。")
  revoke_reason?: SessionEndReason;
}


@doc("Authentication の境界づけられたコンテキストが、OAuth 2.0 / OpenID Connect の境界づけられたコンテキストへ渡す認証の結果。後者は、パスワード、UserRepository、セッションの Cookie の詳細を見ない。")
model AuthenticationContext {
  user_id: string;
  auth_time: int64;
  amr: string[];
  acr?: string;
  session_id?: string;
}


@doc("セルフサービスのセキュリティの画面に示す、一つの MFA の要素の射影。シークレットは返さず、種類、ラベル、作成と最後の利用の時刻だけを返す。現在は TOTP だけである。")
model AccountMfaFactor {
  type: MfaFactorType;
  label?: string;
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("エンドユーザーのアカウントポータルのセキュリティの概要である `/api/account/security` が返すもの。パスワードの変更時刻、MFA（TOTP と WebAuthn）を登録しているかどうか、バックアップのリカバリーコードの残数をまとめる。")
model AccountSecurityResponse {
  password_changed_at?: utcDateTime;
  totp_enrolled: boolean;
  factors: AccountMfaFactor[];
  webauthn_credentials: WebAuthnCredentialSummary[];
  recovery_codes: RecoveryCodeStatus;

  @doc("テナントのデフォルトのサインインのポリシーが MFA を求め始める時刻。その開始がまだ未来で、ユーザーが MFA の要素も WebAuthn の資格情報も保持していない間だけ含み、ポータルが、パスワードだけのサインインが使えなくなる前に登録するようユーザーに警告できるようにする。ポリシーの規則そのものは返さない。")
  mfa_enforcement_start_at?: utcDateTime;
}


@doc("TOTP の登録の開始が返すもの（セルフサービス）。クライアントは確認するまでシークレットを保持し、`otpauth_uri` を QR コードまたは手入力のために提供する。シークレットは確認（ConfirmTotpEnrollment）とともに戻ってくるまで永続化しない。")
model TOTPEnrollmentStart {
  secret: string;
  otpauth_uri: string;
  account_name: string;
  issuer: string;
}


@doc("管理者の承認によるログインの登録のフローの中で、TOTP の登録を確定する。シークレットもコードも監査証跡やログに届かない。")
model BrowserMfaEnrollmentConfirmRequest {
  secret: string;
  code: string;
  return_to?: string;
}


@doc("管理者が対象のユーザーに一回限りの登録の回避を発行する入力。")
model MfaEnrollmentBypassIssueRequest {
  expires_in_seconds: int64 = 900;
}


@doc("管理者に示す登録の回避の表現。シークレットはなく、対象のユーザーのパスワードのステップが成功したときにサーバーが自動で使う。")
model MfaEnrollmentBypassResponse {
  bypass: MfaEnrollmentBypass;
}


@doc("TOTP の登録の確認のリクエスト（セルフサービス）。シークレットは開始時のもので、コードはそこから生成した現在のコードであり、所持の証明である。")
model TOTPEnrollmentConfirmRequest {
  @minLength(1)
  secret: string;
  @minLength(1)
  code: string;
}


@doc("MFA の要素の削除のリクエスト（セルフサービス）。コードは所持の証明としての現在の TOTP のコードである。削除には、ユーザー本人とその証明が必要である。")
model MfaFactorRemoveRequest {
  @minLength(1)
  code: string;
}


@doc("WebAuthn の認証器の接続の方法についてのヒント（WebAuthn Level 3 の 5.8.4 節、AuthenticatorTransport）。")
enum WebAuthnTransport {
  Usb: "usb",
  Nfc: "nfc",
  Ble: "ble",
  Internal: "internal",
  Hybrid: "hybrid",
}


@doc("ユーザーが登録した一つの WebAuthn またはパスキーの資格情報。ユーザーは複数を保持できるので、MfaFactor とは別の集合で、`credential_id` で一意に識別する。`public_key` は COSE の公開鍵、`sign_count` は複製の検出に使う署名のカウンターである。オリジンと RP に結び付く所持の要素で、フィッシングに強い MFA とステップアップに使う。セレモニーは、デプロイメントの設定から RP ID とオリジンを解決し、`user_verification=preferred`、`attestation=none`（エンタープライズのアテステーションは厳密には強制しない）、`resident_key=discouraged`（パスワードレスは範囲外）、32 バイトのチャレンジ、120 秒のタイムアウトを使う（WebAuthn Level 3 の 5 節、7 節、13 節）。")
model WebAuthnCredential {
  @doc("認証器が返す資格情報の ID。base64url で符号化する。RP の中で一意である。")
  credential_id: string;
  @doc("所有するユーザーの `sub`。")
  user_id: string;
  @doc("COSE の形の公開鍵。base64url で符号化する。")
  public_key: string;
  @doc("直近のアサーションの署名のカウンター。後退するカウンターは複製の疑いとして拒否する（`sign_count_regression=reject`）。")
  sign_count: int64;
  @doc("認証器が宣言したトランスポートのヒント。usb、nfc、ble、internal、hybrid のいずれか。")
  transports: string[];
  @doc("認証器の AAGUID。base64url で符号化する。機種についてのヒントである。")
  aaguid?: string;
  @doc("ユーザーが付ける表示名。たとえば `MacBook Touch ID`。")
  label?: string;
  @doc("資格情報をデバイスの間で同期できるかどうか（WebAuthn の BE フラグ）。")
  backup_eligible: boolean;
  @doc("資格情報が現在バックアップされているかどうか（WebAuthn の BS フラグ）。")
  backup_state: boolean;
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("セルフサービスのセキュリティの画面に示す、一つの WebAuthn の資格情報の射影。公開鍵も `sign_count` も返さず、識別と表示に必要なものだけを返す。")
model WebAuthnCredentialSummary {
  credential_id: string;
  label?: string;
  transports: string[];
  created_at: utcDateTime;
  last_used_at?: utcDateTime;
}


@doc("TOTP または WebAuthn を失ったときのための、一つのバックアップのリカバリーコード。平文は生成時に一度だけ示し、`code_hash`（SHA-256 の 16 進数）だけを保存する。`consumed_at` が null でないことは使用済みで再利用できないことを意味する。ユーザーの有効なコードは一つの組として扱い、再生成のときは丸ごと置き換える（`regenerate_replaces_all`）。生成は、紛らわしい文字を除いた base32 に似たアルファベット 23456789abcdefghijkmnpqrstuvwxyz から、それぞれ `code_length=10` 文字のコードを `count=10` 個作る。これはバックアップだけのためのもので、単独で第二要素にはならない（NIST SP 800-63B の 5.1.2 節、Look-up Secrets）。")
model RecoveryCode {
  user_id: string;
  @doc("リカバリーコードの平文の SHA-256 の 16 進数。平文は保存しない。")
  code_hash: string;
  generated_at: utcDateTime;
  consumed_at?: utcDateTime;
}


@doc("セルフサービスのセキュリティの画面に示す、リカバリーコードの状態。コードそのものは返さず、生成した時刻と、全体の数と残りの数だけを返す。")
model RecoveryCodeStatus {
  generated_at?: utcDateTime;
  total: int64;
  remaining: int64;
}


@doc("一つの信頼済みデバイス、すなわち「このデバイスを記憶する」の資格情報。ユーザーごとに、サインイン中に本物の第二要素（TOTP または WebAuthn）が成功した直後にユーザーが明示的に同意したときだけ発行する。ブラウザーは `selector.verifier` を、レルムに限定した HttpOnly、SameSite=Lax の Cookie で受け取り、サーバーはセレクターと `verifier_hash`（SHA-256 の 16 進数）だけを保存する。セレクターで一行を取り出し、検証子のハッシュを定数時間で比べるので、Cookie の平文もデバイスのフィンガープリントも保存しない。検証子は使うたびにローテーションするので、盗まれた古い Cookie は次の正当な Cookie で無効になる。有効性は、絶対の期限（`expires_at`）とアイドルの期限（`last_used_at` にアイドルの期間を加えたもの）の両方で区切る。")
model TrustedDevice {
  @doc("デバイスの識別子。監査と、ユーザーが失効させるときのキーである。")
  id: string;
  @doc("属するテナント。Cookie はレルムに限定されるので、テナントをまたいで解決することはない。")
  tenant_id: string = "default";
  @doc("所有するユーザーの `sub`。")
  user_id: string;
  @doc("Cookie の前半を成す秘密でない検索のキー。base64url で符号化する。一意で、一行を特定するためだけに使う。")
  selector: string;
  @doc("Cookie の後半である検証子の SHA-256 の 16 進数。平文は保存せず、使うたびにローテーションする。")
  verifier_hash: string;
  @doc("User-Agent から導く表示用のラベルで、ブラウザーと OS の系統だけを示す。たとえば `Chrome / macOS`。生の User-Agent も IP も保存しない。")
  label?: string;
  created_at: utcDateTime;
  @doc("第二要素を省くために最後に使った時刻。アイドルの期限はここから測る。")
  last_used_at: utcDateTime;
  @doc("`created_at` にテナントの `trusted_device_max_age_seconds` を加えたもの。絶対の期限である。")
  expires_at: utcDateTime;
  @doc("失効した時刻。失効していない間は null である。失効は墓標であり、行を削除しない。")
  revoked_at?: utcDateTime;
  @doc("失効の理由。`revoked_at` と一緒に設定する。`revoked_at` が null の間は null である。")
  revoke_reason?: TrustedDeviceRevokeReason;
}


@doc("信頼済みデバイスを失効させた理由。`self_revoke` はユーザーが明示的に失効させたこと、`password_change` と `mfa_change` は資格情報の変更にともなう一括の失効、`admin_revoke` は管理者による認証器のリセット、`account_disabled` はアカウントの無効化、`session_revoke` はユーザーまたは管理者がすべてのセッションを失効させたことである。")
enum TrustedDeviceRevokeReason {
  self_revoke: "self_revoke",
  password_change: "password_change",
  mfa_change: "mfa_change",
  admin_revoke: "admin_revoke",
  account_disabled: "account_disabled",
  session_revoke: "session_revoke",
}


@doc("セルフサービスのセキュリティの画面に示す、一つの信頼済みデバイスの射影。セレクターも `verifier_hash` も返さず、識別と表示に必要なものだけを返す。`current` は、現在のリクエストが提示した Cookie に一致するデバイスかどうかを示す。")
model AccountTrustedDevice {
  id: string;
  current: boolean;
  label?: string;
  created_at: utcDateTime;
  last_used_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("ListMyTrustedDevices が返すもの。ユーザー自身の有効な信頼済みデバイスを、最後に使った新しい順に並べる。テナントが信頼済みデバイスを無効にしているときは常に空である。")
model AccountTrustedDeviceListResponse {
  devices: AccountTrustedDevice[];
  @doc("テナントの `trusted_device_max_age_seconds`。0 は機能が無効であることを意味し、UI はデバイスを記憶する方法を示さない。")
  max_age_seconds: int64;
}


@doc("アカウントのセキュリティ通知の種類。どのメッセージもちょうど一つの種類に属し、種類ごとにユーザーが受信を止めてよいかが決まる。`new_device_sign_in` は以前に見たことのないブラウザーからのサインイン、`credential_change` はパスワードの変更、`mfa_change` は認証の要素、リカバリーコード、信頼済みデバイスの追加または削除、`contact_change` はメールアドレスの変更の申し出と確定、`session_revoked` はユーザーまたは管理者による明示的なセッションの失効、`impersonation` は管理者がユーザーとして操作し始めたことである。")
enum SecurityNotificationCategory {
  new_device_sign_in: "new_device_sign_in",
  credential_change: "credential_change",
  mfa_change: "mfa_change",
  contact_change: "contact_change",
  session_revoked: "session_revoked",
  impersonation: "impersonation",
}


@doc("ユーザー自身のセキュリティ通知の設定。行がないことは、すべてが有効であることとまったく同じ意味なので、最初の変更まで行は作らない。`disabled_categories` に載せられるのは任意の種類だけで、必須の種類を指定した更新は拒否する。乗っ取りの後に攻撃者が最初に止めるのは通知であり、止められるのであれば存在しないのと変わらないためである。")
model NotificationPreference {
  @doc("所有するユーザーの `sub`。`users.id` は全体で一意なので、`tenant_id` はない。")
  user_id: string;
  @doc("ユーザーが受信を止めた種類。必須の種類は載せられない。")
  disabled_categories: SecurityNotificationCategory[];
  updated_at: utcDateTime;
}


@doc("ユーザーが以前にサインインしたことのある一つのブラウザー。`new_device_sign_in` の判断と、その重複の除去の両方に使う。サインインのたびに `(user_id, device_hash)` で作成または更新し、行を新たに作ったときだけ、新しいデバイスからのサインインとして通知する。`device_hash` は、テナントの関連付けのソルトを加えた User-Agent の SHA-256 の 16 進数で、生の User-Agent も IP も保存しない。")
model KnownSignInDevice {
  @doc("所有するユーザーの `sub`。")
  user_id: string;
  @doc("`SaltedHash(tenant salt, User-Agent)` の 16 進数。テナントをまたいで関連付けることはできない。")
  device_hash: string;
  @doc("User-Agent から導く表示用のラベルで、ブラウザーと OS の系統だけを示す。たとえば `Chrome / macOS`。")
  label?: string;
  first_seen_at: utcDateTime;
  last_seen_at: utcDateTime;
}


@doc("一つの種類のセキュリティ通知をユーザーにどう示すか。`mandatory` が true の種類は常に `enabled` が true で、更新で false にはできない。")
model AccountNotificationCategoryPreference {
  category: SecurityNotificationCategory;
  @doc("ユーザーが無効にできない種類かどうか。")
  mandatory: boolean;
  @doc("ユーザーが現在この種類の通知を受け取っているかどうか。")
  enabled: boolean;
}


@doc("GetMyNotificationPreferences が返すもの。すべての種類を仕様が定める安定した順序で返すので、UI は種類の一覧を自身のコードに持つ必要がない。")
model AccountNotificationPreferences {
  categories: AccountNotificationCategoryPreference[];
}


@doc("UpdateMyNotificationPreferences の入力。有効にする種類ではなく無効にする種類を指定するので、後から加わった種類は既存の設定で有効として引き継がれる。必須の種類を指定したリクエストは 400 で拒否する。")
model UpdateNotificationPreferencesRequest {
  @doc("受信を止める種類。載せられるのは任意の種類だけである。")
  disabled_categories: SecurityNotificationCategory[];
}


@doc("StartWebAuthnRegistration が返すもの。`navigator.credentials.create` に渡す PublicKeyCredentialCreationOptions を、変えずに JSON で返す。チャレンジは短命で、サーバー側のセッションに結び付く。")
model WebAuthnRegistrationOptions {
  @doc("PublicKeyCredentialCreationOptions（WebAuthn Level 3）。")
  public_key: Record<unknown>;
}


@doc("FinishWebAuthnRegistration の入力。`navigator.credentials.create` が返したアテステーションを、任意の表示用のラベルとともに JSON で送り返す。")
model WebAuthnRegistrationRequest {
  @doc("アテステーションのレスポンスを含む PublicKeyCredential（WebAuthn Level 3）。")
  attestation: Record<unknown>;
  label?: string;
}


@doc("StartBrowserWebAuthn が返すもの。`navigator.credentials.get` に渡す PublicKeyCredentialRequestOptions を、変えずに JSON で返す。チャレンジは保留中のログインセッションに結び付く。")
model WebAuthnAssertionOptions {
  @doc("PublicKeyCredentialRequestOptions（WebAuthn Level 3）。")
  public_key: Record<unknown>;
}


@doc("SubmitBrowserWebAuthn の入力。`navigator.credentials.get` が返したアサーションを JSON で送り返す。保留中のログインセッションの Cookie が `sub` を、`X-CSRF-Token` が CSRF を解決する。")
model BrowserWebAuthnRequest {
  @doc("アサーションのレスポンスを含む PublicKeyCredential（WebAuthn Level 3）。")
  assertion: Record<unknown>;
  return_to?: string;
  @doc("このデバイスを記憶し、以降は第二要素を省くことへのユーザーの明示的な同意。テナントが信頼済みデバイスを無効にしているときは無視する。")
  remember_device?: boolean;
}


@doc("SubmitBrowserRecoveryCode の入力。一つのリカバリーコードを第二要素として使う。コードは監査証跡にもログにも届かない。")
model BrowserRecoveryCodeRequest {
  code: string;
  return_to?: string;
}


@doc("RemoveWebAuthnCredential の入力。削除する資格情報を `credential_id` で指定する。")
model WebAuthnCredentialRemoveRequest {
  @minLength(1)
  credential_id: string;
}


@doc("GenerateRecoveryCodes が返すもの。リカバリーコードを平文で一度だけ返す。以降は残数（RecoveryCodeStatus）だけを読め、コードは保存も再表示もしない。")
model RecoveryCodesResponse {
  codes: string[];
  generated_at: utcDateTime;
}


@doc("ユーザー自身のサインインの履歴の一つの項目の射影。この部分は既存の監査イベントのストア（UserAuthenticated）から導くので、保持する属性は起きた時刻と方式（amr）だけである。IP、UA、sessionId、clientId の追加は、イベント自体を拡張した後に行う。")
model UserSignInActivity {
  occurred_at: utcDateTime;
  @doc("RFC 8176 の技術的な値（RFC8176-AMR-VOCABULARY）。UI がユーザー向けの名前に訳す。pwd はパスワード、otp は認証アプリ、webauthn はパスキー、rc はリカバリーコードなどである。")
  amr: string[];
}


@doc("ListMySignInActivity と ListUserSignInActivity が返すもの。直近の N 件を新しい順に返す。")
model UserSignInActivityResponse {
  activities: UserSignInActivity[];
}


@doc("LoginSession が終わった理由。`self_revoke` と `admin_revoke` はユーザーまたは管理者による明示的な失効、`logout`、`idle`、`absolute` は自動の終了、`password_change` と `mfa_change` は資格情報の変更にともなうものである。")
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


@doc("セルフサービスのセッションの一覧の一つの項目の射影。シークレットは含まず、識別子と認証の詳細だけを含む。`current` は呼び出し元の現在のセッションかどうかを示す。`started_at` は LoginSession の `auth_time`、`expires_at` は期限切れになる予定の時刻である。")
model AccountSession {
  id: string;
  current: boolean;
  amr: string[];
  acr: string;
  started_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("ListMySessions が返すもの。ユーザー自身の有効なセッションを、開始時刻の新しい順に並べる。")
model AccountSessionListResponse {
  sessions: AccountSession[];
}


@doc("管理用のセッションの一覧の一つの項目の射影。シークレットは含まず、対象のユーザーの有効なセッションの識別と監査に必要なものだけを含む。呼び出し元と対象のユーザーは別人なので、セルフサービスの AccountSession と違い `current` の印はない。")
model SessionRecord {
  id: string;
  user_id: string;
  amr: string[];
  acr: string;
  started_at: utcDateTime;
  last_seen_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("ListSessions（管理用）が返すもの。対象のユーザーの有効なセッションを、開始時刻の新しい順に並べる。")
model SessionRecordListResponse {
  sessions: SessionRecord[];
}


@doc("管理用に表示する、一つの認証イベントの集約（バケット）の射影。攻撃の間に個々のサインインの失敗を 5 分のウィンドウへ集約した結果で、`window_start` はウィンドウの開始時刻、`count` はその中の累計、`first_seen` と `last_seen` はその中の最初と最後の時刻である。`key_hash` は SHA-256 の 16 進数で、平文は含まない。テナントの境界の中にとどまる。")
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


@doc("ListAuthenticationEventBuckets が返すもの。`window_start` の降順で、新しいウィンドウから並べる。")
model AuthenticationEventBucketListResponse {
  buckets: AuthenticationEventBucket[];
}


@doc("ステップアップの再認証に使える要素。`password` は常に、`totp` と `webauthn` は登録済みのとき、`recovery_code` は有効なコードが残っている間だけ使える。")
enum StepUpMethod {
  password: "password",
  totp: "totp",
  webauthn: "webauthn",
  recovery_code: "recovery_code",
}


@doc("StartStepUpAuthentication が返すもの。使える再認証の方法で、UI は再認証のモーダルで選択肢として示す。")
model StepUpStartResponse {
  methods: StepUpMethod[];
}


@doc("CompleteStepUpAuthentication の入力。`method` に応じて、パスワードまたはコードを使う。検証した材料は監査証跡にもログにも届かない。")
model StepUpCompleteRequest {
  method: StepUpMethod;
  password?: string;
  @doc("`totp` と `recovery_code` のための一回限りのコード。")
  code?: string;
  @doc("`method` が `webauthn` のときの WebAuthn のアサーション。形は WebAuthn の仕様が定めるブラウザーの PublicKeyCredential なので、ここで書き直さず不透明なまま運ぶ。")
  assertion?: Record<unknown>;
}


@doc("ブラウザーのクライアントが `/api/auth/totp` へ POST する TOTP のコード。認可リクエストの id は HttpOnly のトランザクションの Cookie から、CSRF は `X-CSRF-Token` ヘッダーから解決する。WebAuthn をはじめとするほかの要素は HTTP の形が大きく異なるので、固有のエンドポイントと値オブジェクトで独立に定める。")
model BrowserTotpRequest {
  @doc("認証アプリが生成した数字のコード。6 桁である（RFC 6238 の 4 節と RFC 4226 の 4 節）。")
  code: string;
  @doc("管理コンソールへの直接のサインインが続くときの、同じレルムの中の移動先。")
  return_to?: string;
  @doc("このデバイスを記憶し、以降は第二要素を省くことへのユーザーの明示的な同意。テナントが信頼済みデバイスを無効にしているときは無視する。")
  remember_device?: boolean;
}


@doc("SPA が `/api/auth/change_password` へ POST する JSON。認証済みのセッションの Cookie が `sub` を解決し、CSRF は `X-CSRF-Token` ヘッダーで検証する。")
model BrowserChangePasswordRequest {
  current_password: string;
  @minLength(12)
  @maxLength(128)
  new_password: string;
}


@doc("SPA が未認証の経路 `/api/auth/forgot_password` へ POST する JSON。列挙を防ぐため、メールアドレスが存在するかにかかわらず常に 204 を返す。")
model BrowserForgotPasswordRequest {
  email: string;
}


@doc("SPA が `/api/auth/reset_password` へ POST する JSON。メールのリンクの一回限りのトークンと、新しいパスワードを受け取る。")
model BrowserResetPasswordRequest {
  token: string;
  @minLength(12)
  @maxLength(128)
  new_password: string;
}


@doc("パスワードリセットの一回限りのトークン。トークンは 32 バイトの乱数を base64url にしたもので、SHA-256 のハッシュだけを保存する。発行から 1800 秒（30 分）有効で一度だけ使い、`consumed_at` を設定した後は再利用できない。期限切れと使用済みの行はガベージコレクションで回収し、INSERT のときにも、同じ `sub` の未使用のトークンを無効にする軽い掃除を行う。")
model PasswordResetTokenRecord {
  user_id: string;
  token_hash: string;
  created_at: utcDateTime;
  @doc("`created_at` に 1800 秒（30 分）を加えたもの。")
  expires_at: utcDateTime;
  @doc("null でなければ使用済みで、再利用できない。")
  consumed_at?: utcDateTime;
}


@doc("ResourceOwner の認証が成功した。`amr` は実際に使った手段を並べる（OpenID Connect Core の 2 節）。")
model UserAuthenticated {
  occurredAt: utcDateTime;
  @doc("認証が行われたテナント。監査の表示はこれで絞り込む。")
  tenantId: string;
  userId: string;
  amr?: string[];
  @doc("認証が開始した LoginSession の id。")
  sessionId?: string;
  @doc("認証を開始した OAuth のクライアント。")
  clientId?: string;
  acr?: string;
  @doc("クライアントの IP。監査イベントを保持する間、平文で保存する。")
  ip?: string;
  @doc("クライアントの User-Agent。監査イベントを保持する間、平文で保存する。")
  userAgent?: string;
  @doc("GeoIP による国コード。GeoIP を配線していないときは空である。")
  countryCode?: string;
  @doc("デバイスのフィンガープリント。監査イベントを保持する間、平文で保存する。")
  deviceFingerprint?: string;
  @doc("リスクのスコア。この work item はフィールドを予約するだけで、値は null である。")
  riskScore?: int64;
}


@doc("ResourceOwner の認証が失敗した。レート制限と不正検知の入力のイベントである。存在しないユーザー名に対する試行も記録するので `sub` はなく、ユーザー名は監査の検索のための `actor.username` の関連付けのキーとして平文で保持する。ユーザー名とほかのクライアントの属性はハッシュ化せず、ほかの失敗のイベントと同じ期間、デフォルトでは 30 日保持する。")
model AuthenticationFailed {
  occurredAt: utcDateTime;
  @doc("認証が試みられたテナント。リクエストのパスから得る。")
  tenantId: string;
  username?: string;
  reason?: string;
  sessionId?: string;
  clientId?: string;
  @doc("クライアントの IP。監査イベントを保持する間、平文で保存する。")
  ip?: string;
  @doc("クライアントの User-Agent。監査イベントを保持する間、平文で保存する。")
  userAgent?: string;
  countryCode?: string;
  @doc("デバイスのフィンガープリント。監査イベントを保持する間、平文で保存する。")
  deviceFingerprint?: string;
  riskScore?: int64;
}


@doc("ResourceOwner が `/api/auth/change_password` で自身のパスワードを変更した。パスワードの変更の監査証跡と通知の起点である。")
model PasswordChanged {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
}


@doc("サインインの試行が抑制の閾値、すなわちアカウントごと（900 秒のウィンドウで 10 回、900 秒のロックアウト）または IP ごと（900 秒のウィンドウで 30 回、900 秒のロックアウト）に達し、ロックアウトした。SIEM でのクレデンシャルスタッフィングと総当たりの検知の入力のイベントである。`keyHash` は小文字にしたユーザー名、または IP の SHA-256 なので、平文が監査ログに届くことはない。")
model LoginThrottled {
  occurredAt: utcDateTime;
  @doc("抑制が起きたテナント。リクエストのパスから得る。")
  tenantId: string;
  @doc("`account` または `ip` のどちらか。")
  kind: string;
  @doc("小文字にしたユーザー名、または IP の SHA-256 の 16 進数。")
  keyHash: string;
  retryAfterSeconds: int64;
}


@doc("資格情報の試行が殺到する間、個々の AuthenticationFailed のイベントを一行ずつ書かずに、`(tenantId, kind, keyHash, 5-minute window)` をキーとするバケットに集約したことを記録する。発行するのはウィンドウの最初のイベントだけで、以降はバケットのストアの件数に加える。そのため、このペイロードの `count` は発行時の値であり、実際の数は ListAuthenticationEventBuckets で観測する。`keyHash` は LoginThrottled と同じく SHA-256 の 16 進数なので、平文が監査ログに届くことはない。")
model AuthenticationEventAggregated {
  occurredAt: utcDateTime;
  @doc("集約が行われたテナント。")
  tenantId: string;
  @doc("failed_login | throttled | mfa_failed")
  kind: string;
  @doc("複合キー `kind:keyHash:windowStartUnix`。")
  bucketKey: string;
  @doc("小文字にしたユーザー名、または IP の SHA-256 の 16 進数。")
  keyHash: string;
  @doc("発行時にウィンドウに入った数。実際の数はバケットのストアで増え続ける。")
  count: int64;
  firstSeen: utcDateTime;
  lastSeen: utcDateTime;
  @doc("ウィンドウで最も多く現れた `keyHash` の値。最大 5 個である。")
  topKeys: string[];
}


@doc("パスワードは正しく、LoginSession は第二要素を待っている。ユーザーはまだ要素を選んでいないので、イベントは示した種類を含む。")
model MfaChallengeIssued {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  @doc("ユーザーが提示できる第二要素の種類。リカバリーコードは MfaFactorType ではないので載せない。")
  factorTypes: MfaFactorType[];
  sessionId?: string;
}


@doc("MFA のチャレンジが成功した。")
model MfaChallengeSucceeded {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
  sessionId?: string;
}


@doc("MFA のチャレンジが失敗した。`kind=mfa_failed` のバケットの集約の入力でもある。")
model MfaChallengeFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId?: string;
  factorType: MfaFactorType;
  sessionId?: string;
}


@doc("MFA を求めるポリシーの下で、登録していないユーザーが有効な管理者の承認を使い、登録専用の保留のフローに入った。パスワードの認証だけでは通常のセッションは確立しない。")
model MfaEnrollmentRequired {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
  bypassId: string;
  deadline: utcDateTime;
}


@doc("ログインの登録のフローの中で、要素を登録して所持の証明を完了し、同じ LoginSession を MFA を満たした状態へ引き上げた。")
model MfaEnrollmentCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  sessionId: string;
  factorType: MfaFactorType;
}


@doc("管理者が、MFA を登録していないユーザーに短命で一回限りの登録の回避を発行した。")
model MfaEnrollmentBypassIssued {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  bypassId: string;
  expiresAt: utcDateTime;
}


@doc("パスワードで認証した対象のユーザーが、登録の回避を一度だけ使い、登録専用の保留のフローに入った。")
model MfaEnrollmentBypassConsumed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  bypassId: string;
  sessionId: string;
}


@doc("管理者が未使用の登録の回避を失効させた。")
model MfaEnrollmentBypassRevoked {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  bypassId: string;
}


@doc("未使用の登録の回避が期限を過ぎて使えないことが分かった。期限切れを最初に観測したときに発行する。")
model MfaEnrollmentBypassExpired {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  bypassId: string;
}


@doc("管理者が対象のユーザーの認証器のリセットを求めた。何も削除する前に発行するので、リクエスト時点で何を対象にしたかが監査証跡に残る。")
model AuthenticatorResetRequested {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  targets: AuthenticatorResetTarget[];
}


@doc("管理者による認証器のリセットが完了した。削除した要素と、計算し直した `mfa_enrolled` を記録する。TOTP と WebAuthn がどちらもなくなったときは、`reenrollment_required` が true で、別に MfaEnrollmentBypassIssued を発行する。")
model AuthenticatorResetCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  userId: string;
  targets: AuthenticatorResetTarget[];
  mfaEnrolled: boolean;
  reenrollmentRequired: boolean;
}


@doc("バックアップのリカバリーコードを使い、TOTP または WebAuthn を失った後のアクセスを回復した。")
model BackupCodeConsumed {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  remainingCount?: int64;
}


@doc("外部の IdP によるフェデレーションのサインインが成功した。")
model FederatedAuthenticated {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  @doc("外部の IdP の識別子。たとえば `google`、`github`、`saml:acme`。")
  provider: string;
  sessionId?: string;
}


@doc("テナントの管理者が上流の接続を下書きとして作成した。シークレット、トークン、証明書の本体は含まない。")
model IdentityProviderConnectionCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  protocol: IdentityProviderProtocol;
}


@doc("メタデータと信頼の検証を通った接続が、ログインの振り分けに使えるようになった。")
model IdentityProviderConnectionActivated {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
}


@doc("接続をログインの振り分けから外した。")
model IdentityProviderConnectionDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
}


@doc("検証済みの外部のサブジェクトとローカルの User の連携を作成した。外部のトークンもアサーションも個人情報のクレームも含まない。")
model FederatedIdentityLinked {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  userId: string;
  @doc("explicit | verified_email | jit")
  linkingMethod: string;
}


@doc("ステップアップした ResourceOwner が外部の ID の連携を外した。")
model FederatedIdentityUnlinked {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  userId: string;
}


@doc("上流のコールバックを、信頼、関連付け、またはポリシーの検査がフェイルクローズで拒否した。トークン、アサーション、state、nonce は含まない。")
model FederatedLoginRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  providerId: string;
  reason: string;
}


@doc("管理者がユーザーとして操作するセッション（なりすまし）を開始した。ユーザーへの通知を前提とする監査イベントであり、保持期間の短縮の対象にならない。")
model SessionImpersonationStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  @doc("なりすました管理者の `sub`。")
  actorUserId: string;
  @doc("対象のユーザーの `sub`。")
  targetUserId: string;
  sessionId: string;
}


@doc("なりすましのセッションが終わった。保持期間の短縮の対象にならない。")
model SessionImpersonationEnded {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  sessionId: string;
}


@doc("パスワードリセットのリンクが求められた。列挙を防ぐため、リクエストが届いたことだけを記録する。実際にリンクを発行したかどうか、すなわち確認済みのメールアドレスのユーザーが存在したかどうかは、EmailSent のイベントで別に観測する。`emailHash` は SHA-256 なので、平文が監査ログに届くことはない。")
model PasswordResetRequested {
  occurredAt: utcDateTime;
  @doc("リセットが求められたテナント。リクエストのパスから得る。")
  tenantId: string;
  @doc("送信されたメールアドレスの SHA-256 の 16 進数。")
  emailHash: string;
}


@doc("ユーザーがセルフサービスで MFA の要素を登録した。現在は TOTP である。シークレットは監査証跡に届かず、種類だけが届く。`User.mfa_enrolled` を true にする。")
model MfaFactorEnrolled {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
}


@doc("ユーザーがセルフサービスで MFA の要素を削除した。削除には、有効な TOTP のコードによる所持の証明をともなう。`User.mfa_enrolled` を false に戻す。")
model MfaFactorRemoved {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
  factorType: MfaFactorType;
}


@doc("ユーザーがセルフサービスで WebAuthn またはパスキーの資格情報を登録した。公開鍵も `credential_id` も監査証跡に届かず、種類だけが届く。最初の一つで `User.mfa_enrolled` を true にする。")
model WebAuthnCredentialRegistered {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
}


@doc("ステップアップしたユーザーが、セルフサービスで WebAuthn またはパスキーの資格情報を削除した。第二要素が残らないときは `User.mfa_enrolled` を false に戻す。")
model WebAuthnCredentialRemoved {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
}


@doc("ステップアップしたユーザーが、バックアップのリカバリーコードを生成または再生成した。平文もハッシュも監査証跡に届かず、発行した数だけが届く。再生成は既存の組を丸ごと置き換える。")
model RecoveryCodesGenerated {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
  @doc("発行したリカバリーコードの数。")
  count: int64;
}


@doc("ステップアップしたユーザーが、バックアップのリカバリーコードを明示的に無効にした。以降、どのリカバリーコードも第二要素として働かない。")
model RecoveryCodesRevoked {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
}


@doc("サインイン中に本物の第二要素が成功した直後に、ユーザーが明示的に同意し、デバイスを記憶した。ペイロードには Cookie の平文も User-Agent も IP も含まず、デバイスの識別子と、失効の判断に必要なタイムスタンプだけを含む。")
model TrustedDeviceRegistered {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
  deviceId: string;
  @doc("記憶のきっかけになった第二要素。`otp` または `webauthn` である。")
  factor: string;
  expiresAt: utcDateTime;
}


@doc("信頼済みデバイスを失効させた。ユーザーが明示的に失効させたときのほか、資格情報の変更、管理者による認証器のリセット、アカウントの無効化、すべてのセッションの失効にともなう一括の失効でも発行する。")
model TrustedDeviceRevoked {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
  deviceId: string;
  reason: TrustedDeviceRevokeReason;
}


@doc("LoginSession が終わった。ユーザーまたは管理者が明示的に失効させたときは、`actorUserId` は操作した者で、`reason` は `self_revoke` または `admin_revoke` である。`sessionId` は OIDC の `sid` に等しく、OAuth2 の境界づけられたコンテキストはこれをキーに、`RefreshTokenRecord.sid` が一致するリフレッシュトークンのファミリーを失効させ、登録済みの Relying Party へフロントチャネルとバックチャネルのログアウトで通知する。アクセストークンそのものはこのイベントでは即座に失効させず、短い TTL に任せる。")
model SessionEnded {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
  sessionId: string;
  @doc("失効を行った者の `sub`。自身による失効ではユーザー本人である。")
  actorUserId: string;
  reason: SessionEndReason;
}


@doc("ユーザーが、機微性の高い操作の前にステップアップの再認証を始めた。使える要素を示すよう求めたもので、再認証はまだ成功していない。")
model StepUpRequested {
  occurredAt: utcDateTime;
  @doc("ユーザーが属するテナント。")
  tenantId: string;
  userId: string;
  sessionId: string;
}


@doc("一つのアカウントのセキュリティ通知を送ったか、送信の試行が失敗した。ユーザーに何を伝えたかを監査証跡に残すためのもので、ペイロードには本文も受信者のアドレスも含まない。通知のディスパッチャー自身が発行し、このイベントの種類はカタログにないので、ここからさらに通知が連鎖することはない。")
model AccountSecurityNotificationSent {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  @doc("通知を受け取るユーザーの `sub`。なりすましの下では、操作した管理者ではなく対象のユーザーである。")
  userId: string;
  category: SecurityNotificationCategory;
  @doc("通知のきっかけになったイベントの種類の名前。")
  triggerEventType: string;
  @doc("EmailSender が受け付けたかどうか。配送はベストエフォートなので、これが false でも元の操作は成立する。")
  delivered: boolean;
}


@doc("ステップアップの再認証が成功した。`method` は使った要素で、`password` または `totp` である。検証した材料（パスワードまたはコード）はペイロードに含まない。")
model StepUpCompleted {
  occurredAt: utcDateTime;
  @doc("ユーザーが属するテナント。")
  tenantId: string;
  userId: string;
  sessionId: string;
  @doc("password | totp")
  method: string;
}


@doc("リクエストには認証済みのセッションが必要だが、ない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:authentication_required` である。セッションはあるがポリシーが操作を拒否することを意味する AccessDeniedError とは別である。")
model AuthenticationRequiredError is ProblemDetails;


@doc("提示したアクセストークンは、期限切れ、失効済み、形式の誤り、改ざん、またはオーディエンスの誤りのため、このリソースサーバーで主体を認証できない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_token` であり、`WWW-Authenticate` ヘッダーは RFC 6750 の `invalid_token` のチャレンジを含む。")
model InvalidAccessTokenError is ProblemDetails;


@doc("リクエストが指定したログインセッションが存在しないか、すでに終わっている。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:session_not_found` である。")
model SessionNotFoundError is ProblemDetails;


@doc("上流の接続を作成または更新する JSON のリクエスト。呼び出し元が設定してよいものだけを含み、id、テナント、状態、タイムスタンプ、`client_secret_configured` はサーバーが管理する。新しい接続は Disabled で始まる。")
model IdentityProviderConnectionRequest {
  @doc("サインインの画面でユーザーに示す名前。")
  display_name: string;
  protocol: IdentityProviderProtocol;
  @doc("OIDC では発行者。設定時に検証する。")
  issuer?: string;
  @doc("OIDC では上流に登録したクライアントの id。")
  client_id?: string;
  @doc("書き込み専用のクライアントシークレットそのもの。暗号文だけを保存し、どのレスポンスも返さない。")
  secret_reference?: string;
  authorization_endpoint?: url;
  token_endpoint?: url;
  jwks_uri?: url;
  saml_sso_url?: url;
  saml_entity_id?: string;
  saml_signing_certificates?: string[];
  claim_mapping: ClaimMapping;
  linking_policy: AccountLinkingPolicy;
  @doc("連携がないときに、ローカルの User をジャストインタイムでプロビジョニングするかどうか。")
  jit_provisioning: boolean;
  @doc("この接続へ振り分けてよいメールアドレスのドメイン。空は制限なしを意味する。")
  allowed_email_domains?: string[];
}


@doc("リクエストが指定した上流の ID プロバイダーの接続が、このテナントに存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:not_found` である。")
model IdentityProviderConnectionNotFoundError is ProblemDetails;


@doc("上流の ID プロバイダーの接続には連携した ID がまだ結び付いているので、削除できない。先に連携を外す。HTTP 409 で返す。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:connection_in_use` である。")
model IdentityProviderConnectionInUseError is ProblemDetails;


@doc("パスキーの認証が設定されていないので、WebAuthn のセレモニーを開始することも完了することもできない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:webauthn_unavailable` である。")
model WebAuthnUnavailableError is ProblemDetails;


@doc("ユーザーはパスキーを登録していないので、チャレンジする資格情報がない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:webauthn_not_enrolled` である。")
model WebAuthnNotEnrolledError is ProblemDetails;


@doc("リカバリーコードのストアが設定されていないので、リカバリーコードを生成することも失効させることもできない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:recovery_unavailable` である。")
model RecoveryUnavailableError is ProblemDetails;


@doc("MFA の要素のストアが設定されていないので、第二要素を検証できない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:mfa_unavailable` である。")
model MfaUnavailableError is ProblemDetails;


@doc("パスワードリセットのリンクが不正、使用済み、または期限切れである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_reset_token` である。一度は有効だったリンクが形式の誤りではなく失われたものなので、400 ではなく 410 とする。")
model InvalidResetTokenError is ProblemDetails;

}
