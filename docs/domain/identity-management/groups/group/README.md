# グループ

## 概要

この文書は、User を束ねてロールをまとめて付与する `Group` を、管理者が作成し、変更し、メンバーを管理する機能の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `Group` の作成と更新、連絡先とカスタム属性、手動のメンバーシップ、Group 由来のロールを含む実効ロール |
| 行為者 | 管理者 |
| コードの機能スライス | `backend/idmanagement/group` |
| 扱わないもの | CEL の規則による所属は[動的グループ](../dynamic-group/README.md)が、CSV による一括の操作は[グループ CSV](../../bulk-transfer/group-csv/README.md)が、ロールの正規化と予約ロールは[ロール](../../common/roles/README.md)が扱う |

## モデル

`Group` は、テナントに属する Aggregate root である。
メンバーシップと動的グループの規則は `Group` の境界の内側にあり、User はその ID だけで参照する。

| 項目 | 内容 |
| --- | --- |
| `id` | 生成した後に変わらない `group_<uuid>` |
| `name` | テナントの中で大文字と小文字を区別せずに一意な、編集できる名前 |
| `description` | 説明 |
| `email` | 部署のメーリングリストのような連絡先。形式だけを検査し、確認済みの印も一意性も持たない |
| `roles` | メンバーへまとめて付与するロール |
| `membership_type` | `manual`（管理者が所属を決める）または `dynamic`（規則の評価だけが所属を決める）。作成のときにだけ選べる |
| `attributes` | テナントの `TenantGroupAttributeSchema` で検証する疎な属性 |

User の実効ロールは、User に直接付与したロールと、所属する Group のロールの和集合である。
どの Group にも属さない User の実効ロールは、直接付与したロールと同じである。

- **判断**：実効ロールに優先順位や減算の規則を設けない。平らな和集合で足りるところに拒否の規則を入れると、評価の順序の複雑さだけが増える。
- **判断**：`membership_type` を後から変えない。手動の所属と動的な所属では既存のメンバーシップの意味が変わり、反転させると、どの経路で付いた所属なのかを後から復元できない。
- **判断**：Group の属性を User の属性の定義へ統合せず、別の `GroupAttributeDef` で定義する。Group には、共有できる組み込みのカタログも、本人が編集する画面も、クレームとしての開示もないからである。
- **判断**：メンバーシップを `Group` の境界に入れ、User には入れない。規則が指す Group がない状態や、削除した Group にメンバーが残る状態を一瞬でも作らないためである。逆向きにすると、User 一人の更新が、その人の属するすべての Group を直列化させる。

## 操作

### 作成

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 名前、説明、連絡先、ロール、所属の種類、属性 |
| 成功時の作用 | Group を作り、`GroupCreated` を発行し、下流のプロビジョニングへ通知する |
| 拒否 | 空の名前、同名の Group、メールアドレスとして読めない連絡先、スキーマに合わない属性。どの拒否も Group を作らない |

#### REQ-IDMANAGEMENT-024 Group の作成は、連絡先と属性をテナントのスキーマで検証して保存し、`GroupCreated` を発行する

- 連絡先のメールアドレスと、テナントの Group 属性スキーマで定義した属性を指定した作成は、その値で Group を作り、`GroupCreated` を発行する。
- メールアドレスの形式を満たさない連絡先の作成は、InvalidEmailError で拒否する。
- 定義していないキーの属性と、定義した型と一致しない値の属性の作成は、InvalidGroupAttributeError で拒否する。
- **担保手段**：`usecases.CreateGroup`
- **例**：EX-IDMANAGEMENT-024-01、EX-IDMANAGEMENT-024-03

#### REQ-IDMANAGEMENT-060 Group の作成と更新は、名前、説明、連絡先を正規化して保存する

- 名前は前後の空白を除いて保存する。空白を除いて空になる名前は、422 と `group_name_required` で拒否する。
- 同じテナントのほかの Group と大文字と小文字を区別せずに同じ名前は、409 と `group_name_conflict` で拒否する。
- 説明は前後の空白を除き、空になる説明は設定しない。
- 連絡先のメールアドレスは前後の空白を除き、表示名付きの形式からはアドレスだけを取り出し、小文字にして保存する。空になる値は設定しない。
- **担保手段**：`usecases.CreateGroup`、`usecases.UpdateGroup`
- **要判断**：管理 API は表示名付きのメールアドレスを受け付けてアドレスだけを保存するが、Group の CSV は同じ値を `invalid_email` で拒否する。どちらかに揃えるかを決める。

#### REQ-IDMANAGEMENT-061 テナントに Group 属性スキーマがなければ、Group の作成と更新は属性を拒否する

- テナントが Group 属性スキーマを定義していないとき、属性を一つでも含む作成と更新は、422 と `invalid_attribute` で拒否する。
- 作成は、属性を指定しなくても必須の属性の欠落を拒否する。
- **担保手段**：`usecases.CreateGroup`、`usecases.UpdateGroup`

### 更新

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 名前、説明、連絡先、ロール、属性。`attributes` を指定すると属性の全体を置き換える |
| 成功時の作用 | 値が変わった項目だけを保存し、`GroupUpdated` を発行し、下流のプロビジョニングへ通知する |
| 拒否 | 作成と同じ検証の違反（REQ-IDMANAGEMENT-060、REQ-IDMANAGEMENT-061） |
| 冪等性 | 同じ値での更新は、`updated_at` を進めず、イベントを発行せず、通知しない（REQ-IDMANAGEMENT-062） |

