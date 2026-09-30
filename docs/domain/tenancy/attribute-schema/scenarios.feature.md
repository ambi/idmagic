# Feature: 属性スキーマのシナリオ

## 入力

### Rule: REQ-TENANCY-002 管理者はテナント固有のユーザー属性スキーマを定義できる

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-002-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が editable_by_user=true の custom_attribute を追加する
- Then 更新後のスキーマに追加した属性が含まれる

### Rule: REQ-TENANCY-020 管理者はテナント固有のグループ属性スキーマを定義できる

Primary actor: `TenantAdministrator`

#### Example: EX-TENANCY-020-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が group custom attribute "cost_center" (type=string, required=false) を追加する
- Then 更新後のスキーマに追加した属性が含まれ "TenantGroupAttributeSchemaUpdated" が発行される

#### Example: EX-TENANCY-020-02 既存 key と重複する key を追加する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が group custom attribute "cost_center" (type=string, required=false) を追加する
- But 既存 key と重複する key を追加する
- Then 更新は InvalidGroupAttributeSchemaError で拒否される
