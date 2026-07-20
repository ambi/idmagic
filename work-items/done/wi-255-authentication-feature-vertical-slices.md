---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-18
depends_on: [wi-254-backend-feature-vertical-slice-convention]
completion:
  completed_at: 2026-07-20
  summary: "authentication を6 feature垂直スライスへ全層再配置し、残存portの所有権をIdManagement userとshared notificationへ是正してArchitecture依存グラフを同期した。"
  verification:
    - "just verify-go"
    - "just build-go"
    - "just test-go"
    - "just yaml-check"
    - "just check-ids"
    - "just verify"
  affected_guarantees_state:
    - "SCL の authentication 規範振る舞いと bounded context 境界は変更していない。"
    - "EmailChangeTokenStore は IdManagement user が所有し、memory/PostgreSQL/sqlc を同featureに局所化した。"
    - "EmailSender は shared notification capability が所有し、Authentication 固有portへのcontext横断依存を解消した。"
    - "Authentication直下portはfeature横断のAuthEventBucketStoreのみで、未使用LoginContinuationは削除した。"
    - "ARCHITECTURE.md のmodule ledgerは実importと一致し、依存グラフは循環しない。"
  evidence:
    - id: "verify-go"
      kind: "test"
      procedure: "Go lint and race-enabled repository tests"
      result: "passed"
    - id: "build-and-test-go"
      kind: "test"
      procedure: "All Go package build and tests"
      result: "passed"
    - id: "architecture-and-ra-validation"
      kind: "static_analysis"
      procedure: "SCL, work-item, ID, Architecture cross-check, and traceability validation"
      result: "passed"
    - id: "repository-verification"
      kind: "test"
      procedure: "Full repository verification including Go, UI, and RA tooling"
      result: "passed"
change_kind: refactor
spec_impact:
  kind: none
  reason: "context 境界・context_map を動かさない純粋な物理配置変更。SCL 規範振る舞いは不変で spec/scl.yaml 編集も scl-render も不要。"
initial_context:
  source: [backend/authentication, ARCHITECTURE.md]
  tests: [backend/authentication]
  stop_before_reading: [frontend, spec]
---

# authentication を password/webauthn/mfa/session/recovery の feature 垂直スライスへ再配置する

## Motivation

authentication は ≈8.3k LOC の大型 context で、`usecases/`・`ports/`・
`adapters/persistence/*` に password・webauthn・mfa(totp)・session・recovery という
複数の sub-domain が同居している。ファイル名（`password_*`, `webauthn*`, `totp*`/`mfa_*`,
`session*`, `recovery_code*`）で境界が既に明確に引かれており、wi-254 で確立した feature
垂直スライス規約を機械的に適用できる。context 内の可読性と変更局所性を高める。

## Scope

- `backend/authentication/` を `password/` `webauthn/` `mfa/`（totp 含む）`session/`
  `recovery/` の feature 層へ再配置（`domain`/`ports`/`usecases`/`adapters/http`/
  `adapters/persistence/{memory,postgres,valkey}` の全層）。`git mv` で履歴を保持。
- Go import path の一括置換と、同一 context 複数 feature を同時 import する箇所
  （`backend/authentication/adapters/http/routes.go` 等の context 横断ハブ）の named import 修正。
- feature 横断の共有ドメイン型（auth event bucket、sign-in activity 等）は wi-254 の方針に
  従い context ルートの共有 `domain/`/`usecases/` に残すか帰属を判断する。
- `ARCHITECTURE.md` frontmatter の `modules[].path` の authentication 分を feature 粒度へ
  同期（`new-architecture` skill）。

## Out of Scope

- SCL（`spec/scl.yaml`）の規範定義・context_map の変更。
- `REGENERATIVE_ARCHITECTURE.md` §3.8 と `ARCHITECTURE.md` の規約散文の再編集
  （wi-254 で確定済み。本 wi は frontmatter の path 同期のみ）。
- `module.go`（DI 束）と `backend/cmd/internal/bootstrap` の組み立て構造の変更（据え置き）。
- authentication 内の feature 分割線そのものの再設計。ファイル名ベースの既存境界を踏襲する。

