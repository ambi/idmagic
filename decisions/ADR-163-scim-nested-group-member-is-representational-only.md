---
status: accepted
authors: [tn]
created_at: 2026-08-10
---

# ADR-163: SCIM nested group member(`members[].type == "Group"`)は表現のみ持ち、effective roles には反映しない

## コンテキスト

wi-246 は Group の `members[].type` に `"Group"` を許可する。しかし [ADR-038](ADR-038-group-aggregate-and-effective-roles.md)
はグループ階層 / サブグループを「継承順序と evaluation が複雑化する」という理由で明示的に見送り、
`EffectiveRoles(userRoles, groups) = user.roles ∪ ⋃ g.roles`(ユーザーの直接所属のみ、和集合のみ)に限定した。
`group_members` テーブルは `(group_id, user_id)` を PK とし `users(id)` への FK が必須で、Group を member として
持てない。nested group member をサポートする際、(a) SCIM プロトコル互換のための「表現」としてだけ持つのか、
(b) 認可(`EffectiveRoles`)にも反映する真の階層にするのかを決める必要があった。(b) は ADR-038 が見送った
決定を再び開くことになる。

## 決定

**nested group member は表現のみ**とする。Group が Group を member として持てるようにし、
GET でネスト構造を返す・POST/PUT/PATCH で受け付けることはするが、**`EffectiveRoles`/`ListGroupsByUser` の
認可計算には一切影響しない**(ADR-038 の flat union のまま)。ネストは外部 IdP(Okta、Entra ID 等)が送ってくる
構造を silently に切り捨てず round-trip させるための SCIM protocol fidelity であって、role 継承機構ではない。

データモデルは既存 `group_members`(User member、`AddMember`/`RemoveMember`/`ListMembersByGroup`/
`ListGroupsByUser` の既存契約)を変更せず、Group-in-Group の関係を別の関係として持つ。循環参照は
DFS による検出を書き込み時の domain 制約として必須にし、展開の深さには上限を設ける。具体的なテーブル形状・
深さ上限の値は実装着手時に `backend/idmanagement/ARCHITECTURE.md` に書く。

## 却下した代替案

- **nested group member を effective roles に反映する(真の階層)**: ADR-038 が見送った「グループ階層」を
  再導入することになり、evaluation 順序・循環時の役割解決・既存 `EffectiveRoles` 呼び出し元(認可ゲート、
  `/account` セルフビュー)への影響が大きい。RFC7643 は member の表現のみを要求しており、SP 側の認可モデルが
  階層継承することまでは求めていないため、SCIM 対応の必達要件ではない。
- **`group_members` を `(member_type, member_id)` の多相テーブルに一般化する**: `users(id)` への FK が
  外せなくなる(条件付き FK か FK 撤去が必要)、かつ既存の `GroupMember{UserID: ...}` を組み立てている
  全呼び出し元(idgovernance、admin console、CSV、auth)が type 判別を意識する必要が生じる。leaf の
  role-bearing membership と構造的なネストという性質の異なる 2 つの関心事を 1 テーブルに混ぜることになる。

## 影響

- `spec/contexts/scim.yaml` の Group 関連 interface(実装時に更新)。
- `backend/idmanagement/group` に Group-in-Group 関係を持つ新しい永続化(`group_members` とは別)を追加するが、
  `GroupRepository` の既存メソッド・`groupdomain.EffectiveRoles`・`ListGroupsByUser` の契約は不変。
- 実装時、`backend/idmanagement/ARCHITECTURE.md` にテーブル形状・循環検出・深さ上限を記述する。
