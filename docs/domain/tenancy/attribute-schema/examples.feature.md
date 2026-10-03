# Feature: 属性スキーマの例

## Rule: REQ-TENANCY-002 管理者はテナント固有のユーザー属性スキーマを定義できる

### Example: EX-TENANCY-002-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が editable_by_user=true の custom_attribute を追加する
- Then 更新後のスキーマに追加した属性が含まれる

## Rule: REQ-TENANCY-020 管理者はテナント固有のグループ属性スキーマを定義できる

### Example: EX-TENANCY-020-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が group custom attribute "cost_center" (type=string, required=false) を追加する
- Then 更新後のスキーマに追加した属性が含まれ "TenantGroupAttributeSchemaUpdated" が発行される

### Example: EX-TENANCY-020-02 既存 key と重複する key を追加する

- Given admin ロールを持つ "operator" が認証済みである
- When "operator" が group custom attribute "cost_center" (type=string, required=false) を追加する
- But 既存 key と重複する key を追加する
- Then 更新は InvalidGroupAttributeSchemaError で拒否される

## Rule: REQ-TENANCY-043 動的グループの規則が参照するユーザー属性は、削除も型の変更もできない

### Example: EX-TENANCY-043-01 参照されている属性を削除する

- Given 動的グループの規則がユーザー属性 "region" を参照している
- When "operator" が "region" を含まないスキーマを保存する
- Then `attribute_referenced_by_dynamic_group` の 409 で拒否され、スキーマは "region" を持ったままで、イベントは発行されない

### Example: EX-TENANCY-043-02 参照されている属性の型を変える

- Given 動的グループの規則がユーザー属性 "region" を参照している
- When "operator" が "region" の型を変えたスキーマを保存する
- Then `attribute_referenced_by_dynamic_group` の 409 で拒否され、"region" の型は変わらない

### Example: EX-TENANCY-043-03 参照されていない属性を削除する

- Given テナントはユーザー属性 "region" と "shift" を定義し、動的グループの規則は "region" だけを参照している
- When "operator" が "region" だけを含むスキーマを保存する
- Then 保存は成功し、要求になかった "shift" は削除される
