# Feature: 動的グループのシナリオ

## 対象の操作

### Rule: REQ-IDMANAGEMENT-020 管理者は CEL の規則で動的グループの所属を管理できる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-020-01 通常経路

- Given `department` 属性が定義され、Engineering の User と Sales の User が存在する
- And `membership_type=dynamic` のグループが存在する
- When 管理者が `user.department == "Engineering"` を保存して有効化する
- Then 全件の再評価後、Engineering の有効な User だけが動的規則を由来として所属する
- Then 実効ロールと Application の割り当ては、その所属を参照する

## 結果

### Rule: REQ-IDMANAGEMENT-021 CEL の規則は保存前に選んだユーザーでプレビューできる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-021-01 通常経路

- Given 管理者が最大 100 件の User を選択している
- When 未保存の CEL 式を評価する
- Then レスポンスは一致の有無と、追加・削除・変更なしの判定を返し、属性値そのものは返さない

### Rule: REQ-IDMANAGEMENT-023 評価できない規則は権限を付与しない

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-023-01 通常経路

- Given 有効な動的規則のバージョンが更新された
- When 新しいバージョンの動的規則を再評価する
- Then 旧バージョンのメンバーシップは直ちに実効ロールから除外される
- Then 再評価に失敗した User は、新しいバージョンのメンバーシップを得ない

## 拒否

### Rule: REQ-IDMANAGEMENT-022 不正な CEL の規則と動的グループの手動操作は拒否される

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-022-01 通常経路

- When 管理者が、未定義の属性または許可外の関数を参照する CEL 式を保存する
- Then 保存は拒否される
- When 管理者が動的グループに対して `AddGroupMember` または `RemoveGroupMember` を手動で呼ぶ
- Then メンバーシップの変更は拒否される
