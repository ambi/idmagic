---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# アプリ権限台帳と SoD 競合ルールを導入する

## Motivation
IdMagic は Application assignment と claim mapping を持つが、アプリ内ロールや権限
entitlement を IdM 側の台帳として管理し、ユーザー・グループに割り当てる機能はない。
Okta Entitlement Management や Microsoft Entra entitlement management では、アプリ、
グループ、アプリ内ロールを台帳化し、Separation of Duties (SoD) で競合する権限の同時保有を
防ぐ。これがないと、SaaS 内部権限の過剰付与や監査説明が難しい。

本 WI は、Application ごとの entitlement catalog、entitlement assignment、SoD conflict
rule を導入し、将来の access request / access review / outbound provisioning と連携できる
権限台帳を作る。

## Scope
- **scl**:
  - `Application` に ApplicationEntitlement / EntitlementAssignment / SeparationOfDutiesRule を追加する。
  - User / Group への entitlement assignment と application assignment の関係を明示する。
  - SoD conflict 検出、assignment 拒否、例外承認 events / scenarios を追加する。
  - `IdentityManagement` の UserRef / GroupRef を assignment subject として利用する。
- **go**:
  - entitlement catalog repository、assignment usecase、SoD evaluator、memory / postgres adapter を実装する。
  - claim mapping / provisioning に渡せる entitlement projection を追加する。
- **http**:
  - entitlement catalog、assignment、SoD rule、conflict preview API を追加する。
- **ui**:
  - Application detail に entitlement 管理、assignment、SoD conflict preview を追加する。
- **documentation**:
  - README に entitlement catalog と SoD rule の運用例を追記する。

## Out of Scope
- ReBAC / FGA 判定エンジン。これは `wi-53-rebac-fine-grained-authorization` が扱う。
- 外部アプリから entitlement を自動発見する connector。
- access request / access review workflow 本体。
- outbound SCIM provisioning 本体。これは `wi-45-outbound-scim-provisioning` が扱う。

## Plan
- Entitlement は Application 境界内の管理対象として置き、User / Group に直接または group 経由で割り当てる。
- SoD rule は「同時保有禁止の entitlement 集合」から始め、複雑な条件式や risk scoring は扱わない。
- assignment 保存前に conflict preview と fail-closed な保存時検証を行う。
- 将来の provisioning 連携のため、entitlement projection は protocol-neutral にする。

## Tasks
- [ ] T001 [SCL] Entitlement catalog、assignment、SoD rule、events / scenarios を追加する。
- [ ] T002 [Decision] entitlement と application assignment の関係、SoD 例外方針を ADR に記録する。
- [ ] T003 [App] catalog / assignment / SoD evaluator を実装する。
- [ ] T004 [HTTP] entitlement と SoD 管理 API を追加する。
- [ ] T005 [UI] Application detail に entitlement 管理 UI を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: 同一ユーザーに競合する entitlement を割り当てようとすると保存前 preview と保存時検証で拒否されることを確認する。
- 手動: group 経由の entitlement と直接 entitlement の合算で SoD conflict が検出されることを確認する。

## Risk Notes
entitlement は外部アプリの実権限と対応するため、台帳と実態のずれが監査リスクになる。初期は IdMagic 管理下の明示 catalog に限定し、外部発見や自動削除は扱わない。SoD は保存時にも必ず評価し、UI preview だけに依存しない。