## Plan

**feature 粒度の再検討（レビュー指摘）**: 当初案は `mfa`（totp を含む）と `webauthn` を
並列 feature としていたが、webauthn 自体が MFA の一手段であるため、`mfa`/`webauthn` を
並列に置くのは「webauthn は MFA でない」ように読めて粒度としておかしい、という指摘を受けた。
totp と webauthn を「具体的な認証方式」として並列 feature にし、両方式を横断する
enrollment/step-up/second-factor オーケストレーションを独立の `mfa` feature として括り出す
6 feature 構成へ変更した。

feature 案（ADR-130 の条件付き規約・命名慣習を踏襲）:

- `password/`: `password_policy`（domain+usecases）/`change_password`/`request_password_reset`/
  `reset_password_with_token`、`password_hasher`/`password_history_repository`/
  `password_reset_token_store`/`breached_password_checker`。HTTP は `password_reset_handler`
  + `account_authflow_handler.go` のうち `handleChangePasswordAPI`。persistence は
  `password_history`/`password_reset_token`（memory/postgres）。
- `totp/`: TOTP 固有コード。`totp`/`verify_totp_factor`、`mfa_factor_repository`（port、
  実体は TOTP factor の保存）、domain `MfaFactor`（`authentication.go` から分離）。
  persistence は `mfa.go`/`mfa_test.go`（memory/postgres、TOTP factor テーブル）。
- `webauthn/`: `webauthn`/`account_webauthn`/`verify_webauthn_factor`、
  `webauthn_credential_repository`/`webauthn_session_store`、domain `WebAuthnCredential`。
  HTTP は `account_webauthn_handler`。persistence は `webauthn.go`/`webauthn_session.go`
  （memory）、`webauthn.go`（postgres）、`WebAuthnSessionStore`（valkey、`stores.go` から分離）。
- `mfa/`: totp/webauthn を横断するオーケストレーション。`account_mfa`/`mfa_enrollment`/
  `second_factor`/`step_up`、`mfa_enrollment_bypass_repository`、domain
  `MfaEnrollmentBypass`/`EvaluateMfaEnrollment`/`MfaEnrollmentDecision`（`authentication.go`
  から分離）。HTTP は `account_step_up_handler`/`admin_mfa_enrollment_handler` +
  `account_security_handler.go` のうち `accountMfaDeps`/`handleStartTotpEnrollment`/
  `handleConfirmTotpEnrollment`/`handleRemoveTotpFactor`（TOTP enrollment の self-service
  UI だが依存が `AccountMfaDeps` のため mfa 側）。`writeAccountMfaError` は shared
  `account_handler.go` から呼ばれるため export し package 境界を越える。persistence は
  `mfa_enrollment_bypasses.go`（memory/postgres）。
- `session/`: `sessions`/`session_manager`、`session_store`、`login_attempt_throttle`
  （port + memory + valkey）、domain `LoginSession`/`LoginPendingPurpose`/`LoginRequest`
  （`authentication.go` から分離）。HTTP は `account_sessions_handler`。persistence は
  `sessions.go`（memory/postgres）。
- `recovery/`: `recovery_codes`/`recovery_code_repository`、domain `RecoveryCode`。
  HTTP は `recovery_codes_handler`。persistence は `recovery_code.go`（memory/postgres）。
- **feature 横断の共有は context ルートに残す**（ADR-130 決定 5/6 と同方針）: port は
  `auth_event_bucket_store`/`email_sender`/`email_change_token_store`/`login_continuation`。
  domain は `authentication_context.go`（`AuthenticationContext`/`Headers`）と `events.go`
  （idmanagement の precedent 通り、event 型は個別 feature に対応するものも含め分離しない）。
  usecases は `acr_vocabulary`/`auth_event_buckets`/`demo_header_resolver`/`retention`/
  `signin_activity`/`user_lifecycle_helpers`。HTTP は `routes.go`（`Deps`/`RegisterRoutes`、
  Phase 2 で `httpdeps` leaf package 化）、`account_handler.go`、`account_consents_handler.go`、
  `account_authflow_handler.go` のうち `handleAccountContext`、`account_security_handler.go`
  のうち `handleGetAccountSecurity`、`admin_auth_event_bucket_handler`、
  `account_activity_handler`（signin activity）、`admin_user_handler_test.go`
  （feature 横断の login 統合テスト）。persistence は `auth_event_buckets.go`/
  `email_change_token.go`（memory/postgres）、`helpers.go`/`harness_test.go`（postgres 共有
  test fixture）。
