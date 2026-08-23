---
depends_on: [wi-26-webauthn-passkey-and-recovery-codes]
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-03
change_kind: feature
initial_context:
  specification:
    - docs/contexts/authentication/SPECIFICATION.md#REQ-AUTHENTICATION-015
    - docs/contexts/authentication/SPECIFICATION.md#REQ-AUTHENTICATION-017
    - docs/contexts/authentication/SPECIFICATION.md#REQ-AUTHENTICATION-012
  typespec:
    - IdMagic.Contract.RecoveryCode
    - IdMagic.Contract.AccountSession
    - IdMagic.Contract.SignInRule
    - IdMagic.Contract.AdminSettingsResponse
  source:
    - backend/authentication
    - backend/oauth2/handlers_http/authorize_login.go
    - backend/oauth2/handlers_http/authorize_second_factor.go
    - backend/application/usecases/sign_in_policy.go
    - backend/tenancy/domain/tenancy.go
    - frontend/src/features/auth-flow/TotpPage.tsx
    - frontend/src/features/account/AccountSecurityPage.tsx
  tests:
    - backend/authentication
    - backend/application/usecases
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
    - backend/provisioning
affected_spec:
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-026 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-027 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-028 }
  - { path: docs/contexts/authentication/scenarios.md, requirement: REQ-AUTHENTICATION-029 }
  - { path: spec/contexts/authentication/models.tsp, symbol: IdMagic.Contract.TrustedDevice }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Contract.Operations.ListMyTrustedDevices }
  - { path: spec/contexts/application/models.tsp, symbol: IdMagic.Contract.SignInRule }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.AdminSettingsResponse }
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
- **spec (authentication)**:
  - `TrustedDevice` モデル、`AccountTrustedDevice` 射影、`TrustedDeviceRegistered` / `TrustedDeviceRevoked` イベント、`TrustedDeviceRevokeReason` を `models.tsp` に追加する。
  - `ListMyTrustedDevices` / `RevokeMyTrustedDevice` / `RevokeMyTrustedDevices` を `main.tsp` に追加する。
  - 第二要素の送信リクエスト (TOTP / WebAuthn) に `remember_device` を追加する。
  - `RFC8176-AMR-VOCABULARY` に非 IANA 拡張値 `tdev` を加える。
  - `TrustedDeviceLifecycle` の状態遷移表、Design 節、REQ-AUTHENTICATION-026〜029 を追加する。
- **spec (application)**: `SignInRule.allow_trusted_device` を追加し、MFA 必須ルールが信頼済みデバイスで満たされるかを区別する。
- **spec (tenancy)**: `Tenant` / `AdminSettingsResponse` / `TenantUpdateRequest` に `trusted_device_max_age_seconds` を追加する (既定 0 = 無効)。
- **go**: `backend/authentication/trusteddevice` (domain / ports / usecases / db_memory / db_postgres / handlers_http) と `trusted_devices` テーブルを追加し、login / MFA 経路に発行と評価を差し込む。
- **go (失効)**: パスワード変更・リセット、認証要素の追加と削除、管理者による認証器リセット、全セッション失効、ユーザー無効化、匿名化 cascade から全デバイスを失効させる。
- **ui**: 第二要素画面の「このデバイスを記憶する」チェックボックスと、アカウントのセキュリティ画面の信頼済みデバイス管理を追加する。

## Out of Scope
- device posture / managed device inventory (MDM 連携)。
- 高度な device fingerprinting / リスクベース再認証。
- passwordless-only や first-factor のスキップ (対象は second factor のみ)。
- 信頼済みデバイスの管理者向け一覧 / 失効 API (本人のみ)。
- リカバリコードによる第二要素成功からのデバイス記憶。

## Design
`TrustedDevice` は Authentication が所有する user 単位の資格情報であり、fingerprint ではなくサーバーが発行した秘密で識別する。cookie には `selector.verifier` を入れ、サーバーは `selector` と `SHA-256(verifier)` だけを保存する。selector で 1 行を引き、verifier のハッシュを定数時間比較するので、行の走査もハッシュの平文保存も要らない。

発行はログインで**本物の第二要素 (TOTP / WebAuthn) が成立した直後**に限る。パスワードだけの成功、リカバリコードによる成功、登録専用フローからは発行しない。リカバリコードは要素喪失時の復旧経路であり、その時点の端末を信頼に足るものとして扱えないからである。

テナントの `trusted_device_max_age_seconds` が 0 または未設定なら機能ごと無効で、cookie も発行せず評価もしない。既定は 0 (無効)、上限は 90 日とする。

評価はサインインポリシーが MFA を要求し、かつセッションがまだ第二要素を持たない時にだけ行う。有効なら `tdev` を `amr` に加えて `acr` を `urn:idmagic:acr:mfa` へ昇格させる。`tdev` は `rc` と同じくこのアプリケーション固有の非 IANA 値で、「要素を提示したのではなく端末が記憶されていた」ことを RP にも隠さない。`SignInRule.allow_trusted_device=false` のルールは `tdev` を MFA の充足として認めないため、「毎回 MFA」を明示できる。

step-up 再認証は `StepUpMethod` に列挙された factor だけで成立し、`tdev` は候補に入らない。したがって信頼済みデバイスは機微操作の再認証を一切肩代わりしない。

