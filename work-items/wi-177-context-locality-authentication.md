---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-11
---

# authentication コンテキストへバックエンド・コンテキストローカリティを横展開する

## Motivation

[[wi-172]] で確立した [[ADR-089]]・[[ADR-090]]・[[ADR-091]] の型紙を authentication
context へ適用する。authentication は `spec/scl.yaml` context_map 上で 3 context
（OAuth2 / WsFederation / Saml）から `AuthenticationContext` 等を被依存として参照される
最初の非 leaf context であり、[[wi-173]]〜[[wi-176]] の完了を前提に着手する。

被依存があるため、[[wi-172]] の T002 で確立した「adapter 境界での変換」パターン
（[[ADR-089]] item 5）を踏襲し、依存元 context のコード変更を伴う点が leaf context との
主な違いである。

本 WI は振る舞いを変えない純構造 + 生成方式の変更であり、`spec/contexts/authentication.yaml`
を正として双子定義の parity を保つ（SCL 規範は変更しない）。

## Scope

- `internal/shared/spec/authentication.go`（81 行）・`password_policy_resolver.go`
  （41 行、+test）の業務型を `internal/authentication/domain/` へ移設。
- authentication 固有 repository 実装（`shared/adapters/persistence/{postgres,memory}` の
  `mfa.go` / `password_history.go` / `webauthn.go` / `recovery_code.go` /
  `email_change_token.go` / `auth_event_buckets.go` / `auth_event_bucket_store.go`、および
  memory backed の `login_attempt_throttle`）を
  `internal/authentication/adapters/persistence/{postgres,memory}` へ同居。
- authentication の postgres 実装を sqlc 生成へ置換。
- `internal/authentication/module.go` を新設し、`Deps`/`bootstrap` から authentication 分を
  Module へ移す。
- OAuth2 / WsFederation / Saml（[[wi-173]]〜[[wi-175]] で先行移設済み）が
  `AuthenticationContext` 等を参照している箇所を adapter 境界の変換に更新
  （import path の付け替えを含む）。

## Out of Scope

- IdentityManagement・Tenancy の型移設（[[wi-178]]・[[wi-179]]）。
- memory 二重実装の解消。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. [[wi-172]] と同じ内側→外側の順序で進める。
2. T002（Kernel 選別）で `AuthenticationContext` を含め OAuth2/WsFederation/Saml が
   参照する型を洗い出し、[[ADR-089]] item 5 に従い adapter 境界の変換で吸収するか
   `shared/kernel` へ昇格するかを判定する。[[wi-172]] の実測では `shared/kernel` 新設は
   不要と判断されたため、本 WI でも同様の判断になる可能性が高いが、被依存元が複数
   context にまたがる点が application とは異なるため再評価する。
3. 依存元 3 context（既に per-context 化済み）の import 更新は本 WI の Tasks に含める
   （第 2 波の更新、Pending Tasks の見込み通り）。

## Tasks

- [ ] T001 [Domain] `shared/spec/authentication.go` / `password_policy_resolver.go` の
  業務型を `authentication/domain/` へ移設し参照更新。
- [ ] T002 [Kernel] authentication が他 3 context（OAuth2/WsFederation/Saml）と共有する型を
  選別し、adapter 境界変換 or `shared/kernel` 昇格を判定。
- [ ] T003 [Persistence] authentication 固有 repo 実装を
  `authentication/adapters/persistence/{postgres,memory}` へ同居。
- [ ] T004 [Persistence] authentication postgres 実装を sqlc 生成へ置換。
- [ ] T005 [DI] `authentication/module.go` を新設し Module パターン化。
- [ ] T006 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から authentication 分を撤去。
- [ ] T007 [Cross-context] OAuth2/WsFederation/Saml 側の `AuthenticationContext` 等の
  import path を更新（第 2 波）。
- [ ] T008 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [ ] T009 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

## Verification

- `just verify-go` が green。
- `just test-go` で回帰なし。MFA / WebAuthn / recovery code / password history の
  E2E が通る。加えて OAuth2/WsFederation/Saml 側の E2E で cross-context 参照が壊れて
  いないことを確認する。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- locality 指標：`grep -r "internal/shared/spec" internal/authentication | wc -l` が
  ゼロに近づく。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）でスモーク。

## Risk Notes

- **risk: medium**。3 context から被依存があるため、[[wi-173]]〜[[wi-175]] 完了後の
  「第 2 波」import 更新が波及する。認証まわりのため回帰の実害が大きい。
- 軽減：[[wi-173]]〜[[wi-175]] 完了後に着手し、依存元の import が単純な path 付け替えで
  済む状態（adapter 境界の変換パターン）を維持する。各依存元 context の既存 E2E を
  T007 の後に必ず実行する。
