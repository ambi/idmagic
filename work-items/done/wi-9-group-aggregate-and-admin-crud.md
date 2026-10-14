---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Group aggregate を SCL に追加し、user-group membership と admin CRUD を提供する

tenant-scoped な `Group` 集約と user-group membership、admin CRUD を
spec-first で実装した。SCL に Group / GroupMember モデル・5 イベント・
GroupLifecycle 状態・8 interface・`admin.groups.{read,write}` 権限を追加し、
Go ドメイン (`spec.EffectiveRoles`)、`GroupRepository` port、`admin_groups`
use case、memory / postgres adapter、HTTP handler、bootstrap 配線、
AdminGroupsPage と AdminUsersPage の「所属グループ」セクションまで通した。
認可は `effective_roles(user) = user.roles ∪ ⋃ g.roles` の union で、
管理 RBAC ゲートと `/account` セルフビューの両面に効く。グループが空なら
`user.roles` と一致するため既存 wire 形式・既存テストは無変更で pass する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