- package 名は各層名のまま。context 横断・feature 横断ハブで named import が必要
  （wi-254 と同方針、例 `totpdomain`/`webauthndomain`/`mfadomain`/`sessiondomain`/
  `recoverydomain`/`passworddomain`）。
- `adapters/http`（Deps 集約 + フリー関数化）と `adapters/persistence/postgres`（sqlc 6 分割
  + 既存 shared エントリ）は ADR-130 Phase 2 と同じ設計をパイロットの試行錯誤なしで直接適用する
  （idmanagement で確立済みのため）。

## Tasks

層単位でコンパクトに進める（各層完了時に `go build ./...` で内部整合を確認してから次へ）。

- [x] T001 [Go] Domain 層: `domain/authentication.go` を feature 別ファイルへ分割し
      `git mv`/新規ファイルで `password|totp|webauthn|mfa|session|recovery/domain/` へ配置。
      `authentication_context.go`/`events.go` は共有のまま context ルートに残す。
      `LoginPendingPurpose` は `AuthenticationContext.PendingPurpose` と型を共有するため
      shared に残し、`session/domain` は type alias で再エクスポート。
- [x] T002 [Go] Ports 層: 11 ファイルを feature 別 `ports/` へ `git mv`。共有 4 ファイル
      （`auth_event_bucket_store`/`email_sender`/`email_change_token_store`/
      `login_continuation`）は残す。
- [x] T003 [Go] Usecases 層: feature 単位ファイルを `git mv`。共有ヘルパーは
      `ErrUserNotFound`/`LoadSelfUser`/`NormalizedNow`/`DeriveACR`/`SyncMfaEnrolled`
      （totp/webauthn 双方から参照されるため mfa ではなく shared に配置、import cycle 回避）を
      export して context ルート `usecases/` に集約。`removeRequiredAction` は password のみが
      使うため password/usecases 内部ヘルパーへ格下げ。`otpauthIssuerLabel`/`hasSecondFactor`
      等の feature 内部専用ヘルパーは非 export のまま feature 側に残した。
- [x] T004 [Go] Adapters/persistence/memory・valkey 層: feature 単位ファイルを `git mv`。
      valkey `stores.go` は `WebAuthnSessionStore` を webauthn へ、共有ヘルパーは
      `sharedvalkey` 直呼びに簡約して feature 側へインライン化。
- [x] T005 [Go] Adapters/persistence/postgres 層: `sqlc.yaml` を feature 単位 6 エントリ
      （+ 共有 `auth_event_buckets`/`email_change_token` エントリ）へ分割し
      `sqlc generate` で `queries/*.sql`・`sqlcgen/` を再生成。`sqlc` が削除しない旧
      queries dir の孤立 `*.sql.go` は shared エントリから手動削除。`helpers.go`
      （pgtype 変換）は使用箇所がある 5 feature（totp/webauthn/mfa/session/recovery）へ、
      `harness_test.go`（TestMain embedded-postgres 起動）はテストがある
      password/totp/mfa/session へ複製。pgfixtures/pgtest は元々 shared package の
      ため複製不要（ADR-130 の対象は _test.go ローカルヘルパーのみ）。
