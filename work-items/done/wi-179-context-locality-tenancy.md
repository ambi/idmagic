---
depends_on: [wi-172-context-locality-pilot-application, wi-173-context-locality-oauth2, wi-174-context-locality-wsfederation, wi-175-context-locality-saml, wi-176-context-locality-scim, wi-177-context-locality-authentication, wi-178-context-locality-identity-management]
status: completed
authors: [tn]
risk: critical
created_at: 2026-07-11
---

# tenancy コンテキストへバックエンド・コンテキストローカリティを横展開する

## Motivation

[[wi-172]] で確立した [[ADR-089]]・[[ADR-090]]・[[ADR-091]] の型紙を tenancy context へ
適用する、横展開シリーズの最終 WI。tenancy は `spec/scl.yaml` context_map 上で 9 context
から被依存を持つ最も基盤的な context であり、`TenantRef`/tenant_id は事実上すべての
context・すべての永続化クエリに登場する。他の全横展開 WI（[[wi-173]]〜[[wi-178]]）の
完了後、最後に着手することを推奨する。

本 WI は振る舞いを変えない純構造 + 生成方式の変更であり、`spec/contexts/tenancy.yaml` を
正として双子定義の parity を保つ（SCL 規範は変更しない）。ただし被依存の広さから、
実質的にはリポジトリ全体の import 文に影響しうる点で他の横展開 WI と性質が異なる。

## Scope

- `internal/shared/spec/tenancy.go`（36 行）の業務型を `internal/tenancy/domain/` へ移設。
- tenancy 固有 repository 実装（`shared/adapters/persistence/{postgres,memory}` の
  `tenants.go` / `tenant_salt_store.go`（+test））を
  `internal/tenancy/adapters/persistence/{postgres,memory}` へ同居。
- tenancy の postgres 実装を sqlc 生成へ置換。
- `internal/tenancy/module.go` を新設し、`Deps`/`bootstrap` から tenancy 分を Module へ移す。
- 全 9 依存元 context（本シリーズで per-context 化済みの Application/OAuth2/WsFederation/
  Saml/Scim/Authentication/IdentityManagement を含む）が `TenantRef`/`Tenant` 型を
  参照している箇所を adapter 境界の変換に更新（最終波の import 更新）。

## Out of Scope

- ClaimMapping・SigningKeys・System 自体の context 化。
- memory 二重実装の解消。
- 振る舞い・HTTP route・DB schema・公開 API の変更。
- tenant_id をテナント配下の子テーブルへ追加する変更（[[ADR-083]] の tenant_id key
  policy は不変。user_id/client_id 等は既にグローバル一意であり、子テーブルへの
  tenant_id 追加は本 WI のスコープ外）。

## Plan

1. [[wi-172]] と同じ内側→外側の順序で進めるが、**T007（cross-context import 更新）が
   本 WI の中心作業になる**。9 依存元のうち 7 つは前段の横展開 WI で既に per-context 化
   されている想定であり、`TenantRef` の import path 付け替えは機械的だが対象ファイル数が
   多い。
2. `TenantRef` が [[wi-172]] の実測（`shared/kernel` 新設不要、scalar tenantID の
   引き回しで表現済み）と同じ扱いで済むかを T002 で最終確認する。9 被依存という規模の
   違いから、ここで初めて `shared/kernel` 昇格が必要と判明する可能性が最も高い WI でもある。
3. tenancy は認可・データ分離の根幹（マルチテナント境界）であるため、他の横展開 WI 以上に
   段階ごとの `just test-go`（マルチテナント分離のテストケースを含む）を挟む。

## Tasks

- [x] T001 [Domain] `shared/spec/tenancy.go` の業務型を `tenancy/domain/` へ移設し参照更新。
  `Tenant`/`TenantStatus`/`PasswordPolicyOverride`/`tenantSchema` を `tenancy/domain/tenancy.go`
  へ移設し `spec.Validate`/`spec.ZogError` 経由で ADR-093 の型別 zog schema パターンを踏襲。
  `shared/spec` からは `Tenant*` 型を完全排除（`enums.go`/`validation.go`/
  `wi129_coverage_test.go` を trim、カバレッジは `tenancy/domain/tenancy_test.go` へ移設）。
