---
depends_on: []
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-09
---

# 管理者が認証器をリセットし再登録を強制する緊急復旧導線を追加する

## Motivation

現状 idmagic では TOTP / WebAuthn を喪失したユーザーの復旧経路が、本人管理の backup
recovery code **のみ**である（`ADR-087`）。
recovery code も失うと復旧経路が実質ゼロになり、単一障害点になっている。Explore 調査でも
管理者による認証器リセットや緊急ロック解除の導線は存在しないことを確認した。

Okta（Reset Authenticators）、Microsoft Entra ID（Authentication Administrator による
リセット + Require re-register）、Keycloak（OTP credential 削除 + required action 再登録）は
いずれも「管理者による認証器リセット + 次回ログイン時の再登録強制」を企業向けの緊急
backstop として備える。idmagic はこの層を欠いている。`ADR-088`
の第 2 層としてこの導線を仕様化する。

## Scope

- `spec/contexts/authentication.yaml`:
  - 管理者操作 interface（対象ユーザーの認証器リセット。TOTP factor / WebAuthn credential /
    recovery code の削除と、次回ログイン時の MFA 再登録強制フラグ設定）。
  - 認証器リセットに伴うドメイン状態（`User.mfa_enrolled` の再計算、再登録要求状態）。
  - 監査イベント（authenticator reset requested / completed、re-enrollment required）。
- `spec/contexts/application.yaml` および管理 UI:
  - ユーザー詳細画面での「認証器をリセット」操作、リセット対象 factor の選択、確認 UX。
  - 権限モデル（管理者 / 委任管理者スコープ）を既存の admin 操作に揃える。
- Authentication use cases: リセット実行、`mfa_enrolled` 再計算、再登録強制状態のセット、
  再登録強制と [[wi-127-mfa-enrollment-onboarding-and-enforcement]] の enrollment-required flow の接続。
- OAuth2 browser login handlers: リセット済みユーザーは次回ログインで MFA 再登録 flow に入る。
- Persistence adapters: 再登録強制状態 / required action の保存（必要に応じて）。

## Out of Scope

- 本人確認（ID プルーフィング / ライブネス）ベースのセルフサービス復旧（SSAR 相当）。ADR-088 で
  将来検討に回した。
- 管理者発行の時限アクセスパス（TAP 相当）。必要なら別 work item。ここでは「リセット + 再登録強制」に絞る。
- メールルートによる自動復旧（[[wi-41-secondary-and-recovery-email]] の範疇）。
- 手段冗長化の推進（ADR-088 第 1 層。別 work item）。

## Plan

- 方針:
  - リセットは既存の管理者操作・権限モデル・監査枠組みに揃え、新しい認可軸を増やさない。
  - リセットで対象 factor を削除し、`mfa_enrolled` を残存要素に応じて再計算する。全 factor を
    失った場合は「次回ログインで MFA 再登録を要求」状態にし、wi-127 の enrollment-required flow へ接続する。
  - リセット単体では新しい factor を作らない（管理者が任意の factor を勝手に登録できると別の
    なりすまし面になる）。あくまで削除 + 再登録要求に留める。
  - 全操作を監査イベント化し、誰が誰の何をリセットしたかを追跡可能にする。
- 参考にする外部パターン: Okta Reset Authenticators、Entra ID Require re-register MFA、
  Keycloak OTP 削除 + required action。
- 却下する代替案:
  - 管理者が新 factor を直接登録して渡す: 管理者経由のなりすまし面を作るため不可。削除 + 再登録要求に限定。
  - リセット時に recovery code を自動再発行して管理者に見せる: 平文コードが管理者を経由し漏洩面が広がる。行わない。
- 未決定事項: 委任管理者（[[wi-94-delegated-administration]]）とのスコープ境界、リセット時のユーザー通知メール
  （[[wi-90-account-security-notification-emails]]）連携の要否。

## Tasks

