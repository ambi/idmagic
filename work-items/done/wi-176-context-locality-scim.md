---
status: completed
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

- [x] T001 [Domain] `internal/scim/domain` 内の `shared/spec` 残存参照を棚卸しし、
  scim 自身の型でないもの（Tenant/User 等）は published language 経由の参照に整理する。
- [x] T002 [Persistence] scim 固有 repo 実装を `scim/adapters/persistence/{postgres,memory}` へ同居。
- [x] T003 [Persistence] scim postgres 実装を sqlc 生成へ置換。
- [x] T004 [DI] `scim/module.go` を新設し Module パターン化。
- [x] T005 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から scim 分を撤去。
- [x] T006 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [x] T007 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

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

## Completion

- **Completed At**: 2026-07-11
- **Summary**: `scim_models.go` は元々 `shared/spec` 非依存で domain 移設は不要と確認した。
  `shared/adapters/persistence/{postgres,memory}` の SCIM token/user-ref/group-ref
  repository を `backend/scim/adapters/persistence/{postgres,memory}` へ同居し、postgres 側
  の全 12 クエリを sqlc 生成（`sqlcgen`）へ置換した。`scim/module.go` を新設して
  `Deps`/`bootstrap` から `ScimRepo` フィールドを撤去し、`scim.Module{Repo: ...}` 経由の
  DI・ルート登録に統一した（IdentityManagement の HTTP handler が必要とする `ScimRepo` は
  `d.Scim.Repo` として同一の port 型で受け渡す）。SCIM `filter` query parameter は
  UserRepository 側のアプリケーション内フィルタで処理されており、scim 自身の repository に
  可変 WHERE クエリは存在しなかった。振る舞い・HTTP route・DB schema・公開 API は変更していない。
- **Affected Guarantees State**: preserved
- **Verification Results**:
  - `just format-go` / `just build-go` — passed
  - `just test-go` / `just verify-go`（lint 0 issues, race 含む全テスト green） — passed
  - `just yaml-check` / `just check-ids` — passed（184 file / 272 record id OK）
  - `just sqlc-generate` — passed（再実行後の生成差分なし）
  - memory backend での起動スモーク（`/health` 200、未登録トークンで SCIM endpoint 401）— passed
  - locality 指標：`grep -r "shared/spec" backend/scim --include="*.go" | grep -v _test.go` は
    `module.go`（`spec.DomainEvent` 型のみ）・`usecases.go`（IdentityManagement 未移設の
    User/Group/DomainEvent 参照）の 2 件のみで、scim 自身の型の残存参照はゼロ。
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: macOS local workspace, Go 1.26.5
  - 実行主体: Claude Code
  - 対象ソース版: `bf5989cb`（コミット前、work-items/wi-176 着手時点）
  - 保存先: 外部成果物なし。静的 sqlc query は 12 件、動的エスケープは 0 件。
