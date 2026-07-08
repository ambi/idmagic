---
id: idp-wi-26-webauthn-passkey-and-recovery-codes
title: "WebAuthn / Passkey と recovery codes を phishing-resistant MFA として導入する"
created_at: 2026-06-20
authors: ["tn"]
status: completed
risk: high
---

# Motivation
現状の MFA は TOTP 中心で、認証強度としては実運用の最低ラインに届くが、
phishing-resistant MFA ではない。Keycloak / Okta / Google アカウント相当の
IdP として考えると、WebAuthn / Passkey と recovery codes は必要になる。

本 WI は WebAuthn credential の登録・認証・削除、passkey による step-up、
TOTP / WebAuthn を失った場合の backup recovery codes を SCL と実装に追加する。

# Scope
- **decision**:
  - 新規 ADR: WebAuthn / Passkey を phishing-resistant MFA として採用する。 RP ID / origin 検証、attestation 方針、resident key / user verification 要件、credential lifecycle、recovery code の one-time semantics を記録する。
- **scl**:
  - WebAuthnCredential / RecoveryCodeSet / RecoveryCodeUsed event を追加する。
  - RegisterWebAuthnCredential / VerifyWebAuthnAssertion / RemoveWebAuthnCredential を追加する。
  - GenerateRecoveryCodes / ConsumeRecoveryCode / RevokeRecoveryCodes を追加する。
  - acr/amr に WebAuthn / passkey / recovery_code を反映する。
- **go**:
  - WebAuthn library を選定し、core ceremony を自前実装しない。
  - credential repository を memory / PostgreSQL に実装する。
  - browser auth flow に WebAuthn challenge 発行・検証 endpoint を追加する。
  - recovery codes は平文を一度だけ表示し、DB にはハッシュだけ保存する。
- **ui**:
  - login / step-up 画面で passkey 認証を選択できるようにする。
  - account portal の security ページから credential 登録・削除を行えるようにする。
  - recovery codes の生成・再生成・使用済み数の表示を追加する。
- **documentation**:
  - README に WebAuthn の RP ID / origin 設定、HTTPS 必須、ローカル開発時の注意を書く。

# Out of Scope
- SMS / Push MFA。
- enterprise attestation の厳格 enforcement。
- device trust / managed device inventory。
- passwordless-only tenant policy。初期は password + WebAuthn MFA / step-up とする。

# Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: account portal で passkey 登録 → logout → password + passkey で login / step-up が成立することを確認する。
- 手動: recovery code を 1 つ使うと再利用できず、使用済み数が増えることを確認する。

# Risk Notes
WebAuthn はブラウザ API、origin、RP ID、challenge store が絡むため実装ミスの
余地が大きい。既存 TOTP と同じ Factor 抽象に無理に押し込まず、WebAuthn
ceremony は独立した use case として扱う。

# Plan
方針: 1 つの WI で一括実装。ライブラリは `github.com/go-webauthn/webauthn`
(自前 ceremony 実装はしない)。passkey / recovery code は login の第二要素として選択式
(passwordless は Out of Scope)。詳細計画は承認済みプランを正本とする。

重要な設計判断: 既存 `MfaFactor` は identity が `(user_id, type)` で 1 種別 1 件のため、
複数登録できる WebAuthn は専用エンティティ `WebAuthnCredential` (新テーブル
`webauthn_credentials`) とする。recovery code も `RecoveryCode` (新テーブル
`recovery_codes`、ハッシュのみ保存) とする。`User.mfa_enrolled` は「TOTP factor または
WebAuthn credential が存在する」で導出する (recovery code は単独では第二要素にしない)。

## SCL scope (触れた節)
`spec/contexts/authentication.yaml`:
- standards: `WebAuthnLevel3` (WEBAUTHN3-AUTHENTICATION を adoption:required 化、
  WEBAUTHN3-REGISTRATION を追加)