- [x] T002 [Kernel] `TenantRef` を含む tenancy 共有型の扱いを最終確認し、
  adapter 境界変換 or `shared/kernel` 昇格を判定。
  `Tenant`/`TenantStatus`/`PasswordPolicyOverride` は他シリーズ WI と同じく
  `tenancydomain "tenancy/domain"` の cross-context 直接 import で足りた。ただし
  `DefaultTenantID`/`DefaultRealm` の 2 定数は `shared/spec/policy.go`（AuthZEN 述語
  `actor_is_control_plane_user`）からも参照されており、`tenancy/domain` が
  `spec.Validate` を使うため `shared/spec` → `tenancy/domain` の import は循環になる。
  この 2 定数のみ `shared/kernel`（新設）へ昇格し、`tenancy/domain.DefaultTenantID`/
  `DefaultRealm` はそこからの re-export とした。ADR-089 が却下代替案で触れた
  「真に共有される published language（`TenantRef` 等）」が実際に kernel 昇格を要した
  本シリーズ初の具体例。
- [x] T003 [Persistence] tenancy 固有 repo 実装を `tenancy/adapters/persistence/{postgres,memory}` へ同居。
  `shared/adapters/persistence/{postgres,memory}/tenants.go`(+test) を移設。
  `tenant_salt_store.go`（`shared/adapters/crypto` と `shared/adapters/persistence/postgres`）は
  oauth2 が所有する `TenantSaltStore` port の実装であり `Tenant` 型を参照しないため
  スコープ外と判断し移設しなかった（WI 本文の記載は不正確だったため実態を優先）。
  `pgfixtures.SeedTenant` を新実装へ向け直し、`shared/adapters/persistence/postgres` 自身の
  内部テスト（import cycle 制約により pgfixtures 不可）は生 SQL の自前 seed ヘルパーへ変更。
- [x] T004 [Persistence] tenancy postgres 実装を sqlc 生成へ置換。
  `sqlc.yaml` に tenancy ブロックを追加し `queries/tenants.sql`（4 クエリ）から生成。
  動的エスケープハッチ 0 件（詳細は T008 / ADR-090）。
- [x] T005 [DI] `tenancy/module.go` を新設し Module パターン化。
  `identitymanagement.Module` と同型の「port の束」（`TenantRepo`/`AttrSchemaRepo`）。
- [x] T006 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から tenancy 分を撤去。
  `Dependencies`/`server.Deps` の `TenantRepo`/`AttrSchemaRepo` を `Tenancy tenancy.Module`
  1 field へ集約。`server.Deps` 側は既存 2 モジュールと同じ `mergeLegacyTenancyDeps` 互換
  ブリッジを追加。`support.Deps.TenantRepo`（Authenticator 同様の横断 leaf port）は
  スコープ外のまま維持（bootstrap 側の供給元のみ `deps.Tenancy.TenantRepo` に更新）。
- [x] T007 [Cross-context] 全 9 依存元 context の `Tenant`/`TenantStatus`/`DefaultTenantID`/
  `DefaultRealm`/`PasswordPolicyOverride` 参照 93 ファイルを機械的に `tenancydomain.*`
  （tenancy 自身の 10 ファイルは無 alias `domain.*`）へ一括更新。加えて
  `memory.NewTenantRepository`/`sharedpg.TenantRepository` 等、旧 `shared/adapters/persistence`
  の実体を直接参照していた 15 ファイル（bootstrap・oauth2・scim・identitymanagement・
  authentication の postgres/memory テスト、`pgfixtures.go`）を新パッケージ参照へ更新。
- [x] T008 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
  Tenant 全 4 クエリ、動的エスケープハッチ 0 件（100% 静的）。application (wi-172) から
  本 WI まで全 7 WI 通じて 0〜4% に収まり続けたため sqlc 継続採用を確定と記録。
- [x] T009 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。
  加えて全 context のマルチテナント分離テストを横断実行する。

## Verification

- `just verify-go` が green。
- `just test-go` で回帰なし。テナント CRUD、salt store、全 context のマルチテナント
  分離テスト（cross-tenant アクセス拒否）が通る。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- locality 指標：`grep -r "internal/shared/spec" internal/tenancy | wc -l` がゼロに
  近づく。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）と `just dev` で
  全 API のスモークテスト。

## Risk Notes

