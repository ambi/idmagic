---
status: pending
authors: [tn]
risk: low
created_at: 2026-07-11
---

# scim コンテキストへバックエンド・コンテキストローカリティを横展開する

## Motivation

[[wi-172]] で確立した [[ADR-089]]・[[ADR-090]]・[[ADR-091]] の型紙を scim context へ
適用する。scim は他 context からの被依存が 0（leaf）で、かつドメイン層は既に
`internal/scim/domain/scim_models.go` として部分的に独立済みであり、
`internal/shared/spec` には ScimUser/ScimGroup 相当の型が存在しない（確認済み）。
そのため本 WI は 7 件中最も scope が小さく、永続化同居 + sqlc 化と module.go が主作業になる。

本 WI は振る舞いを変えない純構造 + 生成方式の変更であり、`spec/contexts/scim.yaml` を
正として双子定義の parity を保つ（SCL 規範は変更しない）。

## Scope

- `internal/scim/domain/scim_models.go` が `internal/shared/spec` の型を参照している
  箇所があれば整理する（domain 移設自体は既に完了しているため、残存参照の解消が中心）。
- scim 固有 repository 実装（`shared/adapters/persistence/{postgres,memory}` の
  `scim.go`）を `internal/scim/adapters/persistence/{postgres,memory}` へ同居。
- scim の postgres 実装を sqlc 生成へ置換。
- `internal/scim/module.go` を新設し、`Deps`/`bootstrap` から scim 分を Module へ移す。

## Out of Scope

- Tenancy（`TenantRef`）・IdentityManagement（`UserRef`）等、scim が depends_on する
  他 context の型移設（`spec/scl.yaml` context_map の scim エントリ: publishes
  `ScimUserRef`/`ScimGroupRef`、depends_on Tenancy/IdentityManagement 参照）。
- 他 context の型移設。
- memory 二重実装の解消。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. [[wi-172]] と同じ内側→外側の順序で進めるが、domain 層は既存の
   `internal/scim/domain/scim_models.go` を出発点とし、T001 では移設ではなく
   「残存する shared/spec 参照の棚卸しと解消」を行う。
2. 永続化同居 + sqlc 化・module.go 新設は他 WI と同型の作業。

## Tasks

- [ ] T001 [Domain] `internal/scim/domain` 内の `shared/spec` 残存参照を棚卸しし、
  scim 自身の型でないもの（Tenant/User 等）は published language 経由の参照に整理する。
- [ ] T002 [Persistence] scim 固有 repo 実装を `scim/adapters/persistence/{postgres,memory}` へ同居。
- [ ] T003 [Persistence] scim postgres 実装を sqlc 生成へ置換。
- [ ] T004 [DI] `scim/module.go` を新設し Module パターン化。
- [ ] T005 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から scim 分を撤去。
- [ ] T006 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [ ] T007 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

## Verification

- `just verify-go` が green。
- `just test-go` で回帰なし。SCIM user/group provisioning の E2E が通る。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- locality 指標：`grep -r "internal/shared/spec" internal/scim | wc -l` が Tenant/User
  参照を除きゼロに近づく。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）でスモーク。

## Risk Notes

- **risk: low**。被依存 0、domain 層は既に部分移設済みで scope が最小。
  主リスクは sqlc 化時の filter クエリ（SCIM の `filter` query parameter による属性検索）
  が動的 WHERE を要求する可能性があり、[[ADR-090]] のエスケープハッチ判断が必要になる点。
- 軽減：T003 で filter クエリの実装状況を確認し、動的な場合は早めに手書き pgx
  エスケープハッチへ倒す（narg/COALESCE で吸収できない場合）。