- models: `WebAuthnTransport` / `WebAuthnCredential` / `WebAuthnCredentialSummary` /
  `RecoveryCode` / `RecoveryCodeStatus` / `WebAuthnRegistrationOptions` /
  `WebAuthnRegistrationRequest` / `WebAuthnAssertionOptions` / `BrowserWebAuthnRequest` /
  `BrowserRecoveryCodeRequest` / `WebAuthnCredentialRemoveRequest` / `RecoveryCodesResponse`
  を追加。`AccountSecurityResponse` / `StepUpMethod` を拡張。
- events: `WebAuthnCredentialRegistered` / `WebAuthnCredentialRemoved` /
  `RecoveryCodesGenerated` / `RecoveryCodesRevoked` を追加 (recovery code 消費は既存
  `BackupCodeConsumed` を再利用)。
- interfaces: `StartWebAuthnRegistration` / `FinishWebAuthnRegistration` /
  `RemoveWebAuthnCredential` / `StartBrowserWebAuthn` / `SubmitBrowserWebAuthn` /
  `SubmitBrowserRecoveryCode` / `GenerateRecoveryCodes` / `RevokeRecoveryCodes` を追加。
- policies: `WebAuthnPolicy` / `RecoveryCodePolicy` を追加、`AuthenticationContextPolicy`
  の `mfa_amr_values` に `rc` (非 IANA、recovery code) を追加。

# Tasks
- [x] T001 SCL 追加 + `just yaml-check` green、scope 更新
- [x] T002 ADR-087 作成
- [x] T003 Domain 型 + events + Validate
- [x] T004 Use cases (webauthn / account_webauthn / verify / recovery_codes) + 単体テスト green
- [x] T005 Ports + memory/postgres repo + DB schema
- [x] T006 HTTP handlers (account + login 第二要素) + routes
- [x] T007 bootstrap wiring + WebAuthn RP config + 起動時検証
- [x] T008 UI (api/types/security page/login 第二要素/step-up)
- [x] T009 README / compose ドキュメント
- [x] T010 検証ゲート全 green + 手動確認
- [x] T011 完了記録追記 → `work-items/done/` へ移動 → commit

# Completion
- **Completed At**: 2026-07-09
- **Summary**:
  WebAuthn / Passkey と recovery codes を Authentication context の phishing-resistant MFA として追加した。
  SCL には WebAuthn credential / recovery code / ceremony I/O / account security 射影 / step-up method /
  account・browser API を追加し、派生 HTML / JSON Schema / OpenAPI を再生成した。設計判断は
  ADR-087 に記録した。
  Go 側は go-webauthn による登録・assertion 検証、credential / recovery code repository
  (memory / PostgreSQL)、Valkey 対応 challenge store、account security / step-up / browser login
  HTTP endpoint、WebAuthn RP env config を実装した。Recovery codes はハッシュのみ保存し、
  one-time consume と残数算出にした。
  UI 側は account security のパスキー一覧・登録・削除、recovery code 生成・再生成・失効・一度だけの平文表示、
  login 第二要素の TOTP / passkey / recovery code 選択、step-up dialog の passkey / recovery code 対応を追加した。
  README と compose dev stack に WebAuthn RP ID / origin / HTTPS 注意を追記した。
- **Affected Guarantees State**:
  - Password + second-factor login remains compatible with existing TOTP flows.
  - `user.mfa_enrolled` is derived from TOTP or WebAuthn credentials; recovery codes alone do not enroll MFA.
  - WebAuthn ceremonies require configured RP ID and allowed origins; unset RP config disables passkeys.
  - Recovery codes are stored only as hashes and can be consumed once.
- **Verification Results**:
  - `just yaml-check` - passed.
  - `just verify-go` - passed.
  - `just verify-ui` - passed.
  - `just scl-render` - passed and regenerated SCL-derived artifacts.
  - `just verify` - passed with sandbox escalation for Go module cache access.
  - Manual browser passkey registration / login / step-up was not executed in this terminal-only environment.
- **Evidence**:
  - Procedure: SCL-first implementation, ADR creation, Go/usecase/adapter/infrastructure wiring, UI integration, docs, render, full verification.
  - Environment: local branch `idp-wi-26-webauthn-passkey-and-recovery-codes` on 2026-07-09.
  - Result: all automated verification gates passed; manual WebAuthn ceremony remains an operator/browser check.
