---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-03
---

# 委任管理 (スコープ付き admin ロール) を導入する

## Motivation
現状の管理者権限はほぼ一枚岩で、テナント内の一部だけを任せる副管理者を表現
できない。大規模テナント / 組織では代表的な IdP が委任管理を提供する:

- Entra ID: Administrative Units + directory roles のスコープ。
- Okta: admin roles + resource sets。
- Keycloak: fine-grained admin permissions。

本 WI は「対象リソース集合 × 権限」で表す scoped admin role を導入し、
既存 RBAC ([[wi-15-roles-and-permissions-inspection-page]]) を拡張して、
例えば特定グループ / アプリだけを管理できる副管理者を表現できるようにする。

## Scope
- **decision**:
  - 新規 ADR: スコープ次元 (グループ / アプリ / 属性集合) と、既存 roles / permissions との関係、fail-closed な既定 (deny)、エンドユーザ向け ReBAC ([[wi-53-rebac-fine-grained-authorization]]) と被らない「管理操作の認可に 限定」する境界を記録する。
- **scl**:
  - §3.2 models: AdminRoleAssignment / ResourceSet を追加する。
  - §3.3 interfaces: admin 操作 (users / groups / applications 等) の認可に scope を反映する。副管理者割当の CRUD を追加する。
  - §3.4 states/events: AdminRoleAssigned / AdminRoleRevoked を追加する。
  - §3.7 permissions: scope 外リソースへの管理操作を構造的に拒否する (既定 deny) ことを明示する。
- **go**:
  - 認可判定に scope を織り込み、既存 admin usecase のガードを scope 対応に する。scope 評価器を追加する。
- **http**:
  - 副管理者の割当 / 取消エンドポイントを追加する。
- **ui**:
  - AdminRolesPage / AdminUsers に scoped admin 割当 UI を追加する。
- **documentation**:
  - README に委任管理のスコープ次元と割当手順を追記する。

## Out of Scope
- エンドユーザ向けの ReBAC / FGA ([[wi-53-rebac-fine-grained-authorization]])。
- 汎用ポリシー言語 (Rego / Cedar 等) の導入。
- cross-tenant delegation。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: 特定グループのみ管理できる副管理者を割当 → そのグループは操作でき、 scope 外のユーザ / グループ / アプリの管理操作が拒否されることを確認する。

## Risk Notes
認可の中核を触るため、既存 admin 操作全体に回帰リスクがある。scope 評価を
fail-closed (既定 deny) で設計し、scope 外操作の拒否を操作種別ごとにテストする。
ReBAC (wi-53) と役割が重ならないよう「管理操作限定」に境界を明示する。
