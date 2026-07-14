---
depends_on: [wi-26-webauthn-passkey-and-recovery-codes]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-03
---

# 信頼済みデバイス記憶 (remember this device) で MFA を一定期間スキップする

## Motivation
MFA (TOTP [[wi-40-totp-enrollment-qr-code]] / passwordless [[wi-88-passwordless-email-login]]
/ WebAuthn [[wi-26-webauthn-passkey-and-recovery-codes]]) を常用デバイスで毎回
要求するとサインインの摩擦が大きい。代表的なサービスは "trust this device" /
"remember for N days" を提供する:

- Google / Okta / Entra: 信頼済みデバイスで second factor を一定期間スキップ。

本 WI は、ユーザが明示同意した端末を署名付きで記憶し、有効期間内は second
factor をスキップできるようにする。ただし MFA を条件付きで飛ばすため、期間上限・
即時失効・step-up / 機微操作では無視する、といったガードを設計の中心に置く。

## Scope
- **decision**:
  - 新規 ADR: 信頼済みデバイスの識別 (署名付き device cookie / bound token)、 有効期間の上限、失効条件 (パスワード変更 / factor 追加・削除 / 管理者失効 / 全 session 失効で無効化)、step-up [[wi-43-account-portal-step-up-auth]] や 機微操作では信頼を無視する方針、テナント設定でのゲート (既定 off または短期) を記録する。
- **scl**:
  - §3.3 interfaces: TrustCurrentDevice / ListMyTrustedDevices / RevokeTrustedDevice (self) を追加する。
  - §3.2 models: TrustedDevice を追加する。
  - §3.4 states/events: TrustedDeviceRegistered / TrustedDeviceRevoked を 追加する。
  - 所有要素の constraints/contracts: MFA 判定に trusted device を織り込みつつ、期間上限・ 失効・step-up 時無視を明示する。
  - tenancy: AdminSettings に trusted_device_max_age を追加する。
- **go**:
  - trusted device store (port + memory + postgres + migration) と署名付き cookie を追加し、login / MFA usecase に信頼判定を差し込む。
  - 失効イベント (password / factor 変更、全 session 失効) で信頼を無効化する。
- **http**:
  - login / MFA フローに「このデバイスを信頼」を追加し、account portal に 信頼済みデバイスの一覧 / 失効エンドポイントを追加する。
- **ui**:
  - TOTP / login 画面に信頼チェックボックス、AccountSecurityPage に信頼済み デバイス管理を追加する。
- **documentation**:
  - README に信頼済みデバイスの動作・上限・失効条件を追記する。

## Out of Scope
- device posture / managed device inventory (MDM 連携)。
- 高度な device fingerprinting / リスクベース再認証。
- passwordless-only や first-factor のスキップ (対象は second factor のみ)。

## Plan
- TrustedDevice は Authentication 所有の user/tenant/device credential とし、random selectorをcookie、verifier hash/metadata/revoked_at/expires_atをserver側に保持する。fingerprintingだけで端末を信頼しない。
- MFA成功直後かつtenant policy/application policyが許可したときだけ発行し、HttpOnly/Secure/SameSite、realm-scoped cookie、rotation-on-useを適用する。password-only/email-only loginから直接発行しない。
- bypass評価は user、tenant、device record、absolute/idle expiry、Application required auth strength、high-risk/step-up operationを合成する。MFA-requiredでも「remember許可」か「毎回MFA」かをpolicyで区別する。
- password reset、MFA authenticator reset、account disable、global session revoke、risk eventで全deviceをrevokeする。個別revokeはaccount portalでstep-up対象にする。
- auditはdevice ID、created/last-used/revoked reasonを記録し、raw cookie/user-agent/IPを保存しない。

## Tasks
- [ ] T001 [SCL] TrustedDevice lifecycle、tenant/application remember policy、issue/evaluate/revoke interfaces、events/constraints/contracts/scenarios を追加して再生成する。
- [ ] T002 [Domain/Persistence] selector/verifier credential、expiry/rotation/revoke と memory/PostgreSQL repository/indexを実装する。
- [ ] T003 [Authentication] MFA success時issue、login時constant-time verify/rotation、policy evaluationへのamr/acr結果を実装する。
- [ ] T004 [Revocation] password/MFA/account/session/risk eventsからuser全deviceをrevokeする購読/use caseを追加する。
- [ ] T005 [Account UI] MFA画面のremember checkbox、step-up付きdevice list/revoke-all、masked metadataを追加する。
- [ ] T006 [Verify] stolen/old cookie、rotation race、idle/absolute expiry、realm cookie混同、毎回MFA app、global revoke、multi-replicaを検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: MFA 時に信頼を有効化 → 再ログインで second factor がスキップされる。 パスワード / factor を変更すると信頼が失効し MFA が再要求されることを確認する。
- 手動: step-up 操作では信頼済みでも再認証が要求されることを確認する。

## Risk Notes
second factor を条件付きで飛ばすため、失効の取りこぼしや期間上限の誤りで MFA が
形骸化する。失効条件を網羅的にテストし、step-up / 機微操作では必ず信頼を無視する。
cookie は署名して改竄・持ち出しに備え、既定は保守的 (off / 短期) に置く。
