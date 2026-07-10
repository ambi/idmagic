---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-11
---

# wsfederation コンテキストへバックエンド・コンテキストローカリティを横展開する

## Motivation

[[wi-172]] で確立した [[ADR-089]]・[[ADR-090]]・[[ADR-091]] の型紙を wsfederation
context へ適用する。wsfederation は他 context からの被依存が 0（leaf）であり、
oauth2（[[wi-173]]）に続く低リスクな横展開先として選定した。

本 WI 固有の論点として、業務型が `internal/shared/spec/federation.go`（182 行）に
ClaimMapping の型と**混在**している。ClaimMapping はまだ独立 context（`internal/claimmapping/`
相当）を持たないため、本 WI では federation.go を分割し WsFederation 型のみを移設する。

本 WI は振る舞いを変えない純構造 + 生成方式の変更であり、`spec/contexts/ws-federation.yaml`
を正として双子定義の parity を保つ（SCL 規範は変更しない）。

## Scope

- `internal/shared/spec/federation.go` を分割し、WsFederation 型
  （`WsFedTokenType` / `WsFedRelyingParty` / `WsFedSignInIssued` / `WsFedSignInRejected` /
  `WsFedSignOut` / `WsTrustTokenIssued` / `WsTrustTokenRejected` / `EntraFederationConfigured`）
  のみを `internal/wsfederation/domain/` へ移設。
- wsfederation 固有 repository 実装
  （`shared/adapters/persistence/{postgres,memory}` の `wsfed_relying_parties.go`）を
  `internal/wsfederation/adapters/persistence/{postgres,memory}` へ同居。
- wsfederation の postgres 実装を sqlc 生成へ置換。
- `internal/wsfederation/module.go` を新設し、`Deps`/`bootstrap` から wsfederation 分を
  Module へ移す。

## Out of Scope

- **ClaimMapping 型**（`ClaimMappingSource` / `ClaimMappingRule` / `ClaimMappingPolicy` /
  `NameIdConfiguration` / `EntraFederationProfile` / `IssuedClaim`）は
  `federation.go` に同居しているが、ClaimMapping context 自体がまだ独立 package を持たない
  ため `internal/shared/spec` に残置する。ClaimMapping の context 化は別 WI で扱う。
- Saml・Scim・Authentication・IdentityManagement・OAuth2・Tenancy 等、他 context の型移設。
- memory 二重実装の解消。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. [[wi-172]] と同じ内側→外側の順序で進める。
2. T001 の domain 移設で `federation.go` を「WsFederation 型を残す部分」と「ClaimMapping 型を
   `shared/spec` に残置する部分」に分割する。ファイル分割自体は shared 側でも構造改善になる
   ため、`shared/spec/federation.go` は ClaimMapping 型のみを持つ状態に整理してよい
   （ClaimMapping 側のファイル名変更は任意、内容の移動が本質）。
3. oauth2（[[wi-173]]）が先行する場合、`EntraFederationConfigured` 等が oauth2 側の
   client/consent と cross-context 参照している箇所がないか確認し、あれば adapter 境界の
   変換に寄せる（[[ADR-089]] item 5 適用）。

## Tasks

- [ ] T001 [Domain] `shared/spec/federation.go` から WsFederation 型のみを
  `wsfederation/domain/` へ移設し参照更新。ClaimMapping 型は shared/spec に残置。
- [ ] T002 [Kernel] wsfederation が他 context と共有する型を選別。
- [ ] T003 [Persistence] wsfederation 固有 repo 実装を
  `wsfederation/adapters/persistence/{postgres,memory}` へ同居。
- [ ] T004 [Persistence] wsfederation postgres 実装を sqlc 生成へ置換。
- [ ] T005 [DI] `wsfederation/module.go` を新設し Module パターン化。
- [ ] T006 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から wsfederation 分を撤去。
- [ ] T007 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [ ] T008 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

## Verification

- `just verify-go` が green。
- `just test-go` で回帰なし。WS-Federation の passive requestor / WS-Trust の E2E が通る。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- locality 指標：`grep -r "internal/shared/spec" internal/wsfederation | wc -l` が
  ClaimMapping 型参照を除きゼロに近づく。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）でスモーク。

## Risk Notes

- **risk: medium**。純構造変更で被依存も 0 だが、`federation.go` の型分割が
  ClaimMapping との境界を誤ると意図せず ClaimMapping 側の参照を壊す可能性がある。
- 軽減：分割前後で `grep` によりファイル内の型一覧を突合し、移動漏れ・誤移動がないことを
  確認する。ClaimMapping 型は名前を変えず shared/spec に残すことで参照側の破壊を避ける。
