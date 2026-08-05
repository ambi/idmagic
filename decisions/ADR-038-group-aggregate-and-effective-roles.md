---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-038: グループ集約と実効ロール (effective roles)

## コンテキスト

これまで RBAC は `User.roles []string` のみで表現されてきた (ADR-031)。
「営業チーム = `catalog:read` + `invoice:read`」のようなロール束を一つの単位
として定義し、入社・異動・退職に合わせてまとめて付与・剥奪する手段が無い。
ロールをユーザーごとに個別管理すると、組織変更のたびに全該当ユーザーを
個別更新する必要があり、監査も困難になる。

Keycloak (groups + group roles)、Okta (groups → role/app assignment)、
Google IAM (groups as principals) はいずれもグループによるロール集約を
一級機能として持つ。idmagic にも tenant-scoped なグループ集約を導入する。

本 ADR は WI-9 の決定を記録する。WI のテキストは新 ADR を `ADR-037` と
呼んでいたが、`ADR-037-use-case-layer-only-when-it-carries-domain-work.md`
が既に存在するため番号を `ADR-038` に繰り下げた。

## 決定

`Group` 集約を新規導入する。`(id, tenant_id, name, description?, roles[], created_at, updated_at?)`
を持ち、テナントに閉じ、`(tenant_id, name)` がテナント内で一意である。実効ロールは
`user.roles ∪ ⋃_{g ∈ user.groups} g.roles` (和集合のみ、階層・減算なし) と定義し、管理コンソールの
RBAC ゲートと `/account` セルフビューの二面で適用する。メンバーシップ操作は冪等とし、`User.roles`
は個別 override 経路として維持する。トークンへの groups/roles claim 投入は既定で行わない。

グループによるロール集約を導入するのは、ロールをユーザーごとに個別管理すると組織変更のたびに
全該当ユーザーを個別更新する必要があり監査も困難になるためで、Keycloak / Okta / Google IAM の
グループ機能に倣う。和集合のみに限定するのは、階層・減算・動的メンバーシップ等で評価順序を複雑化
させる機能を持ち込まずに要求を満たすためで、下表の見送った代替案はこの RA-minimal 方針に基づく。
現在の集約フィールド・一意性制約・適用面・冪等性・監査イベントの詳細は
[`backend/idmanagement/ARCHITECTURE.md`](../backend/idmanagement/ARCHITECTURE.md) に置く。

## 影響

- `groups` / `group_members` テーブルを追加 (`infra/migrations/0008_groups.sql`)。
  `groups.tenant_id` は `tenants(id)` へ FK (RESTRICT)、`(tenant_id, name)` に
  unique index、`group_members` は `groups(id)` へ FK ON DELETE CASCADE、
  `users(sub)` へ FK ON DELETE CASCADE。`roles` は JSONB。
- 5 つのドメインイベントを outbox / 監査経路に流す
  (topic `iam.groups.v1`)。
- 管理 UI に `/admin/groups` ページを追加。`AdminUsersPage` の詳細パネルに
  「所属グループ」セクションを追加し、ロール表示を
  明示ロール / グループ由来ロール / 実効ロールに分割する。
- RBAC ゲートは実効ロールを参照するため `GroupRepo` を HTTP `Deps` に注入する。
  未注入 (`GroupRepo == nil`) の場合は raw `user.Roles` を返し、後方互換を保つ。

## 検討したが見送った代替案 (considered & deferred)

Keycloak / Okta / Google IAM との比較で機能ギャップを洗い出し、RA-minimal の
方針で以下を本 WI の scope 外とした。

| 項目 | 参照製品 | 判断 | 理由 |
| --- | --- | --- | --- |
| グループ階層 / サブグループ | Keycloak nested groups | 見送り | 継承順序と evaluation が複雑化。フラットな union で要求を満たす |
| 動的 / ルールベース所属 | Okta group rules, Keycloak | 見送り | 属性ベース自動メンバーシップは別 WI。明示メンバーシップに限定 |
| 既定 / 自動参加グループ | Keycloak default groups, Okta Everyone | 見送り (別 WI) | 2 つの scope 質問でユーザーと合意済み |
| deny / minus ルール | — | 見送り | union のみ。減算は評価順序の複雑性を招く |
| メンバーシップ・ロール / 委譲グループ管理 | Okta group admin roles | 見送り | グループ単位の delegated admin は別関心事 |
| 期限付きメンバーシップ | — | 見送り | time-bound membership は別 WI |
| 自由形式グループ属性 | Keycloak group attributes | 見送り | スキーマレス属性は本フェーズで不要 |
| groups / roles トークン claim | Keycloak group membership mapper | 見送り (opt-in) | role→claim マッピング自体が未実装。別 WI |
| ロール付与時の昇格防止ガード | — | 見送り | 既存 `user.roles` の無制限付与と対称に保つ。横断的関心事として別 WI に明記 |
| SCIM プロビジョニング | Okta/Azure SCIM | 見送り | 外部プロビジョニングは Phase 外 |
