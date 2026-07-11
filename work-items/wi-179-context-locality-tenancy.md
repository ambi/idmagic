---
depends_on: [wi-172-context-locality-pilot-application, wi-173-context-locality-oauth2, wi-174-context-locality-wsfederation, wi-175-context-locality-saml, wi-176-context-locality-scim, wi-177-context-locality-authentication, wi-178-context-locality-identity-management]
status: pending
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

- [ ] T001 [Domain] `shared/spec/tenancy.go` の業務型を `tenancy/domain/` へ移設し参照更新。
- [ ] T002 [Kernel] `TenantRef` を含む tenancy 共有型の扱いを最終確認し、
  adapter 境界変換 or `shared/kernel` 昇格を判定。
- [ ] T003 [Persistence] tenancy 固有 repo 実装を `tenancy/adapters/persistence/{postgres,memory}` へ同居。
- [ ] T004 [Persistence] tenancy postgres 実装を sqlc 生成へ置換。
- [ ] T005 [DI] `tenancy/module.go` を新設し Module パターン化。
- [ ] T006 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から tenancy 分を撤去。
- [ ] T007 [Cross-context] 全 9 依存元 context の `TenantRef`/`Tenant` 型 import path を
  更新（最終波、対象ファイル数が多いため複数コミットに分割してよい）。
- [ ] T008 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [ ] T009 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。
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