- **risk: critical**。9 context からの被依存を持つ最基盤 context であり、(a) import
  波及範囲がリポジトリ全体に及ぶ、(b) tenant 境界はマルチテナント SaaS のセキュリティ
  境界そのものであり、移設ミスがテナント分離の回帰に直結する、(c) 本シリーズの最終 WI
  として他 6 WI の移設結果に依存するため、先行 WI の遅延・変更がそのまま本 WI の
  見積りに影響する、の 3 点が主リスク。
- 軽減：他の全横展開 WI（[[wi-173]]〜[[wi-178]]）完了後に最後に着手する。T007 は
  依存元 context ごとに分割コミットし、各コミット後に `just test-go` を実行する。
  マルチテナント分離テストを Verification に明示し、通常の機能回帰確認より厳格に扱う。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: tenancy context を対象に ADR-089/090/091 の型紙を横展開シリーズ最終 WI として
  貫通実装した。`Tenant`/`TenantStatus`/`PasswordPolicyOverride` 業務型と、それらの zog field
  validation schema を `internal/shared/spec` から `tenancy/domain/` へ移設し、`shared/spec`
  からは完全排除。`DefaultTenantID`/`DefaultRealm` の 2 定数は `shared/spec` の AuthZEN 述語
  からも参照される import cycle 制約により `shared/kernel`（新設）へ昇格し、`tenancy/domain`
  はそこからの re-export とした——本シリーズで初めて実際に kernel 昇格を要した具体例。
  postgres/memory の repository 実装（`tenants.go`）を `tenancy/adapters/persistence/` へ
  同居し、postgres 実装は sqlc 生成コード（動的クエリ 0 件、100% 静的）へ置換。
  `tenancy.Module`（`identitymanagement.Module` と同型の「port の束」）を新設し、中央
  `bootstrap/deps.go` の `Dependencies`・`shared/adapters/http/server/routes.go` の `Deps`
  から `TenantRepo`/`AttrSchemaRepo` を集約（`server.Deps` は既存モジュール同様の legacy
  互換ブリッジ付き）。9 依存元 context・bootstrap・shared/adapters 配下の計 100 以上の
  ファイルを `tenancydomain.*`（tenancy 自身は無 alias `domain.*`）へ機械的に更新した。
- **Affected Guarantees State**: 振る舞い・HTTP route・DB schema・公開 API は不変
  (純構造 + 生成方式の変更)。SCL 規範 (`spec/scl.yaml` / `spec/contexts/tenancy.yaml`) は
  変更していない。`tenant_salt_store.go`（oauth2 所有の `TenantSaltStore` port 実装）は
  `Tenant` 型を参照しないため Scope の記載と異なり移設対象外とした。
- **Verification Results**:
  - `just yaml-check`(Architecture 整合検査含む) — passed (188 files, 276 record ids)
  - `just verify-go`(lint + `go test -race ./...` 相当。本セッションでは
    `go build ./backend/...` / `go vet ./backend/...` / `just lint-go` / `just test-go` を
    個別に実行して確認) — 全 green、0 lint issues
  - `just sqlc-generate` — 冪等性を確認（2 回目の生成で新規/差分ファイルなし）
  - locality 指標: `grep -r "shared/spec" tenancy | wc -l` は 9
    （残存参照はすべて tenancy 外の技術基盤: `spec.Validate`/`spec.ZogError` 汎用ラッパー、
    `spec.SCL`/`spec.DomainEvent` 等。tenancy 自身の業務型はゼロ。identitymanagement
    (wi-178) の 24 件と同種の残存パターン）
  - `tenancy/adapters/persistence/postgres` の実 embedded-postgres 実行
    （Tenant CRUD round-trip、FindByID missing ケース）— passed
  - `oauth2/usecases`（`tenant_isolation_test.go`/`tenant_key_health_test.go`）含む
    全 context のマルチテナント分離テスト — `just test-go` に含めて全 green
  - memory backend の起動 + `/health`・`/.well-known/openid-configuration` 疎通
    スモークテスト — passed
  - postgres_valkey backend のフルスタック（docker compose）起動スモークは、
    identitymanagement (wi-178) と同じ理由（Docker 不使用の方針）で本セッションでは
    実施せず。sqlc クエリ自体は embedded-postgres 経由の実 SQL round-trip で検証済み。
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Claude Code
  - 対象ソース版: `main`（コミット前）
  - 保存先: CI 外部成果物なし。上記コマンドの成功結果を本記録に要約。
