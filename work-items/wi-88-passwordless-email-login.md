---
depends_on: [wi-6-real-email-sender-adapter]
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-03
---

# パスワードレスの email ログイン (magic link / email OTP) を導入する

## Motivation
現状の first-factor は password のみで、パスワードレスの選択肢が無い。代表的な
IdP はいずれも email ベースのパスワードレス認証を持つ:

- Okta: Email magic link / email OTP を authenticator として提供。
- Entra ID: email one-time passcode。
- Keycloak: email OTP 系の拡張 / passwordless。

email magic link / OTP は「パスワードを覚えない・使い回さない」ユーザ体験を
提供し、password の phishing / 使い回しリスクを避けられる。TOTP を持たない
ユーザの初回サインインにも有効。本 WI は email へ送る one-time link / code による
passwordless first-factor を、テナント設定でゲートして追加する。WebAuthn /
passkey は [[wi-26-webauthn-passkey-and-recovery-codes]] の範囲とする。

## Scope
- **decision**:
  - 新規 ADR: email magic link / email OTP を passwordless first-factor として 採用する。challenge は期限付き・単発消費 (ADR-030 の one-time token 方針)、 amr / acr へ反映 (例 amr: otp / mfa)、account enumeration 抑止、テナント設定 (allow_passwordless_email) でのゲートを記録する。
- **scl**:
  - §3.3 interfaces: StartEmailLogin (email 入力で challenge 発行) と CompleteEmailLogin (code 入力 / link 検証) を追加する。
  - §3.2 models: EmailLoginChallenge を追加する。
  - §3.4 states/events: EmailLoginChallengeIssued / EmailLoginSucceeded を 追加する。
  - §3.5 invariants: challenge は単発消費・短命・試行回数制限を明示する。
  - §3.6 scenarios: email login 成立と、期限切れ / 再利用拒否のシナリオを追加。
  - tenancy: AdminSettings に allow_passwordless_email を追加する。
- **go**:
  - email login challenge ストア (port + memory + postgres + migration) を password reset token と同パターンで追加し、既存 email sender で送信する。
  - usecase: challenge 発行 / 検証を追加し、成功時に既存の browser session / 認証イベントへ配線する。amr を正しく反映する。
- **http**:
  - POST /login/email/start / POST /login/email/verify と、magic link の landing ルートを追加する (CSRF + same-origin)。試行回数 / rate limit は [[wi-27-endpoint-rate-limit-and-bot-mitigation]] に委譲する。
- **ui**:
  - LoginPage に「email でログイン」導線を追加し、EmailLoginPage (code 入力) と magic link の着地画面を追加する。
- **documentation**:
  - README に passwordless email login の有効化と磁気リンクの扱いを追記する。

## Out of Scope
- SMS / 音声 OTP (外部 gateway 依存の別 WI)。
- passwordless-only テナントポリシー (password を無効化する強制)。
- WebAuthn / passkey ([[wi-26-webauthn-passkey-and-recovery-codes]])。
- step-up での email factor 利用 (初期は first-factor login に限定)。

## Plan
- magic link と email OTP は `EmailLoginChallenge` の delivery/verification variant とし、tenant policy で一方または両方を許可する。challenge は login transaction、normalized verified email、user、TTL、attempt count に束縛する。
- OTP/token は hash のみを Valkey shared state に保存し、一回消費・再送時旧challenge失効にする。開始応答と送信時間は user 存在/状態で差を出さず、EmailSender と wi-27 limiter を使う。
- magic link が別 browser で開かれた場合は token だけで完了させず、短い code confirmation または元 browser との transaction binding を要求して forwarding/phishing の被害を抑える。
- 成功結果の `amr=email` と `acr` を明示し、Application sign-in policy が MFA/step-up を要求する場合は既存 second-factor flow へ進める。passwordless を WebAuthn と同じ phishing-resistant 強度に扱わない。
- account disabled/deleted、email変更、password/credential global revoke で未消費 challenge を無効化する。

## Tasks
- [ ] T001 [SCL] EmailLoginChallenge states、tenant method policy、Start/Complete interfaces、amr/acr、events/invariants/scenarios を追加して再生成する。
- [ ] T002 [Domain/Store] challenge、OTP/token hash、attempt/resend/expiry と memory/Valkey adapter を実装する。
- [ ] T003 [Usecases] uniform start、EmailSender template、magic-link/OTP consume、login session handoff、credential-change revoke を実装する。
- [ ] T004 [HTTP/UI] realm-aware email start、check-email、OTP入力、cross-browser confirmation、expired/restart 導線を追加する。
- [ ] T005 [Policy] `amr=email` を Application/tenant sign-in policy と second-factor selection に接続する。
- [ ] T006 [Verify] replay/resend race、forwarded link、brute force、enumeration timing、disabled user、MFA-required app、multi-replica を検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: テナント設定で有効化 → email でログイン開始 → 届いた code / link で サインイン成立。期限切れ / 使用済み challenge が拒否されることを確認する。

## Risk Notes
email 経由の認証は「メール受信箱を握れば入れる」性質があり、challenge の
短命化・単発消費・試行制限を誤ると突破される。enumeration 抑止と rate limit
(wi-27) 委譲、link token を audit に残さない点を必ずテストで担保する。既定 off。
