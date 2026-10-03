# Feature: グループの例

## Rule: REQ-IDMANAGEMENT-024 管理者はグループの連絡先メールとカスタム属性を、テナント定義のスキーマに従って設定できる

### Example: EX-IDMANAGEMENT-024-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- Then 作成したグループの `email` と `attributes` が指定どおりに保存され、"GroupCreated" が発行される
- When "operator" が同じグループの `email` と `attributes` を更新する
- Then 更新後のグループに新しい値が反映され、"GroupUpdated" の `changed_fields` に "email" と "attributes" が含まれる

### Example: EX-IDMANAGEMENT-024-02 `email` がメールアドレスの形式を満たさない

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- But `email` がメールアドレスの形式を満たさない
- Then 作成は InvalidEmailError で拒否される

### Example: EX-IDMANAGEMENT-024-03 `attributes` に未定義のキーを指定する、または定義済みのキーと型が一致しない

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- But `attributes` に未定義のキーを指定する、または定義済みのキーと型が一致しない
- Then 作成は InvalidGroupAttributeError で拒否される

## Rule: REQ-IDMANAGEMENT-060 Group の名前、説明、連絡先の正規化

### Example: EX-IDMANAGEMENT-060-01 大文字と小文字だけが異なる名前

- Given テナントに名前 "engineering" の Group がある
- When 管理者が名前 "Engineering" の Group を作成する
- Then 作成は `group_name_conflict` で拒否される

### Example: EX-IDMANAGEMENT-060-02 表示名付きの連絡先

- When 管理者が連絡先 " Sales Team <Sales@Example.TEST> " で Group を作成する
- Then 連絡先は "sales@example.test" で保存される

### Example: EX-IDMANAGEMENT-060-03 空白だけの説明

- When 管理者が説明 "   " で Group を作成する
- Then Group は説明を持たない

## Rule: REQ-IDMANAGEMENT-061 Group の属性は、テナントの Group 属性スキーマがなければ持てない

### Example: EX-IDMANAGEMENT-061-01 スキーマのないテナントの属性

- Given テナントは Group 属性スキーマを定義していない
- When 管理者が属性 `cost_center` を指定して Group を作成する
- Then 作成は `invalid_attribute` で拒否され、Group は作られない

### Example: EX-IDMANAGEMENT-061-02 必須の属性を省いた作成

- Given テナントの Group 属性スキーマは必須の属性 `cost_center` を定義している
- When 管理者が属性を指定せずに Group を作成する
- Then 作成は `invalid_attribute` で拒否される

## Rule: REQ-IDMANAGEMENT-062 Group の更新は、値が変わった項目だけを記録する

### Example: EX-IDMANAGEMENT-062-01 何も変わらない更新

- When 管理者がグループ "engineering" の現在と同じ説明を指定して更新する
- Then 更新は成功し、`updated_at` は変わらず、`GroupUpdated` は発行されない

## Rule: REQ-IDMANAGEMENT-015 管理者はグループを作成しユーザーを所属させると有効ロールにグループ由来ロールが乗る

### Example: EX-IDMANAGEMENT-015-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が認証済みである
- And ロールが空のユーザー "alice" が同一テナントに存在する
- When 管理者 "operator" がロール=["catalog:read"] のグループ "engineering" を作成する
- Then "GroupCreated" が発行される
- When 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
- Then "GroupMemberAdded" が発行される
- When 管理者がユーザー "alice" の所属グループを取得する
- Then 実効ロールに "catalog:read" が含まれる
- Then `group_roles` は "catalog:read" を含み、`direct_roles` は空である

### Example: EX-IDMANAGEMENT-015-02 同じユーザーを同じグループへ再度所属させる

- Given ロール=["admin"] のユーザー "operator" が認証済みである
- And ロールが空のユーザー "alice" が同一テナントに存在する
- When 管理者 "operator" がロール=["catalog:read"] のグループ "engineering" を作成する
- Then "GroupCreated" が発行される
- When 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
- But 同じユーザーを同じグループへ再度所属させる
- Then 管理者 "operator" がユーザー "alice" をグループ "engineering" に再度所属させる
- And "GroupMemberAdded" は再発行されない

## Rule: REQ-IDMANAGEMENT-063 手動のメンバーの追加は、削除されていない同じテナントの User を受け付ける

### Example: EX-IDMANAGEMENT-063-01 無効化された User の追加

- Given ユーザー "alice" は `Disabled` である
- When 管理者が "alice" を手動グループ "engineering" に追加する
- Then "alice" は "engineering" のメンバーになり、`GroupMemberAdded` が発行される

### Example: EX-IDMANAGEMENT-063-02 削除済みの User の追加

- Given ユーザー "alice" は `Deleted` である
- When 管理者が "alice" を手動グループ "engineering" に追加する
- Then 追加は `user_not_found` で拒否され、メンバーシップは作られない

## Rule: REQ-IDMANAGEMENT-064 下流への通知に失敗した Group の変更は、確定したままエラーを返す

### Example: EX-IDMANAGEMENT-064-01 通知の失敗

- Given 下流のプロビジョニングへの通知は失敗する
- When 管理者がユーザー "bob" を手動グループ "engineering" に追加する
- Then 操作はエラーを返し、"bob" は "engineering" のメンバーとして残り、`GroupMemberAdded` は発行済みである
