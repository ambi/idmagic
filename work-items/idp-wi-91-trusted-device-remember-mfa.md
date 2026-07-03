---
id: idp-wi-91-trusted-device-remember-mfa
title: "信頼済みデバイス記憶 (remember this device) で MFA を一定期間スキップする"
created_at: 2026-07-03
authors: ["tn"]
status: pending
risk: high
---
# Motivation
MFA (TOTP [[wi-40-totp-enrollment-qr-code]] / passwordless [[wi-88-passwordless-email-login]]
/ WebAuthn [[wi-26-webauthn-passkey-and-recovery-codes]]) を常用デバイスで毎回
要求するとサインインの摩擦が大きい。代表的なサービスは "trust this device" /
"remember for N days" を提供する:

- Google / Okta / Entra: 信頼済みデバイスで second factor を一定期間スキップ。

本 WI は、ユーザが明示同意した端末を署名付きで記憶し、有効期間内は second
factor をスキップできるようにする。ただし MFA を条件付きで飛ばすため、期間上限・
即時失効・step-up / 機微操作では無視する、といったガードを設計の中心に置く。

# Scope
- **decision**: 新規 ADR: 信頼済みデバイスの識別 (署名付き device cookie / bound token)、 有効期間の上限、失効条件 (パスワード変更 / factor 追加・削除 / 管理者失効 / 全 session 失効で無効化)、step-up [[wi-43-account-portal-step-up-auth]] や 機微操作では信頼を無視する方針、テナント設定でのゲート (既定 off または短期) を記録する。
- **scl**: §3.3 interfaces: TrustCurrentDevice / ListMyTrustedDevices / RevokeTrustedDevice (self) を追加する。, [object Object], §3.4 states/events: TrustedDeviceRegistered / TrustedDeviceRevoked を 追加する。, §3.5 invariants: MFA 判定に trusted device を織り込みつつ、期間上限・ 失効・step-up 時無視を明示する。, [object Object]
- **go**: trusted device store (port + memory + postgres + migration) と署名付き cookie を追加し、login / MFA usecase に信頼判定を差し込む。, 失効イベント (password / factor 変更、全 session 失効) で信頼を無効化する。
- **http**: login / MFA フローに「このデバイスを信頼」を追加し、account portal に 信頼済みデバイスの一覧 / 失効エンドポイントを追加する。
- **ui**: TOTP / login 画面に信頼チェックボックス、AccountSecurityPage に信頼済み デバイス管理を追加する。
- **documentation**: README に信頼済みデバイスの動作・上限・失効条件を追記する。

# Out of Scope
- device posture / managed device inventory (MDM 連携)。
- 高度な device fingerprinting / リスクベース再認証。
- passwordless-only や first-factor のスキップ (対象は second factor のみ)。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動: MFA 時に信頼を有効化 → 再ログインで second factor がスキップされる。 パスワード / factor を変更すると信頼が失効し MFA が再要求されることを確認する。
- 手動: step-up 操作では信頼済みでも再認証が要求されることを確認する。

# Risk Notes
second factor を条件付きで飛ばすため、失効の取りこぼしや期間上限の誤りで MFA が
形骸化する。失効条件を網羅的にテストし、step-up / 機微操作では必ず信頼を無視する。
cookie は署名して改竄・持ち出しに備え、既定は保守的 (off / 短期) に置く。
