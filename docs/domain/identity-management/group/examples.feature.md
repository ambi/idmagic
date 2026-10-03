# Feature: グループの例

## Rule: REQ-IDMANAGEMENT-024 Group の作成は、連絡先と属性をテナントのスキーマで検証して保存し、`GroupCreated` を発行する

### Example: EX-IDMANAGEMENT-024-01 通常経路

- Given admin ロールを持つ "operator" が認証済みである
- And テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- When "operator" が email="sales@example.test" と attributes={cost_center: "CC-100"} を指定してグループ "sales" を作成する
- Then 作成したグループの `email` と `attributes` が指定どおりに保存され、"GroupCreated" が発行される

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

## Rule: REQ-IDMANAGEMENT-060 Group の作成と更新は、名前、説明、連絡先を正規化して保存する

一次情報は [実行可能な具体例](../../../../backend/idmanagement/group/usecases/testdata/normalization.examples.json) である。
同じ表を作成と更新の両方へ実行する。
成功時は保存値を読み戻し、拒否時は状態、イベント、通知が変わらないことを確かめる。
`null` は未設定、空の `error` は成功を表す。

<!-- spec:examples backend/idmanagement/group/usecases/testdata/normalization.examples.json -->

### Example: EX-IDMANAGEMENT-060-01 大文字と小文字だけが異なる名前

- Given 次の前提を満たす

  | 項目 | 値 |
  | --- | --- |
  | existing_name | "engineering" |
- When 次の入力で規則の対象操作を実行する

  | 項目 | 値 |
  | --- | --- |
  | name | "Engineering" |
  | email | null |
  | description | null |
- Then 次の結果になる

  | 項目 | 値 |
  | --- | --- |
  | error | "group_name_conflict" |
  | name | null |
  | email | null |
  | description | null |

### Example: EX-IDMANAGEMENT-060-02 表示名付きの連絡先

- Given 追加の前提はない

- When 次の入力で規則の対象操作を実行する

  | 項目 | 値 |
  | --- | --- |
  | name | "sales" |
  | email | " Sales Team <Sales@Example.TEST> " |
  | description | null |
- Then 次の結果になる

  | 項目 | 値 |
  | --- | --- |
  | error | "" |
  | name | "sales" |
  | email | "sales@example.test" |
  | description | null |

### Example: EX-IDMANAGEMENT-060-03 空白だけの説明

- Given 追加の前提はない

- When 次の入力で規則の対象操作を実行する

  | 項目 | 値 |
  | --- | --- |
  | name | "sales" |
  | email | null |
  | description | "   " |
- Then 次の結果になる

  | 項目 | 値 |
  | --- | --- |
  | error | "" |
  | name | "sales" |
  | email | null |
  | description | null |

### Example: EX-IDMANAGEMENT-060-04 空の名前

- Given 追加の前提はない

- When 次の入力で規則の対象操作を実行する

  | 項目 | 値 |
  | --- | --- |
  | name | "" |
  | email | null |
  | description | null |
- Then 次の結果になる

  | 項目 | 値 |
  | --- | --- |
  | error | "group_name_required" |
  | name | null |
  | email | null |
  | description | null |

### Example: EX-IDMANAGEMENT-060-05 Unicode の空白だけの名前

- Given 追加の前提はない

- When 次の入力で規則の対象操作を実行する

  | 項目 | 値 |
  | --- | --- |
  | name | "　\\t " |
  | email | null |
  | description | null |
- Then 次の結果になる

  | 項目 | 値 |
  | --- | --- |
  | error | "group_name_required" |
  | name | null |
  | email | null |
  | description | null |

### Example: EX-IDMANAGEMENT-060-06 名前と説明と連絡先の前後の空白

- Given 追加の前提はない

- When 次の入力で規則の対象操作を実行する

  | 項目 | 値 |
  | --- | --- |
  | name | " engineering " |
  | email | " Sales@Example.TEST " |
  | description | " Sales team " |