- [x] T006 [Go] Adapters/http 層: `Deps` を `adapters/http/httpdeps`（leaf package）へ
      切り出し、ハンドラをメソッドからフリー関数へ変換して feature ディレクトリへ移す。
      `routes.go`（`RegisterRoutes` + 横断ハンドラ 3 本: account context/consents/
      signin_activity/auth_event_buckets/security 集計）は context ルート共有のまま
      free 関数化。当初想定より横断ヘルパーが多く（`RequireAuthenticatedSub`/
      `RequireStepUpSub`/`RequireStepUpSession`/`RequireAuthenticatedAuthn`/
      `WriteAccountError`/`WriteAccountMfaError`/`AccountProfileDeps`）、これらは
      mfa/webauthn/session/recovery の複数 feature から呼ばれるため httpdeps へ集約
      （httpdeps は usecases 層へ依存するが adapters/http へは依存しないので import
      cycle は生じない）。`WebAuthnAccountDeps` は webauthn/mfa の 2 feature専用なので
      webauthn/adapters/http が export し mfa 側が import（一方向、cycle なし）。
      `passwordResetCSRF` 等 `_test.go` ローカルヘルパーは package 境界を越えられないため
      session/root の各 test package へ複製した。
- [x] T007 [Go] 内部完了後、リポジトリ全体の import path を修正: 外部消費者（oauth2/
      wsfederation/saml/tenancy/shared/cmd 等、約75ファイル）が参照する
      `backend/authentication/{domain,ports,usecases}` を feature 別 import path へ
      張り替え（named import、コンパイラのエラーメッセージを手掛かりに逐次修正）。
      `shared/adapters/http/server/routes.go` の `authhttp.Deps{...}` 構築はフィールド名
      不変のまま `httpdeps.Deps` 経由に更新。
- [x] T008 [Docs] `ARCHITECTURE.md` frontmatter `modules[].path` の authentication 分を
      `new-architecture` skill で feature 粒度へ同期。約 75 の外部 module の `depends_on`
      も実際の import 先 feature へ更新。
- [x] T009 [Verify] 下記 Verification を実行し全緑を確認。
- [x] T010 [Ownership] `EmailChangeTokenStore` と persistence adapter / sqlc query を
      IdManagement `user` feature へ移し、DI 所有権を `idmanagement.Module` へ移す。
- [x] T011 [Ownership] 汎用 `EmailSender` / `EmailMessage` を shared notification
      capability へ移し、Authentication 固有 port への context 横断依存を解消する。
- [x] T012 [Cleanup] 未使用の `LoginContinuation` を削除し、Authentication 直下 port は
      feature 横断の `AuthEventBucketStore` のみにする。
- [x] T013 [Docs/Verify] `ARCHITECTURE.md` の責務・依存グラフと traceability を同期し、
      Verification を再実行する。

## Verification

- `just verify-go` / `just build-go` / `just test-go` — format/lint/typecheck/build/テスト緑。
- `just yaml-check` / `just check-ids` — RA/SCL の ID・YAML 整合（SCL 不変を確認）。
- `just verify` — 全体スイートの最終確認。
- `git log --follow` で `git mv` の履歴保持を確認、旧配置への import 残存ゼロを grep で確認。

## Risk Notes

- **境界がファイル名で明確**なため機械的だが、mfa と session/webauthn は second-factor/step-up の
  オーケストレーションで相互参照しうる。cross-feature import が増える箇所は named import で対応し、
  過剰な共有型移動を避ける。
- wi-254 完了（規約・パイロット確定）に依存。規約が固まる前に着手しない。
- module.go / bootstrap 据え置きにより DI 面の破壊的変更を回避する。

## Completion

- **Completed At**: 2026-07-20
- **Summary**: Authentication の6 feature全層分割に加え、残存portを再精査した。
  `EmailChangeTokenStore` と全永続化実装を IdManagement `user` featureへ移設し、
  `EmailSender` / `EmailMessage` を shared notification capabilityへ移設した。
  未使用の `LoginContinuation` は削除し、Authentication直下にはfeature横断の
  `AuthEventBucketStore`だけを残した。DIとArchitectureの依存グラフも新しい所有権へ同期した。
- **Verification Results**: `just verify-go`、`just build-go`、`just test-go`、
  `just yaml-check`、`just check-ids`、`just verify` はすべて passed。
- **Out of Scope**: SCL / context_map、RA §3.8 とArchitectureの規約散文、feature分割線の
  再設計は変更していない。
