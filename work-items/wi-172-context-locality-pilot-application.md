---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-10
---

# application コンテキストでバックエンド・コンテキストローカリティ回復を貫通実装する（パイロット）

## Motivation

[[ADR-089]]（ドメイン型の per-context 化）・[[ADR-090]]（永続化同居＋sqlc）・
[[ADR-091]]（Module パターン DI/ルーティング）で決めた 4 レバーを、まず低リスクな 1
コンテキストで**貫通実装してパターンを確立**するためのパイロット。以降の横展開 WI が
参照する「型紙」を作る。application を選ぶ理由は、他コンテキストへの被依存が小さく
domain も薄く（現状 `domain/` 1 file）、回帰リスクが最小だから。

本 WI は振る舞いを変えない純構造 + 生成方式の変更であり、`spec/contexts/application.yaml`
を正として双子定義の parity を保つ（SCL 規範は変更しない）。

## Scope

- `internal/shared/spec/applications.go`(400 行) の業務型を `internal/application/domain/` へ移設。
- application 固有 repository 実装（`shared/adapters/persistence/{postgres,memory}` の
  `applications.go` / assignments / icons / categories / orderings / sign-in policy）を
  `internal/application/adapters/persistence/{postgres,memory}` へ同居。
- application の postgres 実装を sqlc 生成へ置換（`internal/application/adapters/persistence/queries/*.sql`）。
- `internal/application/module.go` を新設し、`Deps`/`bootstrap` から application 分を Module へ移す。
- `justfile` に `sqlc-generate` レシピ追加。
- 動的クエリ比率の実測（[[ADR-090]] の sqlc 継続 / bob 切替の判断材料）。

## Out of Scope

- application 以外のコンテキストの移設（後続の横展開 WI）。
- memory 二重実装の解消（testcontainers 退役の是非は別評価）。
- SCL→Go 双子定義のコンテキスト別生成（[[ADR-089]] の将来余地、別 WI）。
- `authorize_handler.go` 等 application 外の神ファイル分割。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. **内側→外側で進める**（AGENTS.md の実装順序）。domain 移設 → ports 確認 →
   persistence 同居 → sqlc 化 → module.go → 中央 Deps/bootstrap から application 分を撤去。
2. **domain 移設**：`shared/spec/applications.go` の型を `application/domain/` へ移し、
   参照を付け替え。他コンテキストが application 型を参照している箇所は published language
   （`shared/kernel`）経由 or adapter 変換に寄せる。`spec/context-map.yaml` の
   application `publishes` を kernel 収録判定の基準にする。
3. **永続化同居 + sqlc**：`ResilientDB` を `DBTX` 実装としてラップ。`queries/*.sql` を
   起こし `just sqlc-generate` で生成。schema 入力は `deploy/schema/postgres.sql`
   （[[ADR-071]]）。動的なもの（admin list/filter）は `sqlc.narg`+`COALESCE`、真に動的な
   ものは薄い手書き pgx をエスケープハッチとして併置。
4. **Module 化**：`application/module.go` に `Module.Register(g, infra Infra)` を実装。
   application の repo/usecase/handler 組立と route 登録を自前化。中央
   `server/routes.go` の `Deps` と `bootstrap/deps.go` から application 由来 field を除去。
5. **実測**：application の全クエリ中、静的（tenant+id 引き）と動的（可変 WHERE）の比率を
   数え、[[ADR-090]] の Considered Alternatives へ追記。
6. 未決定事項：`Infra` の最小構成（どの純技術依存を含めるか）はパイロットで確定し、
   横展開 WI の型紙とする。

## Tasks

- [ ] T001 [Domain] `shared/spec/applications.go` の業務型を `application/domain/` へ移設し参照更新。
- [ ] T002 [Kernel] application が他 context と共有する型を `shared/kernel` へ選別（context-map の publishes 基準）。
- [ ] T003 [Persistence] application 固有 repo 実装を `application/adapters/persistence/{postgres,memory}` へ同居。
- [ ] T004 [Tooling] sqlc を導入し `justfile` に `sqlc-generate` レシピ追加、`ResilientDB` を `DBTX` ラップ。
- [ ] T005 [Persistence] application postgres 実装を sqlc 生成へ置換（動的は narg/COALESCE + pgx エスケープハッチ）。
- [ ] T006 [DI] `application/module.go` を新設し Module パターン化。
- [ ] T007 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から application 分を撤去。
- [ ] T008 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [ ] T009 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

## Verification

- `just verify-go`（format-check / lint / typecheck / build）が green。
- `just test-go`（および -race 相当）で回帰なし。application の E2E / 単体が通る。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等（再生成で差分が出ない）。
- locality 指標：`grep -r "internal/shared/spec" internal/application | wc -l` が 0 に近づく
  （application が自 domain と `shared/kernel`・純技術アダプタのみ参照）。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）と `just dev` でスモーク。

## Risk Notes

- **risk: medium**。純構造 + 生成方式変更で振る舞いは不変だが、(a) sqlc 新規導入の
  ビルド/CI 統合、(b) `shared/spec` からの型移設に伴う広範な import 付け替え、
  (c) 中央 `Deps`/bootstrap の解体で他コンテキストの配線に波及、の 3 点が主リスク。
- 軽減：application は被依存が小さい context を選定。段階ごとに `just verify-go` /
  `just test-go` を挟み、各タスクを小さくコミット可能な粒度に保つ。sqlc 生成物は
  レビュー可能な `.sql` を正とし、`ResilientDB` ラップで既存の circuit breaker/timeout を
  温存する（挙動不変を担保）。
- パイロットで sqlc の動的クエリ負担が想定超なら [[ADR-090]] の分岐に従い bob 切替を
  再評価する（本 WI 内で結論を出さず、実測値を ADR に残す）。
