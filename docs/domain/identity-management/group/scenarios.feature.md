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

### Rule: REQ-IDMANAGEMENT-060 Group の名前、説明、連絡先の正規化

- 名前は前後の空白を除いて保存する。空白を除いて空になる名前は、422 と `group_name_required` で拒否する。
- 同じテナントのほかの Group と大文字と小文字を区別せずに同じ名前は、409 と `group_name_conflict` で拒否する。
- 説明は前後の空白を除き、空になる説明は設定しない。
- 連絡先のメールアドレスは前後の空白を除き、表示名付きの形式からはアドレスだけを取り出し、小文字にして保存する。空になる値は設定しない。
- **担保手段**：`usecases.CreateGroup`、`usecases.UpdateGroup`
- **要判断**：管理 API は表示名付きのメールアドレスを受け付けてアドレスだけを保存するが、Group の CSV は同じ値を `invalid_email` で拒否する。どちらかに揃えるかを決める。

#### Example: EX-IDMANAGEMENT-060-01 大文字と小文字だけが異なる名前

- Given テナントに名前 "engineering" の Group がある
- When 管理者が名前 "Engineering" の Group を作成する
- Then 作成は `group_name_conflict` で拒否される

#### Example: EX-IDMANAGEMENT-060-02 表示名付きの連絡先

- When 管理者が連絡先 " Sales Team <Sales@Example.TEST> " で Group を作成する
- Then 連絡先は "sales@example.test" で保存される

#### Example: EX-IDMANAGEMENT-060-03 空白だけの説明

- When 管理者が説明 "   " で Group を作成する
- Then Group は説明を持たない

### Rule: REQ-IDMANAGEMENT-061 Group の属性は、テナントの Group 属性スキーマがなければ持てない

- テナントが Group 属性スキーマを定義していないとき、属性を一つでも含む作成と更新は、422 と `invalid_attribute` で拒否する。
- 作成は、属性を指定しなくても必須の属性の欠落を拒否する。
- 更新は、`attributes` を指定したときだけ属性を検証し、指定した対応表で属性の全体を置き換える。
- **担保手段**：`usecases.CreateGroup`、`usecases.UpdateGroup`

#### Example: EX-IDMANAGEMENT-061-01 スキーマのないテナントの属性

- Given テナントは Group 属性スキーマを定義していない
- When 管理者が属性 `cost_center` を指定して Group を作成する
- Then 作成は `invalid_attribute` で拒否され、Group は作られない

#### Example: EX-IDMANAGEMENT-061-02 必須の属性を省いた作成

- Given テナントの Group 属性スキーマは必須の属性 `cost_center` を定義している
- When 管理者が属性を指定せずに Group を作成する
- Then 作成は `invalid_attribute` で拒否される

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

## 拒否

### Rule: REQ-IDMANAGEMENT-063 手動のメンバーの追加は、削除されていない同じテナントの User を受け付ける

- 追加の対象は、同じテナントの `Deleted` でない User である。`Disabled` と `PendingDeletion` の User も追加できる。
- 存在しない User、`Deleted` の User、別のテナントの User の追加は、404 と `user_not_found` で拒否し、メンバーシップを作らない。
- すでにメンバーである User の追加と、メンバーでない User の除外は成功を返し、イベントを発行しない。
- **担保手段**：`usecases.AddMember`、`usecases.RemoveMember`

#### Example: EX-IDMANAGEMENT-063-01 無効化された User の追加

- Given ユーザー "alice" は `Disabled` である
- When 管理者が "alice" を手動グループ "engineering" に追加する
- Then "alice" は "engineering" のメンバーになり、`GroupMemberAdded` が発行される

#### Example: EX-IDMANAGEMENT-063-02 削除済みの User の追加

- Given ユーザー "alice" は `Deleted` である
- When 管理者が "alice" を手動グループ "engineering" に追加する
- Then 追加は `user_not_found` で拒否され、メンバーシップは作られない

## 作用

### Rule: REQ-IDMANAGEMENT-062 Group の更新は、値が変わった項目だけを記録する

- `GroupUpdated` の `changed_fields` には、`name`、`description`、`email`、`attributes`、`roles` のうち値が変わった項目だけを載せる。
- どの項目の値も変わらない更新は成功を返し、`updated_at` を進めず、`GroupUpdated` を発行せず、下流のプロビジョニングへ通知しない。
- **担保手段**：`usecases.UpdateGroup`

#### Example: EX-IDMANAGEMENT-062-01 何も変わらない更新

- When 管理者がグループ "engineering" の現在と同じ説明を指定して更新する
- Then 更新は成功し、`updated_at` は変わらず、`GroupUpdated` は発行されない

### Rule: REQ-IDMANAGEMENT-064 下流への通知に失敗した Group の変更は、確定したままエラーを返す

- Group の作成、更新、削除、手動のメンバーの追加と除外は、変更を確定してから下流のプロビジョニングへ通知する。
- 通知に失敗した操作はエラーを返すが、確定した変更と発行したイベントは取り消さない。
- **担保手段**：`usecases.AddMember`、`usecases.CreateGroup`
- **要判断**：管理者には失敗が返るが変更は残るため、再試行が重複した操作になり得る。User の変更は通知の失敗を記録して成功を返す。Group も同じにするかを決める。

#### Example: EX-IDMANAGEMENT-064-01 通知の失敗

- Given 下流のプロビジョニングへの通知は失敗する
- When 管理者がユーザー "bob" を手動グループ "engineering" に追加する
- Then 操作はエラーを返し、"bob" は "engineering" のメンバーとして残り、`GroupMemberAdded` は発行済みである
