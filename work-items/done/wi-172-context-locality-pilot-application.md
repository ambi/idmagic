---
status: completed
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

- [x] T001 [Domain] `shared/spec/applications.go` の業務型を `application/domain/` へ移設し参照更新。
- [x] T002 [Kernel] application が他 context と共有する型を `shared/kernel` へ選別（context-map の publishes 基準）。
  実測の結果、`ApplicationRef` / `ApplicationAssignmentRef`（`spec/scl.yaml` context_map の
  publishes）は `TenantRef` 同様に具象 Go 型を持たず、tenantID/applicationID の scalar
  引き回しで表現済み。新規 `shared/kernel` package は本パイロットでは作成不要と判断した。
  oauth2 / saml / wsfederation / bootstrap / shared/support の cross-context 参照は
  adapter・usecase 境界のコードであり、`appdomain "internal/application/domain"` を直接
  import する形に統一した（ADR-089 item 5 の「adapter 境界での変換」条項を適用）。
- [x] T003 [Persistence] application 固有 repo 実装を `application/adapters/persistence/{postgres,memory}` へ同居。
  技術基盤の再利用のため `internal/shared/adapters/persistence/postgres` に
  `RowScanner`(旧 rowScanner) を export、`internal/shared/adapters/persistence/memory` に
  `TenantKey`(旧 tenantKey) を export。テスト用に per-context postgres パッケージが
  共有できる `pgtest`(embedded-postgres harness) と `pgfixtures`(Tenant/User/Group/Client
  シードヘルパー) を新設（shared/postgres 自身の内部テストは import cycle のため従来通り
  ローカルヘルパーを維持）。
- [x] T004 [Tooling] sqlc を導入し `justfile` に `sqlc-generate` レシピ追加、`ResilientDB` を `DBTX` ラップ。
  `internal/shared/adapters/persistence/postgres/base.go` の `DB` interface が sqlc pgx/v5 の
  `DBTX`(Exec/Query/QueryRow 同一シグネチャ)を構造的にすでに満たすため、明示ラッパ型は不要
  だった(`ResilientDB`/`*pgxpool.Pool` をそのまま `sqlcgen.New()` に渡せる)。
- [x] T005 [Persistence] application postgres 実装を sqlc 生成へ置換（動的は narg/COALESCE + pgx エスケープハッチ）。
  `queries/*.sql` → `sqlcgen` package。`ListBySubjects`(UNNEST ペア突合せ)のみ sqlc の型解決が
  効かず手書き pgx を維持(エスケープハッチ、詳細は T008)。`just sqlc-generate` 冪等性を確認済み。
- [x] T006 [DI] `application/module.go` を新設し Module パターン化。
  `application.Module` が 7 repo を束ね、`Register`(自 context route 登録)・`Gate`・
  `ClientDisplayNames`(oauth2/saml/wsfederation が消費する published capability の組立)を持つ。
- [x] T007 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から application 分を撤去。
  両構造体とも 7 個の `Application*` field を `Application application.Module` 1 個へ集約。
  `appGate`/`clientDisplayNames` の組立は routes.go から `Module.Gate`/`Module.ClientDisplayNames`
  へ移した。完全な自己登録(bootstrap が Module 一覧を回すだけ)・`Infra` 抽象化・神ファイル分割は
  他 context(oauth2/saml/wsfederation 等)が未移行のため本パイロットの scope 外(横展開 WI で実施)。
  memory backend 起動 + `/health` `/api/admin/applications` 疎通をスモークテスト済み。
- [x] T008 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
  application context 全 26 クエリ中、sqlc 静的生成 25 件(96%)・手書き pgx エスケープハッチ
  1 件(4%、UNNEST ペア突合せの sqlc 型解決制約)。sqlc 継続を支持する実測値を ADR-090 に追記。
- [x] T009 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

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

## Completion

- **Completed At**: 2026-07-11
- **Summary**: application context を対象に ADR-089/090/091 の 4 レバーを貫通実装した。
  業務型を `internal/application/domain/` へ移設し `internal/shared/spec` から排除、
  postgres/memory の repository 実装を `internal/application/adapters/persistence/` へ同居、
  application の postgres 実装を sqlc 生成コード（`sqlcgen` package、動的クエリ 1 件のみ手書き
  pgx エスケープハッチ）へ置換、`application.Module` を新設して中央 `server/routes.go` の
  `Deps` と `bootstrap/deps.go` の `Dependencies` から application 由来 7 field を 1 field
  (`Application application.Module`) へ集約した。副産物として `shared/adapters/persistence/postgres`
  の `RowScanner` / `shared/adapters/persistence/memory` の `TenantKey` を export し、
  per-context postgres テスト向けの再利用可能な `pgtest`(embedded-postgres harness) /
  `pgfixtures`(Tenant/User/Group/Client シード) package を新設した。
  ADR-090 に動的クエリ比率の実測値(96% 静的 / 4% エスケープハッチ)を追記し sqlc 継続を確認した。
- **Affected Guarantees State**: 振る舞い・HTTP route・DB schema・公開 API は不変(純構造 +
  生成方式の変更)。SCL 規範 (`spec/scl.yaml` / `spec/contexts/application.yaml`) は変更していない。
- **Verification Results**:
  - `just yaml-check` — passed (173 files, 259 record ids)
  - `just verify-go`(lint + `go test -race ./...`) — passed, 0 lint issues
  - `just verify`(yaml-check + verify-go + verify-ui) — passed, exit 0
  - `just sqlc-generate` — 冪等性を sha256 比較で確認（2 回目の生成で差分なし）
  - locality 指標: `grep -r "internal/shared/spec" internal/application | wc -l` は 11
    (残存参照はすべて application 外の型: `DomainEvent`/`NewUUIDv4`/`Tenant`/`User`/
    `ClientType`/`GrantType`/`ClaimMappingRule`/`SamlServiceProvider`/`WsFedRelyingParty` 等で
    本 WI の scope 外。application 自身の業務型はゼロ)
  - memory backend の起動 + `/health`・`/api/admin/applications` 疎通スモークテスト — passed
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Claude Code
  - 対象ソース版: `main`（コミット前）
  - 保存先: CI 外部成果物なし。上記コマンドの成功結果を本記録に要約。
