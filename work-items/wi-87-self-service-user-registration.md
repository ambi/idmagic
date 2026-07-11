---
depends_on: [wi-6-real-email-sender-adapter]
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-03
---

# セルフサービスのユーザ登録 (self-service sign-up) を導入する

## Motivation
現状ユーザは管理者の CreateAdminUser でしか作成できず、エンドユーザ自身が
アカウントを新規登録する導線が無い。一方、代表的な IdP はいずれも
self-service registration を標準機能として持つ:

- Keycloak: realm の "User registration" スイッチと registration ページ。
- Okta / Entra External ID (旧 Azure AD B2C): self-service sign-up flow。
- OneLogin: self-registration policy。

顧客向け (B2C / B2B) IdP として使うには、テナントが許可した場合に限り
エンドユーザが自分でサインアップし、email 検証を経て利用開始できる導線が要る。
本 WI は password ベースの最小 sign-up (email + password + 表示名) を、
テナント設定でゲートされた public フローとして追加する。ソーシャル / 外部 IdP
経由の JIT 登録は [[wi-30-inbound-federation-and-identity-broker]] の範囲とする。

## Scope
- **decision**:
  - 新規 ADR: self-registration をテナント設定 (allow_self_registration、既定 off) でゲートする。email 検証必須 (ADR-030 の one-time token 方針を踏襲)、 tenant 内 email 一意、登録直後は未検証 = login 不可 (fail-closed)、 account enumeration を避ける応答方針を記録する。
- **scl**:
  - §3.3 interfaces: RegisterUser (public / unauthenticated / tenant-scoped) と、 browser フロー用の登録トランザクション取得を追加する。
  - §3.2 models: RegistrationRequest (email / password / 表示名) を追加する。
  - §3.4 states/events: UserSelfRegistered イベントを追加する。
  - §3.5 invariants: tenant 内 email 一意、未検証ユーザは login 不可を明示する。
  - §3.7 permissions: RegisterUser は public、対象 tenant は解決済み tenant に固定。
  - tenancy: AdminSettings に allow_self_registration を追加し、 UpdateAdminSettings で切り替え可能にする。
- **go**:
  - usecase RegisterUser を追加し、既存 email 検証トークンストア (EmailChangeTokenStore / password reset と同パターン) を再利用して VerifyEmail 相当の検証を経てから login 可能にする。
  - テナント設定が off の場合は構造的に 404/無効化する (fail-closed)。
- **http**:
  - POST /register (public, CSRF + same-origin 必須) と登録トランザクション context を追加する。bot 対策 / rate limit は [[wi-27-endpoint-rate-limit-and-bot-mitigation]] に委譲する。
- **ui**:
  - auth-flow に RegisterPage を追加し、LoginPage から導線を張る。送信後は 「確認メールを送信しました」を表示し、検証リンクで利用開始できるようにする。
- **documentation**:
  - README に self-registration の有効化手順とテナント設定を追記する。

## Out of Scope
- 外部 IdP / ソーシャル経由の JIT 登録 ([[wi-30-inbound-federation-and-identity-broker]])。
- CAPTCHA / bot mitigation ([[wi-27-endpoint-rate-limit-and-bot-mitigation]])。
- 招待ベースのオンボーディング (invite flow)。
- progressive profiling / 多段の追加属性収集。

## Plan
- registration は Identity Management の User 作成能力を再利用するが、未検証 user を先に作らず `PendingRegistration` を Authentication が所有する。tenant policy が disabled の realm では route 自体を発見可能にしても一律拒否する。
- request は normalized email、tenant、terms/policy version、TTL、one-time token hash を保持する。応答は既存 user/新規 request/重複 request で均一化し、wi-27 の endpoint limiter と既存 login throttle の correlation hash を再利用する。
- email verify 成功時だけ、Identity Management の create-user command、password policy/HIBP/history、default group/required action を1 transactionで適用する。既存 user と競合した場合は account を上書きせず recovery 導線へ送る。
- tenant policy は allowed email domain、admin approval requirement、default groups、initial authentication method を持つ。admin approval を有効にした場合は Verified→PendingApproval→Activated の状態を追加する。
- frontend は realm-aware signup→check-email→verify→password/enrollment→login の導線を既存 auth shell/branding 上に置き、管理 UI は tenant settings で policy を編集する。

## Tasks
- [ ] T001 [SCL] PendingRegistration lifecycle、tenant RegistrationPolicy、Start/Verify/Approve interfaces、events/invariants/scenarios を追加して再生成する。
- [ ] T002 [Domain/Store] registration/token model、normalization、memory/ValkeyまたはPostgreSQL store、TTL/one-time consume を実装する。
- [ ] T003 [Usecases] start/resend/verify/approve/expire を EmailSender、limiter、password policy、Identity Management user creationへ接続する。
- [ ] T004 [HTTP] realm-scoped signup/verify endpoints と uniform response、CSRF/browser transaction、secure cookie を追加する。
- [ ] T005 [UI] signup/check-email/expired/approval/complete 画面と tenant policy 管理を追加する。
- [ ] T006 [Verify] existing/disabled/deleted user、domain policy、token replay/expiry、concurrent verify、default group、enumeration/load を検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: テナント設定で登録を有効化 → RegisterPage で email/password 登録 → 確認メールのリンクで検証 → login 成立。設定を無効化すると登録導線が塞がる ことを確認する。
- 手動: 既存 email で登録しても「存在する」ことが応答から判別できないことを 確認する。

## Risk Notes
public な書き込みエンドポイントを増やすため、account enumeration / spam 登録 /
未検証ユーザの滞留が主なリスク。enumeration は応答統一で、spam は rate limit
(wi-27) への委譲と検証必須で緩和する。既定 off とし、明示的に有効化した
テナントだけが露出する設計にする。
