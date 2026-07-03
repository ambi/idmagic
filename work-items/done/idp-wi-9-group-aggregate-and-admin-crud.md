---
id: idp-wi-9-group-aggregate-and-admin-crud
title: "Group aggregate を SCL に追加し、user-group membership と admin CRUD を提供する"
created_at: 2026-06-15
authors: ["tn"]
status: completed
risk: medium
---
# Motivation
現状 RBAC は `User.roles[]` のみで、組織内のロール割当はユーザ単位で
手作業の積み上げになる。次が成立しない:
  - 「営業チーム = catalog:read + invoice:read + support:read」のような
    まとまった権限束を再利用する。
  - 新規入社時に "営業チーム" にだけ入れれば必要権限が一斉に揃う、
    退職時に "営業チーム" から外せば一斉に剥奪される、という運用。
  - tenant 内でロール体系が増えてきた場合の admin 工数。

本 WI では `Group` aggregate を SCL に正式に導入し、user-group
membership と group-role を持たせる。認可時の有効ロールは
`effective_roles(user) = user.roles ∪ ⋃_{g ∈ user.groups} g.roles`
とする (union)。階層なし、role 削減方向のルール (deny / minus) なし、
attribute-based 自動所属なし、SCIM 同期なし — RA らしく最小集合で
始める。

Group は tenant-scoped aggregate とし、cross-tenant 参照は禁止する
(既存 ADR-032〜034 と整合)。既存の `requireAdmin` / `requireSystemAdmin`
/ `permissions` セクションは effective roles を参照する形に揃える
(内部実装の差し替えのみで wire 形式は変えない)。

# Scope
- **decision**: 新規 ADR (`idmagic/decisions/ADR-037-group-aggregate-and-effective-roles.md`): Group を tenant-scoped aggregate とする、union semantics、階層なし、 deny ルールなし、自動所属なし、Group は `permissions` の `admin.groups.*` で保護、effective_roles の解決順序、id_token / userinfo に groups claim を出さない既定 (admin が opt-in できるかは別 WI)、`User.roles` を 残す理由 (個別 user override の経路として)。
- **scl**: vocabulary: `aggregate.Group`、`event.GroupCreated` / `GroupUpdated` / `GroupDeleted` / `GroupMemberAdded` / `GroupMemberRemoved` を追加。, models: `Group { id, tenant_id, name, description?, roles[], created_at, updated_at }`、`GroupMember { group_id, user_sub, added_at }` を追加。, state_machines: `Group { Active }` の単一状態 (削除は aggregate 自体の lifecycle で表す)。, interfaces: `group.list / group.get / group.create / group.update / group.delete / group.members.add / group.members.remove` を AdminConsole コンポーネントに追加。, permissions: `admin.groups.read` / `admin.groups.write` を追加し、 `admin` ロールに紐付ける。`system_admin` は default tenant の管理 面のみ追加権限。, objectives: effective_roles の合成則と「membership 操作は audit event を必ず emit する」を保証義務として明記。, scenarios: admin が group を作成して user を所属させる → user の effective roles に group.roles が乗る、を 1 シナリオで追加 (自然文 ステップ、§3.6 形式)。
- **go**:
  - domain: internal/spec/ に Group / GroupMember 型と `EffectiveRoles(user, groups)` ヘルパを追加。`User` 自体は変更しない (memberships は 別 aggregate)。
  - ports: GroupRepo: ListByTenant / FindByID / Save / Delete / ListMembersByGroup / ListGroupsByUser / AddMember / RemoveMember。, イベント emit は既存 `Emit` port を流用。
  - usecases: admingroups.{ListGroups, GetGroup, CreateGroup, UpdateGroup, DeleteGroup, AddMember, RemoveMember}。すべて tenant boundary check と admin RBAC を通す。, 既存 `requireAdmin` / `requireSystemAdmin` が role 判定で EffectiveRoles を参照するように internal で差し替える (外部 API の wire 形式は不変)。, DeleteGroup は所属 user の membership も一括解除する。
  - adapters: persistence: memory / postgres 両方に GroupRepo を追加。postgres は `groups` / `group_members` テーブルと外部キー制約 (tenant_id / cascade) を migration で導入。, http: `/admin/groups` (list / create)、`/admin/groups/:id` (get / update / delete)、`/admin/groups/:id/members/:sub` (POST add / DELETE remove)。CSRF と requireAdmin を共通。, [object Object]
  - bootstrap: 起動時 EffectiveRoles の解決経路を組み立て、Deps 構造体に GroupRepo を配線する。
