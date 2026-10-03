# Feature: 動的グループの例

## Rule: REQ-IDMANAGEMENT-020 管理者は CEL の規則で動的グループの所属を管理できる

### Example: EX-IDMANAGEMENT-020-01 通常経路

- Given `department` 属性が定義され、Engineering の User と Sales の User が存在する
- And `membership_type=dynamic` のグループが存在する
- When 管理者が `user.department == "Engineering"` を保存して有効化する
- Then 全件の再評価後、Engineering の有効な User だけが動的規則を由来として所属する
- Then 実効ロールと Application の割り当ては、その所属を参照する

## Rule: REQ-IDMANAGEMENT-066 動的グループの規則の版と有効化

### Example: EX-IDMANAGEMENT-066-01 初めての保存と有効化

- When 管理者が動的グループに初めて規則を保存し、続けて有効化する
- Then 保存した規則は無効で版 1 であり、有効化した規則は版 2 である
- And 全件の再評価が予約される

### Example: EX-IDMANAGEMENT-066-02 有効な規則の式の保存

- Given 動的グループの規則は有効で版 2 である
- When 管理者が式を保存する
- Then 規則は有効のまま版 3 になり、全件の再評価が予約される

## Rule: REQ-IDMANAGEMENT-065 動的グループの規則の式の制約

### Example: EX-IDMANAGEMENT-065-01 ロールを参照する式

- When 管理者が `"admin" in user.roles` を保存する
- Then 保存は `invalid_dynamic_group_rule` で拒否される

### Example: EX-IDMANAGEMENT-065-02 真偽値を返さない式

- When 管理者が `user.preferred_username` を保存する
- Then 保存は `invalid_dynamic_group_rule` で拒否される

### Example: EX-IDMANAGEMENT-065-03 定数でない正規表現

- When 管理者が `user.email.matches(user.preferred_username)` を保存する
- Then 保存は `invalid_dynamic_group_rule` で拒否される

## Rule: REQ-IDMANAGEMENT-022 不正な CEL の規則と動的グループの手動操作は拒否される

### Example: EX-IDMANAGEMENT-022-01 通常経路

- When 管理者が、未定義の属性または許可外の関数を参照する CEL 式を保存する
- Then 保存は拒否される
- When 管理者が動的グループに対して `AddGroupMember` または `RemoveGroupMember` を手動で呼ぶ
- Then メンバーシップの変更は拒否される

## Rule: REQ-IDMANAGEMENT-068 規則の無効化は、動的グループの所属をすべて直ちに外す

### Example: EX-IDMANAGEMENT-068-01 所属を持つ規則の無効化

- Given 有効な規則によりユーザー "alice" と "bob" が動的グループに所属している
- When 管理者が規則を無効化する
- Then 応答を返す時点で、動的グループのメンバーシップは空である

## Rule: REQ-IDMANAGEMENT-067 動的グループの規則に一致するのは `Active` の User だけである

### Example: EX-IDMANAGEMENT-067-01 無効化された User

- Given ユーザー "alice" の `department` は "Engineering" で、"alice" は `Disabled` である
- When 規則 `user.department == "Engineering"` で "alice" を評価する
- Then "alice" は一致しない

## Rule: REQ-IDMANAGEMENT-023 評価できない規則は権限を付与しない

### Example: EX-IDMANAGEMENT-023-01 通常経路

- Given 有効な動的規則のバージョンが更新された
- When 新しいバージョンの動的規則を再評価する
- Then 旧バージョンのメンバーシップは直ちに実効ロールから除外される
- Then 再評価に失敗した User は、新しいバージョンのメンバーシップを得ない

## Rule: REQ-IDMANAGEMENT-069 古い版の再評価のジョブは、所属を変えずに成功する

### Example: EX-IDMANAGEMENT-069-01 規則が更新された後の古いジョブ

- Given 版 2 の再評価のジョブが予約され、その後に規則は版 3 になった
- When 版 2 のジョブが実行される
- Then ジョブは成功し、メンバーシップは変わらない

## Rule: REQ-IDMANAGEMENT-070 動的な所属の変化は、メンバーごとのイベントを発行しない

### Example: EX-IDMANAGEMENT-070-01 全件の再評価による追加

- Given ユーザー "alice" は有効な規則に一致し、まだ所属していない
- When 全件の再評価を実行する
- Then "alice" は所属し、`GroupMemberAdded` は発行されず、追加の件数 1 を持つ `DynamicMembershipEvaluated` が発行される

## Rule: REQ-IDMANAGEMENT-021 CEL の規則は保存前に選んだユーザーでプレビューできる

### Example: EX-IDMANAGEMENT-021-01 通常経路

- Given 管理者が最大 100 件の User を選択している
- When 未保存の CEL 式を評価する
- Then レスポンスは一致の有無と、追加・削除・変更なしの判定を返し、属性値そのものは返さない

## Rule: REQ-IDMANAGEMENT-071 規則のプレビューは 100 件以下の、同じテナントの User だけを受け付ける

### Example: EX-IDMANAGEMENT-071-01 101 件の User

- When 管理者が 101 件の User を指定してプレビューする
- Then プレビューは `invalid_dynamic_group_rule` で拒否される

### Example: EX-IDMANAGEMENT-071-02 存在しない User を含むプレビュー

- When 管理者がユーザー "alice" と存在しない User を指定してプレビューする
- Then プレビューは結果を返さず `user_not_found` で拒否される