- [x] T001 [SCL] 認証器リセット interface、再登録強制状態、監査イベント、管理 UX を `authentication.yaml` に仕様化する。実装調査の結果、対象 UI アクションは wi-127 の enrollment bypass ボタンと同じ画面 (AdminUserDetailPage) に載る既存パターンであり、`flows:` セクションへのエントリ追加は wi-127 でも行われていない (先例踏襲)。`application.yaml` は変更不要と判断。
- [x] T002 [Domain] リセット後の `mfa_enrolled` 再計算と再登録要求状態の判定を追加する。既存 `SyncMfaEnrolled` (authentication/usecases) をそのまま再利用し、新規 `AuthenticatorResetTarget` enum の SCL↔Go coherence test を追加。RED: `TestAuthenticatorResetTargetMatchesSCL` を先に fail 確認 (wire alias 未登録) → glossary に `RecoveryCode` alias を追加して GREEN (spec `AuthenticatorResetTarget`)。
- [x] T003 [UseCase] 管理者リセット use case（対象 factor 削除 + 再登録要求セット）を追加する。RED: `TestResetUserAuthenticatorsFullResetForcesReenrollment` を `SyncMfaEnrolled` 呼び出しを一時的に外して fail 確認 (mfa_enrolled が再計算されない) → GREEN (scenario `管理者は認証器を全リセットしたユーザーに次回ログインで再登録を強制できる`)。`TestResetUserAuthenticatorsPartialResetKeepsMfaEnrolled` / `RejectsEmptyTargets` / `RejectsCrossTenantTarget` も追加。
- [x] T004 [UseCase] リセット済みユーザーのログインを wi-127 の enrollment-required flow に接続する。設計判断: 新しい状態機械は追加せず、mfa_enrolled が false になった時点で既存 `IssueMfaEnrollmentBypass` を呼んで同じ `MfaEnrollmentBypass` を発行するだけで、既存の `EvaluateMfaEnrollment` / `beginMfaEnrollment` (wi-127) がそのまま次回ログインを Enrollment pending へ導く。RED: `TestAdminResetUserAuthenticatorsFullResetForcesReenrollment` (Go HTTP e2e) をバイパス自動発行を無効化して fail 確認 (`reenrollment_required=false`) → GREEN。
- [x] T005 [Admin/UI] ユーザー詳細画面にリセット操作・対象選択・確認 UX を追加する。`AdminUserDetailPage.tsx` にドロップダウン項目、`AdminUserDialogs.tsx` に `ResetAuthenticatorDialog` (対象 factor チェックボックス、削除のみである旨の警告) を追加。
- [x] T006 [Audit] リセット要求 / 完了 / 再登録要求を監査イベントに出す。`AuthenticatorResetRequested` / `AuthenticatorResetCompleted` を新設し、再登録要求は既存 `MfaEnrollmentBypassIssued` を再利用 (bypass 発行自体が要求の証跡になるため新規イベントは追加しない)。監査カテゴリ `"user"` に登録。
- [x] T007 [Verify] E2E で、全 factor リセット後の再登録強制ログイン、部分リセット後の残存要素動作、権限外操作の拒否を固定する。Go HTTP e2e (`backend/shared/http/server_http/admin_authenticator_reset_e2e_test.go`) で 3 scenario を実際のログインフロー込みで固定し、ブラウザ e2e (`frontend/tests/e2e/ui-scenario-actions.spec.ts`) で admin コンソールの実操作 (ドロップダウン→ダイアログ→送信→通知→メニュー再表示) を固定した。

## Verification

- `just yaml-check`
- `just test-go`
- `just verify-ui`
- `just test-ui-e2e`
- 手動確認:
  - 管理者が対象ユーザーの全認証器をリセットすると、そのユーザーは次回ログインで MFA 再登録を求められる。
  - 一部 factor のみリセットした場合、残存要素で引き続きログインできる。
  - 権限を持たない操作者はリセットできない。
  - リセット・再登録要求が監査イベントに記録される。

## Risk Notes

リスクは高い。認証器リセットは認証境界を管理者権限で越える操作であり、乱用や設計ミスは
なりすまし・恒久ロックアウトに直結する。緩和策として、リセットは削除 + 再登録要求に限定して
管理者による factor 直接登録を禁じ、既存の admin 権限・監査枠組みに揃え、全操作を監査イベント
必須とする。再登録強制は wi-127 の fail-closed な enrollment-required flow を再利用する。

## Completion

- **Completed At**: 2026-08-02
- **Summary**:
  - 管理者操作 `ResetUserAuthenticators` (`POST /api/admin/users/{user_id}/authenticator-reset`,
    policy `TenantAdministrator`) を追加し、対象 user の TOTP factor / WebAuthn credential /
    recovery code から選んだ種別だけを削除する。管理者は代わりの factor を登録できない。
  - 削除後は既存 `SyncMfaEnrolled` で `mfa_enrolled` を再計算し、TOTP と WebAuthn が両方
    無くなった場合だけ既存の `IssueMfaEnrollmentBypass` (wi-127) をそのまま呼んで単発
    enrollment bypass を自動発行する。新しい状態機械や pending 概念は追加せず、既存の
    fail-closed な `EvaluateMfaEnrollment` / `beginMfaEnrollment` gate がそのまま次回ログインを
    Enrollment pending へ導く。一部 factor のみ削除した場合は bypass を発行せず、残存要素で
    通常ログインを継続できる。
  - 新規監査イベント `AuthenticatorResetRequested` / `AuthenticatorResetCompleted` を追加し、
    再登録要求そのものは既存 `MfaEnrollmentBypassIssued` を再利用した (二重イベント化を避けた)。
  - 管理 UI (`AdminUserDetailPage.tsx` / `AdminUserDialogs.tsx`) に「認証器をリセット」操作と
    対象選択・警告文を備えた確認ダイアログを追加した。
  - 永続化層の追加は不要だった (TOTP/WebAuthn/recovery code の `DeleteAllForSub` と
    `mfa_enrollment_bypasses` テーブルを既存のまま再利用)。
