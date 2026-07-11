---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-11
---

# identity-management コンテキストへバックエンド・コンテキストローカリティを横展開する

## Motivation

[[wi-172]] で確立した [[ADR-089]]・[[ADR-090]]・[[ADR-091]] の型紙を identity-management
context へ適用する。identity-management は `spec/scl.yaml` context_map 上で 7 context
から被依存を持つ、Tenancy（[[wi-179]]）に次いで基盤性の高い context である。User/Group/
Agent は他のほぼ全 context から参照されるため、本 WI は [[wi-173]]〜[[wi-177]] の完了後、
[[wi-179]] の前に着手することを推奨する（Tenancy より被依存が少ないため）。

本 WI は振る舞いを変えない純構造 + 生成方式の変更であり、
`spec/contexts/identity-management.yaml` を正として双子定義の parity を保つ
（SCL 規範は変更しない）。

## Scope

- `internal/shared/spec/users.go`（250 行）・`groups.go`（75 行、+test）・
  `agents.go`（60 行）・`attributes.go`（183 行、+`user_attributes_test.go` 241 行）の
  業務型を `internal/identitymanagement/domain/` へ移設。
- identity-management 固有 repository 実装（`shared/adapters/persistence/{postgres,memory}`
  の `users.go` / `groups.go` / `agents.go` / `tenant_user_attribute_schema.go`）を
  `internal/identitymanagement/adapters/persistence/{postgres,memory}` へ同居。
- identity-management の postgres 実装を sqlc 生成へ置換。
- `internal/identitymanagement/module.go` を新設し、`Deps`/`bootstrap` から
  identity-management 分を Module へ移す。
- 既に per-context 化済みの 7 依存元 context（[[wi-173]]〜[[wi-177]] で移設済みの
  OAuth2/WsFederation/Saml/Scim/Authentication、および未移設の ClaimMapping/SigningKeys
  相当の参照）が User/Group/Agent 型を参照している箇所を adapter 境界の変換に更新
  （第 2 波の import 更新）。

## Out of Scope

- Tenancy の型移設（[[wi-179]]）。
- ClaimMapping・SigningKeys 自体の context 化（まだ独立 package を持たないため）。
- memory 二重実装の解消。
- 振る舞い・HTTP route・DB schema・公開 API の変更。

## Plan

1. [[wi-172]] と同じ内側→外側の順序で進める。
2. 被依存 7 context は本 WI 開始時点で一部が per-context 化済み（[[wi-173]]〜[[wi-177]]）、
   一部が未移設（ClaimMapping/SigningKeys は元々 context 化していないため shared のまま）。
   T007 の「第 2 波」更新は per-context 化済みの依存元のみを対象とし、shared に残る
   依存元は通常の import 付け替えで扱う。
3. `UserRef`/`GroupRef`/`AgentRef` 等の published language 相当が
   `spec/scl.yaml` context_map の publishes に明示されているかを確認し、
   `shared/kernel` 昇格の要否を T002 で判定する（[[wi-172]] 実測では不要と判断されたが、
   7 被依存という規模から再評価が必要）。

## Tasks

- [x] T001 [Domain] `shared/spec/users.go` / `groups.go` / `agents.go` / `attributes.go` の
  業務型を `identitymanagement/domain/` へ移設し参照更新。
  ADR-093 に従い型別 zog schema (`userSchema`/`groupSchema`/`groupMemberSchema`/
  `agentSchema`/`agentCredentialBindingSchema`/`userAttributeDefSchema`) と、
  identity-management 固有 enum (`UserStatus`/`RequiredAction`/`AttributeType`/
  `AttrVisibility`/`AgentKind`/`AgentStatus`、旧 `shared/spec/enums.go`) も型と共に移設。
  `shared/spec` には `Validate`/`ZogError` 汎用ラッパー経由でのみ依存。
  リポジトリ全体で `spec.User`/`spec.Group`/`spec.Agent` 等の参照 103+1 ファイルを
  `idmdomain.*` へ機械的に置換（第 2 波 T007 相当を前倒しで実施、詳細は T007 参照）。
- [x] T002 [Kernel] identity-management が 7 context と共有する型を選別し、
  adapter 境界変換 or `shared/kernel` 昇格を判定。
  `spec/scl.yaml` context_map の publishes (`UserRef`/`GroupRef`/`AgentRef`/`EffectiveRoles`) は
  application/wi-172 の `TenantRef` 同様、具象 Go 型を持たず scalar ID 引き回しで表現される
  想定だが、実装は `User`/`Group`/`Agent` 集約そのものを cross-context で必要とする箇所が
  複数あり (例: `wsfederation/domain/attributes.go` の `ResolveUserAttributes(u idmdomain.User)` は
  claim 発行のため属性値そのものを要求)。この依存は本 WI 以前から `shared/spec.User` として
  既に存在していた実質的な cross-context 依存であり、今回の移設で `idmdomain.User` に
  置き換わっただけで新規の結合ではない。新規 `shared/kernel` package を起こすと
  Go 型として何も対応しない抽象 Ref 型を作ることになり、`spec/scl.yaml` の publishes 定義とも
  乖離するため、wi-172〜wi-177 の判断と同様に見送り、adapter/domain 境界での
  `idmdomain "identitymanagement/domain"` 直接 import に統一した。
