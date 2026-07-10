---
status: pending
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

- [ ] T001 [Domain] `shared/spec/users.go` / `groups.go` / `agents.go` / `attributes.go` の
  業務型を `identitymanagement/domain/` へ移設し参照更新。
- [ ] T002 [Kernel] identity-management が 7 context と共有する型を選別し、
  adapter 境界変換 or `shared/kernel` 昇格を判定。
- [ ] T003 [Persistence] identity-management 固有 repo 実装を
  `identitymanagement/adapters/persistence/{postgres,memory}` へ同居。
- [ ] T004 [Persistence] identity-management postgres 実装を sqlc 生成へ置換。
- [ ] T005 [DI] `identitymanagement/module.go` を新設し Module パターン化。
- [ ] T006 [DI] 中央 `server/routes.go` `Deps` と `bootstrap/deps.go` から
  identity-management 分を撤去。
- [ ] T007 [Cross-context] per-context 化済みの依存元 context の User/Group/Agent 型
  import path を更新（第 2 波）。
- [ ] T008 [Measure] 動的クエリ比率を実測し [[ADR-090]] に追記。
- [ ] T009 [Verify] `just verify-go` / `just test-go` green、locality 指標を確認。

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
