---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-18
depends_on: [wi-254-backend-feature-vertical-slice-convention]
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

feature 案（既存ファイル名の境界を踏襲）:

- `password/`: `password_policy`/`change_password`/`request_password_reset`/
  `reset_password_with_token`、`password_hasher`/`password_history_repository`/
  `password_reset_token_store`/`breached_password_checker`。
- `webauthn/`: `webauthn`/`account_webauthn`/`verify_webauthn_factor`、
  `webauthn_credential_repository`/`webauthn_session_store`。
- `mfa/`: `totp`/`verify_totp_factor`/`account_mfa`/`mfa_enrollment`/`second_factor`/`step_up`、
  `mfa_factor_repository`/`mfa_enrollment_bypass_repository`。
- `session/`: `sessions`/`session_manager`、`session_store`、`login_attempt_throttle`。
- `recovery/`: `recovery_codes`、`recovery_code_repository`。
- feature 横断の共有（`auth_event_buckets`/`signin_activity` 等）は context ルートに残す。
- package 名は各層名のまま。context 横断ハブで named import が必要（wi-254 と同方針）。

## Tasks

- [ ] T001 [Move] `backend/authentication/` を `git mv` で 5 feature 配下へ再配置（全層）。
      共有型は context ルートに残す。
- [ ] T002 [Go] import path を一括置換し、context 横断ハブの named import を修正。
- [ ] T003 [Docs] `ARCHITECTURE.md` frontmatter `modules[].path` の authentication 分を
      `new-architecture` skill で feature 粒度へ同期。
- [ ] T004 [Verify] 下記 Verification を実行し全緑を確認。

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