- [x] T003 [Persistence] identity-management 固有 repo 実装を
  `identitymanagement/adapters/persistence/{postgres,memory}` へ同居。
  postgres 側は既存の hand-written SQL のまま移設（sqlc 化は T004 で実施）。
  `shared/adapters/persistence/postgres/pgfixtures` の `SeedUser`/`SeedGroup` を
  identitymanagement の repository 実装に向け直した。identitymanagement 自身の
  postgres テストパッケージは pgfixtures を import すると
  postgres -> pgfixtures -> postgres の import cycle になるため、
  `shared/adapters/persistence/postgres` 自身の内部テスト (wi-172 以前からの既存パターン) と
  同じくローカル fixture ヘルパーを新設した。同じ制約が scim/oauth2 の内部 postgres
  テストにも既に存在していたため、それらも `idmpg.UserRepository`/`idmpg.GroupRepository`
  参照へ更新した。
- [x] T004 [Persistence] identity-management postgres 実装を sqlc 生成へ置換。
  `sqlc.yaml` に identitymanagement ブロックを追加し、`queries/{users,groups,agents,
  tenant_user_attribute_schemas}.sql` から `sqlcgen` を生成。SELECT の列順をテーブル定義順に
  揃え、sqlc が per-table 共有モデル (`User`/`Group`/`Agent`/`TenantUserAttributeSchema`) を
  再利用するようにした。動的 WHERE を要するクエリは無く、手書き pgx エスケープハッチは
  0 件（詳細は T008）。`just sqlc-generate` の冪等性を sha256 比較で確認済み。
- [x] T005 [DI] `identitymanagement/module.go` を新設し Module パターン化。
  `authentication.Module` と同じ「port の束」パターンを採用（`Register` 自己登録は持たない）。
  identitymanagement/adapters/http は oauth2/scim/authentication/tenancy 由来の port も
  必要とするため、HTTP route 登録は引き続き中央 `server/routes.go` が組み立てる。
- [x] T006 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から
  identity-management 分を撤去。
  `Dependencies`/`server.Deps` の `UserRepo`/`GroupRepo`/`AgentRepo` を
  `IdentityManagement identitymanagement.Module` へ集約。`server.Deps` 側は
  `authentication.Module`/`oauth2.Module` と同じ `mergeLegacyIdentityManagementDeps`
  互換ブリッジを追加し、直接 `server.Deps{UserRepo: ...}` を構築する既存 20 件以上の
  テストファイルの書き換えを避けた（既存 2 モジュールと同一の移行パターン）。
- [x] T007 [Cross-context] per-context 化済みの依存元 context の User/Group/Agent 型
  import path を更新（第 2 波）。
  T001 のドメイン移設と同時に機械的な repo 全体スイープで実施済み
  （`spec.User`/`spec.Group`/`spec.Agent` 等 103+1 ファイルを `idmdomain.*` へ、
  `memory.New*Repository` 44 ファイルを `idmmemory.*` へ一括更新）。
- [x] T008 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
  identity-management context 全 26 クエリ（users 6、groups 9、agents 8、
  tenant_user_attribute_schemas 3）中、sqlc 静的生成 26 件（100%）・手書き pgx
  エスケープハッチ 0 件。tombstone 除外述語 (`lifecycle->>'status' IS DISTINCT FROM
  'deleted'`) は全クエリ共通の固定述語であり動的 WHERE には該当しない。実測値を
  ADR-090 に追記した。
- [x] T009 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

## Verification

- `just verify-go` が green。
- `just test-go` で回帰なし。user/group/agent CRUD、属性スキーマ検証の E2E が通る。
  加えて依存元 7 context の既存 E2E で cross-context 参照が壊れていないことを確認する。
- `just yaml-check` / `just check-ids` で SCL・双子定義・ID の整合。
- `just sqlc-generate` が冪等。
- locality 指標：`grep -r "internal/shared/spec" internal/identitymanagement | wc -l`
  がゼロに近づく。
- `just build-go`（memory / postgres_valkey 両バックエンド起動）でスモーク。

## Risk Notes