#### REQ-IDMANAGEMENT-062 Group の更新は、値が変わった項目だけを記録する

- `attributes` を指定した更新だけが属性を検証し、指定した対応表で属性の全体を置き換える。
- `GroupUpdated` の `changed_fields` には、`name`、`description`、`email`、`attributes`、`roles` のうち値が変わった項目だけを載せる。
- どの項目の値も変わらない更新は成功を返し、`updated_at` を進めず、`GroupUpdated` を発行せず、下流のプロビジョニングへ通知しない。
- **担保手段**：`usecases.UpdateGroup`
- **例**：EX-IDMANAGEMENT-062-01、EX-IDMANAGEMENT-062-02

### メンバーの追加と除外

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 手動の Group と、User |
| 成功時の作用 | メンバーシップを作るか消し、`GroupMemberAdded` または `GroupMemberRemoved` を発行し、下流のプロビジョニングへ通知する。メンバーの実効ロールに Group のロールが加わる |
| 拒否 | 存在しない User、`Deleted` の User、別のテナントの User（404 `user_not_found`）。動的グループへの追加と除外（409 `dynamic_membership_managed_by_rule`、REQ-IDMANAGEMENT-085）。どの拒否もメンバーシップを変えない |
| 冪等性 | すでにメンバーである User の追加と、メンバーでない User の除外は、成功を返しイベントを発行しない |

#### REQ-IDMANAGEMENT-015 User のメンバーへの追加は `GroupMemberAdded` を発行し、User の実効ロールに Group のロールを加える

- User を Group に所属させると、`GroupMemberAdded` を発行し、User の実効ロールに Group のロールを加える。
- 同じ User を同じ Group へもう一度所属させる操作は、`GroupMemberAdded` を再発行しない。
- **担保手段**：`usecases.AddMember`、`domain.EffectiveRoles`
- **例**：EX-IDMANAGEMENT-015-01、EX-IDMANAGEMENT-015-02

#### REQ-IDMANAGEMENT-063 手動のメンバーの追加は、削除されていない同じテナントの User を受け付ける

- 追加の対象は、同じテナントの `Deleted` でない User である。`Disabled` と `PendingDeletion` の User も追加できる。
- 存在しない User、`Deleted` の User、別のテナントの User の追加は、404 と `user_not_found` で拒否し、メンバーシップを作らない。
- すでにメンバーである User の追加と、メンバーでない User の除外は成功を返し、イベントを発行しない。
- **担保手段**：`usecases.AddMember`、`usecases.RemoveMember`

#### REQ-IDMANAGEMENT-085 動的グループへの手動のメンバーの追加と除外は拒否する

- `membership_type=dynamic` の Group への `AddGroupMember` と `RemoveGroupMember` の呼び出しは、409 と `dynamic_membership_managed_by_rule` で拒否し、メンバーシップを変えない。
- **判断**：動的グループの所属は規則の評価だけが決める。手動の操作を許すと、所属がどの経路で付いたのかを区別できなくなる。
- **担保手段**：`usecases.AddMember`、`usecases.RemoveMember`
- **例**：EX-IDMANAGEMENT-085-01

#### REQ-IDMANAGEMENT-064 下流への通知に失敗した Group の変更は、確定したままエラーを返す

- Group の作成、更新、削除、手動のメンバーの追加と除外は、変更を確定してから下流のプロビジョニングへ通知する。
- 通知に失敗した操作はエラーを返すが、確定した変更と発行したイベントは取り消さない。
- **担保手段**：`usecases.AddMember`、`usecases.CreateGroup`
- **要判断**：管理者には失敗が返るが変更は残るため、再試行が重複した操作になり得る。User の変更は通知の失敗を記録して成功を返す。Group も同じにするかを決める。

### 所属グループの参照

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | User |
| 成功時の作用 | User が所属する Group と、実効ロール、Group 由来のロール、直接付与したロールを返す。状態は変えない |
| 拒否 | 存在しない User と別のテナントの User（404 `user_not_found`） |

#### REQ-IDMANAGEMENT-084 User の所属グループの参照は、実効ロールと、Group 由来のロールと直接付与したロールを分けて返す

- 応答は、所属する Group と、実効ロール、Group 由来のロール（`group_roles`）、直接付与したロール（`direct_roles`）を返す。
- 実効ロールは、Group 由来のロールと直接付与したロールの和集合である。
- 存在しない User と別のテナントの User の参照は、404 と `user_not_found` で拒否する。
- **担保手段**：`usecases.UserGroups`、`domain.EffectiveRoles`
- **例**：EX-IDMANAGEMENT-084-01

## セキュリティ上の考慮

| 観点 | 内容 |
| --- | --- |
| テナント境界 | テナントをまたぐメンバーシップは作らない。追加では対象の User を読み、別のテナントの User を存在しない User と同じ応答で拒否する |
| ロールの付与 | Group のロールはメンバー全員の実効ロールに加わるので、Group のロールの変更は管理者の権限の変更と同じ重さを持つ。予約ロールの制限は[ロール](../../common/roles/README.md)が定める |