- **Verification Results**:
  - `just check-scl` — passed
  - `just scl-render` — passed
  - `just test-go` — passed (新規: use case 単体 4 件、HTTP e2e 3 件、SCL↔Go coherence 1 件)
  - `just lint-go` — passed (0 issues)
  - `just verify-ui` — passed (format / lint / typecheck / build / unit tests 523 件)
  - `just test-ui-e2e` — passed (新規ブラウザ e2e 1 件を含む)。既存スイートに事前から
    存在する無関係テスト (tenant attributes / MCP resource servers / agent credential binding)
    のタイミング起因の間欠的失敗を数回観測したが、本 work item の変更を戻した clean な
    `main` でも同様に発生し、再実行で解消することを確認済み。wi-143 が追加したテストは
    全実行で成功。
  - `just check` (SCL/work-items/ids/architecture/traceability) — passed
    (`architecture.yaml` / `backend/authentication/architecture.yaml` を新規依存・
    ui-page-lines 予算増分に合わせて更新)
  - `just verify` — passed
  - `git diff --check` — passed
- **Affected Guarantees State**:
  - guarantee: 管理者は対象 user の認証器 (TOTP / WebAuthn / recovery code) を選択削除できるが、
    代わりの factor を直接登録することはできない。
  - state: passed
  - guarantee: TOTP と WebAuthn の両方が無くなった場合、次回ログインは既存の wi-127
    enrollment-required flow (fail-closed) に入り、新しい factor の登録を確定するまで
    元の authorization transaction は完了しない。
  - state: passed
  - guarantee: 一部 factor のみ削除した場合、残存要素で通常ログインを継続でき、意図しない
    強制再登録は発生しない。
  - state: passed
  - guarantee: `TenantAdministrator` 以外の操作者、および他テナントの管理者はリセットを
    拒否され、対象 user の認証器は変更されない。
  - state: passed
  - guarantee: リセット要求・完了・再登録要求 (bypass 発行) はすべて監査イベントとして
    記録され、誰が誰の何をリセットしたか追跡できる。
  - state: passed
- **Evidence**:
  - procedure: SCL-first で `AuthenticatorResetTarget` / `AuthenticatorResetRequest` /
    `AuthenticatorResetResponse` / 2 イベント / `ResetUserAuthenticators` interface と 2 scenario
    を `authentication.yaml` に定義した後、domain (既存関数再利用 + enum coherence test)、
    use case、HTTP adapter、監査カテゴリ登録、admin UI の順に実装した。各層で RED (該当機能を
    一時的に無効化して意図した理由で fail することを確認) → GREEN を経た。E2E は
    Go HTTP レベル (実ログインフロー込み) とブラウザレベル (実際の管理コンソール操作) の
    両方で固定した。
  - commands: `just check-scl`, `just scl-render`, `just test-go`, `just lint-go`,
    `just verify-ui`, `just test-ui-e2e`, `just check`, `just verify`, `git diff --check`
  - environment: macOS arm64 workspace
  - actor: Claude (implement-work-item skill)
  - source: pre-commit working tree based on `4c4af2c5`
  - result: passed
  - artifacts: `spec/contexts/authentication.yaml`, `spec/idmagic.html`,
    `spec/idmagic.models.schema.json`, `spec/idmagic.openapi.json`,
    `backend/authentication/mfa/usecases/admin_reset.go`,
    `backend/authentication/mfa/handlers_http/admin_reset_handler.go`,
    `backend/shared/http/server_http/admin_authenticator_reset_e2e_test.go`,
    `frontend/src/features/admin-users/AdminUserDialogs.tsx`,
    `frontend/tests/e2e/ui-scenario-actions.spec.ts`,
    `architecture.yaml`, `backend/authentication/architecture.yaml`

## Out of Scope Disclosure (self-attest, ADR-121)

対象範囲に `adoption: partial` / `excluded` の requirement は無い。work item 本文の
`## Out of Scope` に列挙した項目 (本人確認ベース復旧、管理者発行の時限アクセスパス、
メールルート自動復旧、手段冗長化) は今回未実装のまま。加えて実装過程で以下を明示的に
判断した:
- `AdminUserResponse` (identity-management context) に factor 種別の詳細 (TOTP 有無 /
  WebAuthn 件数 / recovery code 残数) は追加しなかった。リセットダイアログは常時 3種の
  静的チェックボックスを提示し、存在しない factor への削除操作は冪等な no-op として扱う。
  UX 上は「今何が登録されているか」を見せられないため、必要なら別 work item で
  `AdminUserResponse` を拡張する。
- 委任管理者（[[wi-94-delegated-administration]]）
  とのスコープ境界は未決定のまま (wi-94 自体が未実装)。既存の `IssueMfaEnrollmentBypass`
  等と同じ `TenantAdministrator` ポリシーのみを適用した。
- リセット時のユーザー通知メール
  （[[wi-90-account-security-notification-emails]]）
  連携は行っていない。監査イベントには記録されるが、対象ユーザー本人への能動的な通知は
  出さない。
- 自己宛てのリセット (管理者が自分自身の認証器をリセットする) を禁止するガードは
  設けていない。既存の `IssueMfaEnrollmentBypass` / `RevokeMfaEnrollmentBypass` にも
  同様の自己操作ガードが無く、既存の admin 操作モデルとの一貫性を優先した。
