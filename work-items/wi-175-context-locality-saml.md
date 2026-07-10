---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-11
---

# saml コンテキストへバックエンド・コンテキストローカリティを横展開する

## Motivation

[[wi-172]] で確立した [[ADR-089]]・[[ADR-090]]・[[ADR-091]] の型紙を saml context へ
適用する。saml は他 context からの被依存が 0（leaf）であり、業務型が
`internal/shared/spec/saml.go`（113 行）にすでに単独ファイルとして分離されているため、
横展開の中でも実装コストの低い部類に入る。

本 WI は振る舞いを変えない純構造 + 生成方式の変更であり、`spec/contexts/saml.yaml` を
正として双子定義の parity を保つ（SCL 規範は変更しない）。

## Scope

- `internal/shared/spec/saml.go`（113 行）の業務型を `internal/saml/domain/` へ移設。
- saml 固有 repository 実装（`shared/adapters/persistence/{postgres,memory}` の
  `saml_service_providers.go`）を `internal/saml/adapters/persistence/{postgres,memory}`
  へ同居。
- saml の postgres 実装を sqlc 生成へ置換。
- `internal/saml/module.go` を新設し、`Deps`/`bootstrap` から saml 分を Module へ移す。

## Out of Scope

- WsFederation・Scim・Authentication・IdentityManagement・OAuth2・Tenancy 等、他 context
  の型移設。
- memory 二重実装の解消。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. [[wi-172]] と同じ内側→外側の順序で進める。`saml.go` は既に他型と混在していないため、
   domain 移設はほぼ機械的な package rename + import 付け替えになる見込み。
2. `SamlServiceProvider` 等を cross-context で参照している箇所（application context の
   `ApplicationKind` 判定等、[[wi-172]] の T002 で確認済みの参照）を adapter 境界の変換に
   寄せる（[[ADR-089]] item 5 適用）。

## Tasks

- [ ] T001 [Domain] `shared/spec/saml.go` の業務型を `saml/domain/` へ移設し参照更新。
- [ ] T002 [Kernel] saml が他 context と共有する型を選別。
- [ ] T003 [Persistence] saml 固有 repo 実装を `saml/adapters/persistence/{postgres,memory}` へ同居。
- [ ] T004 [Persistence] saml postgres 実装を sqlc 生成へ置換。
- [ ] T005 [DI] `saml/module.go` を新設し Module パターン化。
- [ ] T006 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から saml 分を撤去。
- [ ] T007 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [ ] T008 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

## Verification

- `just verify-go` が green。
- `just test-go` で回帰なし。SAML SSO/SLO の E2E が通る。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- locality 指標：`grep -r "internal/shared/spec" internal/saml | wc -l` がゼロに近づく。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）でスモーク。

## Risk Notes

- **risk: medium**。被依存 0・型が既に単独ファイルであるため構造的リスクは低いが、
  SAML の署名検証・アサーション処理はセキュリティ境界であり、移設時の import 誤りが
  実行時の型不一致に繋がらないことを型検査とテストで担保する必要がある。
- 軽減：`just verify-go`（typecheck）と既存 SAML E2E を各タスク後に都度実行する。