利用のたびに verifier を回転させ、cookie を再発行する。盗まれた古い cookie は次回の正規利用で無効になる。有効期限は絶対期限 (`created_at + max_age`) と idle 期限 (`last_used_at + min(30 日, max_age)`) の両方で切る。

## Plan
1. Spec を先に変える (authentication / application / tenancy)。`just check-spec`。
2. domain と永続化 (memory / PostgreSQL / schema / sqlc)。
3. 発行と評価を login / 第二要素経路へ差し込む。
4. 失効の購読点を配線する。
5. アカウントポータル API と UI。
6. `just verify`。

## Tasks
- [x] T001 [Spec] `TrustedDevice` ライフサイクル、テナント / アプリケーションの記憶ポリシー、発行・評価・失効のインターフェース、イベント、状態遷移、REQ-AUTHENTICATION-026〜029 を追加して再生成する。
- [x] T002 [Domain/Persistence] selector/verifier 資格情報、絶対 / idle 期限、回転、失効と、memory / PostgreSQL のリポジトリとインデックスを実装する。
- [x] T003 [Authentication] 第二要素成功時の発行、ログイン時の定数時間検証と回転、ポリシー評価への `tdev` の反映を実装する。
- [x] T004 [Revocation] パスワード / 認証要素 / アカウント / セッションの各イベントから、ユーザーの全デバイスを失効させる経路を追加する。
- [x] T005 [Account UI] 第二要素画面の記憶チェックボックスと、step-up 付きのデバイス一覧 / 失効、マスク済みメタデータを追加する。
- [x] T006 [Verify] 盗難 / 期限切れ cookie、回転、idle / 絶対期限、realm cookie の混同、毎回 MFA のアプリ、全失効を検証する。

## Verification
- `just test-go-package ./backend/authentication/trusteddevice/...`
- `just verify`
- 手動: MFA 時に信頼を有効化 → 再ログインで second factor がスキップされる。パスワード / factor を変更すると信頼が失効し MFA が再要求されることを確認する。
- 手動: step-up 操作では信頼済みでも再認証が要求されることを確認する。

## Risk Notes
second factor を条件付きで飛ばすため、失効の取りこぼしや期間上限の誤りで MFA が
形骸化する。失効条件を網羅的にテストし、step-up / 機微操作では必ず信頼を無視する。
cookie は selector/verifier の単発資格情報として扱い、既定は無効 (テナント設定 0) に置く。

## Completion
- **Completed At**: 2026-08-16
- **Summary**:
  ログインで本物の第二要素 (TOTP / WebAuthn) が成立した直後に本人が同意したブラウザーを、テナントが明示的に有効にした期間だけ記憶し、次回以降のログインで第二要素の提示を省略できるようにした。端末は指紋ではなく `selector.verifier` のサーバー発行資格情報で識別し、cookie は realm scope の HttpOnly / SameSite=Lax、保存するのは `selector` と `SHA-256(verifier)` だけで、利用のたびに verifier を回転させる。省略が成立したセッションは `amr` に非 IANA 拡張値 `tdev` を得て `acr` が `urn:idmagic:acr:mfa` へ上がるので、記憶による充足と本物の第二要素は下流からも区別できる。`SignInRule.allow_trusted_device=false` はこの充足を認めず「毎回 MFA」を表す。ステップアップ再認証は `tdev` を候補に持たず `step_up_at` も進めないため、機微操作の再認証は肩代わりされない。有効期限は絶対期限 (テナント設定、上限 90 日) と idle 期限 (最終利用から 30 日、絶対期限が短ければそちら) の両方で切る。失効はパスワードの変更とリセット、認証要素の登録と解除、管理者による認証器リセット、全セッション失効、アカウント無効化、匿名化 cascade から配線した。テナント設定 `trusted_device_max_age_seconds` の既定は 0 (機能無効) である。
- **Semantic Difference** (`just spec-diff`):
  - 追加した正規シナリオ: REQ-AUTHENTICATION-026 / 027 / 028 / 029
  - 追加した状態遷移: `TrustedDeviceLifecycle`
  - 追加した TypeSpec 宣言: `TrustedDevice`、`TrustedDeviceRevokeReason`、`AccountTrustedDevice`、`AccountTrustedDeviceListResponse`、`TrustedDeviceRegistered`、`TrustedDeviceRevoked`、`ListMyTrustedDevices`、`RevokeMyTrustedDevice`、`RevokeMyTrustedDevices`
  - 変更した既存宣言: `SignInRule.allow_trusted_device`、`Tenant` / `TenantUpdateRequest` / `TenantSummaryResponse` / `AdminSettingsResponse` の `trusted_device_max_age_seconds`、`BrowserTotpRequest` / `BrowserWebAuthnRequest` の `remember_device`、標準 `RFC8176-AMR-VOCABULARY` への `tdev` の追加
- **Verification Results**:
  - `just verify` - passed (check / lint-go / test-go / lint-ui / format-check-ui / test-ui-unit / build-ui / typecheck-tools / test-tools / check-api-compat)
  - `just test-go-package ./backend/authentication/trusteddevice/...` - passed (domain / usecases / db_postgres)
  - `just check-schema` - passed (`trusted_devices` と `tenants.trusted_device_max_age_seconds` が psqldef で収束)
  - 手動確認は未実施 (自動テストで同等の経路を検証済み: 発行・省略・回転・盗難 cookie・改竄 cookie・毎回 MFA・パスワード変更による失効・認証要素変更による失効・本人による個別失効)
