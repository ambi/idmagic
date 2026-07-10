---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-11
---

# oauth2 コンテキストへバックエンド・コンテキストローカリティを横展開する（client / consent / 認可詳細タイプ）

## Motivation

[[wi-172]]（application context パイロット）で [[ADR-089]]（ドメイン型の per-context 化）・
[[ADR-090]]（永続化同居＋sqlc）・[[ADR-091]]（Module パターン DI/ルーティング）の 3 ADR を
貫通実装し型紙を確立した。本 WI はその型紙を oauth2 context へ適用する第一弾横展開である。

oauth2 は `spec/scl.yaml` context_map 上で他 context からの被依存が 0（leaf）だが、
`internal/shared/spec` / `internal/shared/adapters/persistence` 双方で最大規模を占める
context であり、横展開の中でも最も工数の大きい 1 件になる見込みだった。

**T001（domain 移設の実測）の結果、`shared/spec/oauth2.go`（292 行）の 14 業務型が
92 ファイル・340 箇所から参照されており、wi-172（95 ファイル変更のパイロット）を上回る
規模と判明した**。Plan に明記していた分岐に従い、client / token / audit・outbox の
3 分割に切り出した（[[wi-181]] が token/grant、[[wi-182]] が audit/outbox を担当）。
本 WI は縮小後の Scope として client / consent / authorization detail type のみを扱う。

本 WI は振る舞いを変えない純構造 + 生成方式の変更であり、`spec/contexts/oauth2.yaml` を
正として双子定義の parity を保つ（SCL 規範は変更しない）。

## Scope

- `internal/shared/spec/oauth2.go`（292 行）のうち client / consent / 認可詳細タイプ系
  業務型（`OAuth2Client` / `Consent` / `AuthorizationDetailFieldRule` /
  `AuthorizationDetailsSchema` / `AuthorizationDetailType`）を `internal/oauth2/domain/`
  へ移設。**`AuthorizationDetail`（実行時インスタンス、`AuthorizationRequest` /
  `AuthorizationCodeRecord` / `AccessTokenClaims` からも参照される）は [[wi-181]] 側の
  型が主に依存するため本 WI では移設せず shared に残す**（T002 実測で判明、下記 Plan 参照）。
- oauth2 固有 repository 実装（`shared/adapters/persistence/{postgres,memory}` の
  `clients.go` / `consents.go` / `authorization_detail_types.go`）を
  `internal/oauth2/adapters/persistence/{postgres,memory}` へ同居。
- 上記 postgres 実装を sqlc 生成へ置換（動的クエリはエスケープハッチ、[[ADR-090]] 準拠）。
- `internal/oauth2/module.go` を新設し、`ClientRepo` / `ConsentRepo` /
  `AuthzDetailTypeRepo` を Module へ移す（[[wi-181]]・[[wi-182]] がこの Module を
  拡張していく前提の初回導入）。`Deps`/`bootstrap` から該当 3 field を撤去。

## Out of Scope

- token/grant 系型・valkey backed store・`authorize_handler.go` 分割
  （[[wi-181]] で扱う）。
- audit event / outbox（[[wi-182]] で扱う）。
- SigningKeys 関連（`shared/adapters/persistence/postgres/keys.go`、`ports.SigningKey`）。
  SigningKeys はまだ独立 context（`internal/signingkeys/` 相当）を持たないため、本 WI では
  shared に残置する。SigningKeys 自身を context 化するかは別途評価する。
- ClaimMapping・Authentication・IdentityManagement・Tenancy 等、他 context の型移設
  （[[wi-174]]〜[[wi-179]] で扱う）。
- memory 二重実装の解消（testcontainers 退役の是非は別評価、[[ADR-090]] 決定 6 を参照）。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. [[wi-172]] と同じ内側→外側の順序（domain → ports → persistence → sqlc → module.go →
   中央 Deps/bootstrap 撤去）で進める。
2. sqlc 化にあたり、client 一覧の admin filter 等の可変 WHERE を持つクエリが含まれる
   可能性がある。[[ADR-090]] の「動的比率が支配的なら bob へ切替」の再評価トリガーに
   なりうるため、実測値を [[ADR-090]] へ追記する。
3. `internal/oauth2/module.go` は [[wi-181]]・[[wi-182]] が後続で拡張する前提で設計する
   （Module 構造体へのフィールド追加のみで済むよう、Client/Consent 固有の組み立てロジックを
   自己完結させる）。