- **risk: high**。7 context からの被依存を持つ基盤 context であり、(a) 移設の
  import 波及範囲が最も広い部類、(b) User/Group/Agent はほぼ全 API のリクエスト経路に
  乗るため回帰の実害が大きい、(c) 属性スキーマ（`attributes.go`）はテナントごとの動的
  スキーマを扱うため sqlc の静的クエリ前提と相性が悪い可能性がある、の 3 点が主リスク。
- 軽減：[[wi-173]]〜[[wi-177]] 完了後に着手し、依存元の大半が既に per-context 化された
  状態で import 更新の見通しを立てやすくする。属性スキーマの動的クエリ有無は T004 で
  早期に確認し、必要なら手書き pgx エスケープハッチへ倒す。各タスク後に依存元 context
  の E2E を横断的に実行する。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: identity-management context を対象に ADR-089/090/091/093 の 4 レバーを
  貫通実装した。`User`/`Group`/`Agent`/`AttributeValue`/`UserAttributeDef`/
  `TenantUserAttributeSchema` 業務型と、それらが所有する zog field validation schema
  (ADR-093)・enum (`UserStatus`/`RequiredAction`/`AttributeType`/`AttrVisibility`/
  `AgentKind`/`AgentStatus`) を `internal/shared/spec` から `identitymanagement/domain/`
  へ移設し、`shared/spec` からは完全排除。postgres/memory の repository 実装を
  `identitymanagement/adapters/persistence/` へ同居し、postgres 実装は sqlc 生成コード
  （動的クエリ 0 件、100% 静的）へ置換。`identitymanagement.Module`（`authentication.Module`
  と同型の「port の束」パターン）を新設し、中央 `bootstrap/deps.go` の
  `Dependencies`・`shared/adapters/http/server/routes.go` の `Deps` から
  `UserRepo`/`GroupRepo`/`AgentRepo` 3 field を `IdentityManagement
  identitymanagement.Module` 1 field へ集約した（`server.Deps` 側は
  `authentication.Module`/`oauth2.Module` と同じ legacy 互換ブリッジを付与）。
  リポジトリ全体で `spec.User` 等参照 103+1 ファイル・`memory.New*Repository` 44 ファイルを
  機械的に `idmdomain.*`/`idmmemory.*` へ置換し、T007（第 2 波 import 更新）を T001 と
  同時に実施した。副産物として `shared/adapters/persistence/postgres/pgfixtures` の
  `SeedUser`/`SeedGroup` を identitymanagement 実装へ向け直し、scim/oauth2 の内部
  postgres テストが直接参照していた旧 `sharedpg.UserRepository`/`GroupRepository` も
  `idmpg.*` へ更新した。ADR-090 に動的クエリ比率の実測値（identity-management: 100%
  静的）を追記し、`ARCHITECTURE.md` の「`identitymanagement` は per-context `domain/` を
  持たない」という記述（now-stale）を是正した。
- **Affected Guarantees State**: 振る舞い・HTTP route・DB schema・公開 API は不変(純構造 +
  生成方式の変更)。SCL 規範 (`spec/scl.yaml` / `spec/contexts/identity-management.yaml`) は
  変更していない。
- **Verification Results**:
  - `just yaml-check`(Architecture 整合検査含む) — passed (186 files, 274 record ids)
  - `just verify-go`(lint + `go test -race ./...`) — passed, 0 lint issues
  - `just verify`(yaml-check + verify-go + verify-ui) — passed, exit 0
  - `just sqlc-generate` — 冪等性を sha256 比較で確認（2 回目の生成で差分なし）
  - locality 指標: `grep -r "shared/spec" identitymanagement | wc -l` は 24
    （残存参照はすべて identitymanagement 外の型・技術基盤: `spec.Validate`/`spec.ZogError`
    汎用ラッパー、`spec.DefaultTenantID`、`spec.Tenant` 等で本 WI の scope 外。
    identitymanagement 自身の業務型はゼロ。application パイロット(wi-172)の 11 件と
    同種の残存パターン）
  - `identitymanagement/adapters/persistence/postgres` の実 embedded-postgres 実行
    （4 テスト: User/Group/Agent/TenantUserAttributeSchema round-trip）— passed
  - memory backend の起動 + `/health`・`/api/admin/users`(401 with invalid token) 疎通
    スモークテスト — passed
  - postgres_valkey backend のフルスタック（docker compose）起動スモークは、
    ユーザー指示により本セッションでは実施せず（Docker 不使用の方針）。sqlc クエリ自体は
    embedded-postgres 経由の実 SQL round-trip で検証済みのため、残存リスクは低いと判断。
    フルスタック起動確認は次回のデプロイ前検証で実施を推奨する。
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Claude Code
  - 対象ソース版: `main`（コミット前）
  - 保存先: CI 外部成果物なし。上記コマンドの成功結果を本記録に要約。
