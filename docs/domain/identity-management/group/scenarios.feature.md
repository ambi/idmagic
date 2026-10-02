# Feature: グループのシナリオ

## 入力

### Rule: REQ-IDMANAGEMENT-024 管理者はグループの連絡先メールとカスタム属性を、テナント定義のスキーマに従って設定できる

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-024-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- Then 作成したグループの `email` と `attributes` が指定どおりに保存され、"GroupCreated" が発行される
- When "operator" が同じグループの `email` と `attributes` を更新する
- Then 更新後のグループに新しい値が反映され、"GroupUpdated" の `changed_fields` に "email" と "attributes" が含まれる

#### Example: EX-IDMANAGEMENT-024-02 `email` がメールアドレスの形式を満たさない

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- But `email` がメールアドレスの形式を満たさない
- Then 作成は InvalidEmailError で拒否される

#### Example: EX-IDMANAGEMENT-024-03 `attributes` に未定義のキーを指定する、または定義済みのキーと型が一致しない

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- But `attributes` に未定義のキーを指定する、または定義済みのキーと型が一致しない
- Then 作成は InvalidGroupAttributeError で拒否される

## 結果

### Rule: REQ-IDMANAGEMENT-015 管理者はグループを作成しユーザーを所属させると有効ロールにグループ由来ロールが乗る

Primary actor: `TenantAdministrator`

#### Example: EX-IDMANAGEMENT-015-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が認証済みである
- And ロールが空のユーザー "alice" が同一テナントに存在する
- When 管理者 "operator" がロール=["catalog:read"] のグループ "engineering" を作成する
- Then "GroupCreated" が発行される
- When 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
- Then "GroupMemberAdded" が発行される
- When 管理者がユーザー "alice" の所属グループを取得する
- Then 実効ロールに "catalog:read" が含まれる
- Then `group_roles` は "catalog:read" を含み、`direct_roles` は空である

#### Example: EX-IDMANAGEMENT-015-02 同じユーザーを同じグループへ再度所属させる

- Given ロール=["admin"] のユーザー "operator" が認証済みである
- And ロールが空のユーザー "alice" が同一テナントに存在する
- When 管理者 "operator" がロール=["catalog:read"] のグループ "engineering" を作成する
- Then "GroupCreated" が発行される
- When 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
- But 同じユーザーを同じグループへ再度所属させる
- Then 管理者 "operator" がユーザー "alice" をグループ "engineering" に再度所属させる
- And "GroupMemberAdded" は再発行されない