4. **T002/T003 実測の結果、`ClientType`/`GrantType`/`ResponseType` は shared に残置する**。
   理由は 2 点：(a) `shared/spec/policy.go` の SCL permissions 評価エンジン
   （`AuthZSubjectProps.ClientType`/`GrantTypes`）が全 context 横断で直接参照する、
   (b) `ResponseType` は [[wi-181]] 側に残る `AuthorizationRequest` が参照する。
   同じ理由で `AuthorizationDetail`（実行時インスタンス）も shared 残置とする。
   `oauth2/domain` はこれら shared 側の型を import する一方向依存とし、shared 側は
   oauth2/domain を import しない（循環回避）。`grants.go`（`GetGrantSpec` 等）は
   依存する 3 型がすべて shared に残るため本 WI では移設しない。

## Tasks

- [x] T001 [Domain] `shared/spec/oauth2.go` の移設規模を実測。1 WI で収まらないと判明し
  [[wi-181]]・[[wi-182]] へ分割（本タスクの成果として分割を実施済み）。
- [x] T002 [Domain] client / consent / 認可詳細タイプ系業務型を `oauth2/domain/` へ移設し
  参照更新。`OAuth2Client` / `Consent` / `AuthorizationDetailFieldRule` /
  `AuthorizationDetailsSchema` / `AuthorizationDetailType` と関連 zog スキーマ・イベント
  （`ClientRegistered` / `AdminOAuth2Client{Created,Updated,Deleted}` /
  `ConsentGrantedEvent` / `ConsentRevokedEvent`）を移設。zog 検証は [[ADR-093]] に従い
  `spec.Validate`/`spec.ZogError` をエクスポートして再利用。移設規模: 65 ファイル修正
  （うち 3 ファイルは Validate()/enum カバレッジテストの移動）。
- [x] T003 [Kernel] oauth2 が他 context と共有する型を選別（context-map の publishes 基準）。
  結果: `shared/kernel` の新設は不要（[[wi-172]] と同じ結論）。`ClientType`/`GrantType`/
  `ResponseType` は `shared/spec/policy.go`（SCL permissions 評価エンジン）と
  [[wi-181]] 側に残る `AuthorizationRequest` 等が直接参照するため shared に残置し、
  `oauth2/domain` から一方向 import する形とした（循環回避、詳細は Plan 4 参照）。
- [x] T004 [Persistence] `clients.go` / `consents.go` / `authorization_detail_types.go` を
  `oauth2/adapters/persistence/{postgres,memory}` へ同居。postgres 側は新設と同時に
  sqlc 化したため T005 を本タスクで合わせて実施（下記）。`shared/adapters/persistence/memory`
  の `defaultTenant` を `DefaultTenant` としてエクスポートし per-context adapter から再利用
  （[[wi-172]] の RowScanner/TenantKey と同じパターン）。`pgfixtures.SeedClient` の参照先を
  新 `oauth2/adapters/persistence/postgres.OAuth2ClientRepository` へ更新。postgres の
  round-trip テストは import cycle 回避のため oauth2 パッケージ自身に自前の
  seedTenant/seedUser/seedClient ヘルパーを持たせて移設（shared 側の `fixtures_test.go` の
  `seedClient` は FK 充足専用の生 SQL ヘルパーへ縮小）。
- [x] T005 [Persistence] 上記 postgres 実装を sqlc 生成へ置換（動的はエスケープハッチ）。
  `sqlc.yaml` に oauth2 用の 2 個目の `sql:` エントリを追加し
  `oauth2/adapters/persistence/postgres/{queries,sqlcgen}` を新設。3 リポジトリ 9 クエリ
  すべて静的生成（エスケープハッチなし、動的 WHERE を持つクエリが無いため）。
  `just sqlc-generate` の冪等性を確認済み。
- [ ] T006 [DI] `oauth2/module.go` を新設し Client/Consent/AuthzDetailType を Module 化。
- [ ] T007 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から該当 3 field を撤去。
- [ ] T008 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [ ] T009 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

## Verification

- `just verify-go`（format-check / lint / typecheck / build）が green。
- `just test-go` で回帰なし。client 管理・consent フローの E2E / 単体が通る。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- locality 指標：`grep -r "internal/shared/spec" internal/oauth2 | wc -l` が本 WI 対象外
  （token/grant・audit・SigningKeys 等）の参照を除き減少していることを確認。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）と `just dev` でスモーク。

## Risk Notes

- **risk: medium**（当初 high から縮小後に見直し）。client/consent は認可コアの
  token/grant フローそのものではないが、client 認証情報の scope が広い（admin API・
  token issuance 双方から参照される）ため純粋な低リスクではない。
- 軽減：[[wi-172]] で確立した型紙をそのまま踏襲し、各タスクを小さくコミット可能な粒度に
  保つ。`oauth2.Module` の設計を [[wi-181]]・[[wi-182]] が拡張しやすい形にすることで
  後続 WI のやり直しリスクを下げる。振る舞い不変を `just test-go`（既存 E2E 含む）で
  都度確認する。