- Then 次の結果になる

  | 項目 | 値 |
  | --- | --- |
  | error | "" |
  | name | "engineering" |
  | email | "sales@example.test" |
  | description | "Sales team" |

### Example: EX-IDMANAGEMENT-060-07 空白だけの連絡先と空の説明

- Given 追加の前提はない

- When 次の入力で規則の対象操作を実行する

  | 項目 | 値 |
  | --- | --- |
  | name | "sales" |
  | email | "   " |
  | description | "" |
- Then 次の結果になる

  | 項目 | 値 |
  | --- | --- |
  | error | "" |
  | name | "sales" |
  | email | null |
  | description | null |

### Example: EX-IDMANAGEMENT-060-08 形式を満たさない連絡先

- Given 追加の前提はない

- When 次の入力で規則の対象操作を実行する

  | 項目 | 値 |
  | --- | --- |
  | name | "sales" |
  | email | "not-an-email" |
  | description | null |
- Then 次の結果になる

  | 項目 | 値 |
  | --- | --- |
  | error | "invalid_email" |
  | name | null |
  | email | null |
  | description | null |

### Example: EX-IDMANAGEMENT-060-09 名前の内部の空白と区切りは保持する

- Given 追加の前提はない

- When 次の入力で規則の対象操作を実行する

  | 項目 | 値 |
  | --- | --- |
  | name | " 開発 \| Team\\n" |
  | email | null |
  | description | " 開発部\\n内製 " |
- Then 次の結果になる

  | 項目 | 値 |
  | --- | --- |
  | error | "" |
  | name | "開発 \| Team" |
  | email | null |
  | description | "開発部\\n内製" |

<!-- /spec:examples -->

## Rule: REQ-IDMANAGEMENT-061 テナントに Group 属性スキーマがなければ、Group の作成と更新は属性を拒否する

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

### Example: EX-IDMANAGEMENT-062-02 連絡先と属性を変える更新

- Given テナントの Group 属性スキーマに "cost_center" (string, required=false) が定義されている
- And グループ "sales" の `email` は "sales@example.test"、`attributes` は {cost_center: "CC-100"} である
- When 管理者がグループ "sales" の `email` と `attributes` を更新する
- Then 更新後のグループに新しい値が反映され、"GroupUpdated" の `changed_fields` に "email" と "attributes" が含まれる

## Rule: REQ-IDMANAGEMENT-015 User のメンバーへの追加は `GroupMemberAdded` を発行し、User の実効ロールに Group のロールを加える

### Example: EX-IDMANAGEMENT-015-01 通常経路

- Given ロール=["admin"] のユーザー "operator" が認証済みである
- And ロールが空のユーザー "alice" が同一テナントに存在する
- When 管理者 "operator" がロール=["catalog:read"] のグループ "engineering" を作成する
- Then "GroupCreated" が発行される
- When 管理者 "operator" がユーザー "alice" をグループ "engineering" に所属させる
- Then "GroupMemberAdded" が発行される
- And ユーザー "alice" の実効ロールに "catalog:read" が含まれる

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

## Rule: REQ-IDMANAGEMENT-084 User の所属グループの参照は、実効ロールと、Group 由来のロールと直接付与したロールを分けて返す

### Example: EX-IDMANAGEMENT-084-01 Group 由来のロールだけを持つ User

- Given ロールが空のユーザー "alice" は、ロール=["catalog:read"] のグループ "engineering" に所属している
- When 管理者がユーザー "alice" の所属グループを取得する
- Then 実効ロールに "catalog:read" が含まれる
- Then `group_roles` は "catalog:read" を含み、`direct_roles` は空である

## Rule: REQ-IDMANAGEMENT-085 動的グループへの手動のメンバーの追加と除外は拒否する

### Example: EX-IDMANAGEMENT-085-01 動的グループへの手動の操作

- Given `membership_type=dynamic` のグループが存在する
- When 管理者が動的グループに対して `AddGroupMember` または `RemoveGroupMember` を手動で呼ぶ
- Then メンバーシップの変更は `dynamic_membership_managed_by_rule` で拒否される