- **ui**: 新規 AdminGroupsPage を追加: 一覧 (name / member count / roles 数)、 詳細パネル (name 編集 / description / roles 編集 + メンバー編集)、 作成・削除ダイアログ。確認なしの破壊的操作はしない。, AdminUsersPage の詳細パネルに「所属グループ」セクションを追加し、 group 一覧を表示する。group 名の click で AdminGroupsPage の 該当 group へ遷移する。membership 編集は AdminGroupsPage 側に 集約する (user 側からは membership 追加だけ可能とする inline コントロールを 1 つ用意する)。, 既存 sidebar (lib/adminNav.ts) に「グループ」を追加する。tenant 共通で表示。AdminTenantsPage 同様に navigation 配列の更新のみ。, effective roles の表示: AdminUsersPage の詳細パネルで「明示ロール (user.roles)」と「グループ由来ロール」を分けて表示する。union を 別段に出すことで admin の理解を助ける。
- **documentation**: idmagic/README.md の Phase 4 「グループ」行は completion で 除去を記録する。, decisions/ 配下の README index に ADR-037 を追加する。, dev demo seed に `engineering` / `support` 2 group を入れ、demo.sh の認可 flow が group-derived 権限で動くことを確認できるようにする。

# Out of Scope
- グループ階層 (group of groups / nested membership)。
- "属性に基づき自動所属" 等の dynamic group ルール。
- deny / minus 権限ルール (union のみ)。
- groups claim の id_token / userinfo 出力 (admin が opt-in できるかは 別 WI)。
- SCIM 2.0 経由の group 同期 (Phase 6 のスコープ)。
- グループ単位の MFA 強制 / セッションポリシー (Phase 2-3 のスコープ)。
- グループに対する consent の bulk 操作 (Phase 5)。

# Verification
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- [object Object]
- 手動: dev サーバを起動し、admin で `/admin/groups` から `engineering` (`catalog:read`) を作成 → alice を追加 → `/admin/users` の詳細で「グループ由来ロール」に `catalog:read` が 表示される → demo.sh の authorization_code flow で `catalog:read` scope が許可されることを確認する。

# Risk Notes
effective_roles の解決経路を追加するため、`requireAdmin` /
`requireSystemAdmin` の通過テストを resolved roles 経路で再走らせる。
既存テストが空 group で走ることを確認し、その後 group を入れた
別シナリオを足す。PostgreSQL migration は前方互換 (空テーブル) で
AUTO_MIGRATE の挙動を確認する。UI 側の sidebar 配列を 1 箇所
(lib/adminNav.ts) に集約しているため、全 6 ページの遷移回帰が出やすい
— 全ページの type-check + lint + build を verification に含める。

# Completion
- **Completed At**: 2026-06-20
- **Summary**:
  tenant-scoped な `Group` 集約と user-group membership、admin CRUD を
  spec-first で実装した。SCL に Group / GroupMember モデル・5 イベント・
  GroupLifecycle 状態・8 interface・`admin.groups.{read,write}` 権限を追加し、
  Go ドメイン (`spec.EffectiveRoles`)、`GroupRepository` port、`admin_groups`
  use case、memory / postgres adapter、HTTP handler、bootstrap 配線、
  AdminGroupsPage と AdminUsersPage の「所属グループ」セクションまで通した。
  認可は `effective_roles(user) = user.roles ∪ ⋃ g.roles` の union で、
  管理 RBAC ゲートと `/account` セルフビューの両面に効く。グループが空なら
  `user.roles` と一致するため既存 wire 形式・既存テストは無変更で pass する。
- **Verification Results**:
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - 手動確認 (residual): dev サーバでの「/admin/groups で engineering 作成 → alice 追加 → /admin/users 詳細で『グループ由来ロール』に catalog:read 表示」の e2e は本セッションでは未実施。memory / http の単体テストで effective roles の RBAC 通過と membership cascade を検証済。
