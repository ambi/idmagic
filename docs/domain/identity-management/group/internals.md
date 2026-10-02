# グループの内部設計

## Group Aggregate と実効ロール

`Group` はテナント単位の Aggregate であり、`(id, tenant_id, name, description?, roles[], created_at, updated_at?)` を持つ。組織変更のたびに影響する全 User の `roles` を個別に編集せず、ロールの組（「営業チーム = `catalog:read` + `invoice:read`」）を 1 単位として付与・取り消しできるよう導入した。`id` は生成後に変わらない `group_<uuid>` である。`name` はテナント内で一意な編集可能な表示名であり、`(tenant_id, name)` の一意インデックスで強制する。テナントをまたぐメンバーシップは無条件に拒否する。`AddMember` は対象の `User` を読み込み、存在しない場合や別テナントに属する場合は拒否する。

User の実効ロールは `user.roles ∪ ⋃_{g ∈ user.groups} g.roles` である。単純な和集合を整列して重複を除き、減算や優先順位の規則は持たない。平らな和集合で十分なところへ deny や minus の演算子を導入すると、評価順の複雑さが増すためである。User がどの Group にも属さない場合、実効ロールは `user.roles` に戻るため、`Group` の導入によって既存アカウントの挙動は変わらない。管理コンソールの RBAC 制御と `/account` の自己管理ビューでは、生の `user.roles` ではなく実効ロールを解決する。User は自身の実効権限のうち、グループメンバーシップに由来するものを確認できる。`User.roles` 自体は、どの Group にも覆われない User 個別の上書き経路として残す。ロールはデフォルトではトークンクレームへ射影しない。個別ロールにも Group 由来のロールにも、そのマッピングはまだ存在せず、ここでは意図的に対象外とする。

メンバーシップ操作は冪等である。既存メンバーの追加や、メンバーではない User の削除はドメインイベントを再発行しない `no_op` とし、Okta と Keycloak のメンバーシップ API の扱いに合わせる。Group の CRUD とメンバーシップの変更では、`AdminAuditEvent` と、`GroupCreated` / `GroupUpdated` / `GroupDeleted` / `GroupMemberAdded` / `GroupMemberRemoved` のいずれかを発行する。Group の削除ではメンバーシップをカスケード削除し、最後の `GroupDeleted` より前にメンバーごとの `GroupMemberRemoved` を発行する。

## グループの連絡先とカスタム属性

`Group` は任意の `email` (部署のメーリングリストなど単純な連絡先) と、テナントが定義する任意の項目を入れる疎な `attributes` も持つ。スキーマを持たない自由形式のキー・値ではなく、管理者が定義したスキーマでグループのプロフィールを拡張する方式を採り、`User` の属性と同じ統治の姿勢を保つ。`email` は `User.email` と同じ形式検査だけを行い、検証済みフラグ、変更要求のフロー、一意性の制約は持たない。グループには受信箱を支配していることを示せる本人がおらず、それに依存する認証経路もないからである。

`Group.attributes` は、`GroupAttributeDef` で定義する `TenantGroupAttributeSchema` に対して検証する。これは `Tenancy` に属するテナント単位の Aggregate である。どのプリンシパルを統治するスキーマであっても、テナント単位のスキーマ管理は `Tenancy` の関心事であるため、`TenantUserAttributeSchema` と同じ場所に置く。`GroupAttributeDef` は `UserAttributeDef` と異なり、`key`、`label`、`type`、`multi_valued`、`required` だけを持ち、`editable_by_user`、`claim_name` / `oidc_scope`、`visibility` は持たない。`Group` にはセルフサービスの編集画面がなく、その属性を OIDC / SAML クレームへ射影しないためである。和集合にする組み込みカタログもない。`User` の組み込み層は、OIDC §5.1 と SCIM `enterprise:User` が多数の任意プロフィールクレームを固定的に定めるため存在するが、Group には同様の標準語彙がない。そのため `TenantGroupAttributeSchema.attributes` だけが実効定義の集合になる。未定義キーの拒否、型の一致、`multi_valued` の整合性、`required` の充足という `ValidateAttributes` 型の検査は概念として再利用する。一方で、2 つの定義がすべてのフィールドを共有するわけではないため、`GroupAttributeDef` に対する Group 固有の処理として実装する。管理者は、ユーザースキーマと同じ形の 2 つのエンドポイント `GetTenantGroupAttributeSchema` / `UpdateTenantGroupAttributeSchema` (`/api/admin/v1/tenant/group_attribute_schema`) を通じてスキーマを管理する。
