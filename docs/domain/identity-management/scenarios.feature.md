# Feature: IdManagement のシナリオ

## Rule: REQ-IDMANAGEMENT-014 管理 API のアクセスはロールに応じて制御される

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-014-01 通常経路

- Given ロールに "admin" を持つユーザー "operator" が認証済みである
- When 管理者 "operator" が `preferred_username` "bob" のユーザーを作成する
- Then "UserCreated" が発行される
- When 管理者 "operator" がユーザー一覧を取得する
- Then レスポンスにユーザー "bob" が含まれる

### Example: EX-IDMANAGEMENT-014-02 `admin` ロールを持たないユーザーが管理 API を呼ぶ

- Given ロールに "admin" を持つユーザー "operator" が認証済みである
- When 管理者 "operator" が `preferred_username` "bob" のユーザーを作成する
- Then "UserCreated" が発行される
- When 管理者 "operator" がユーザー一覧を取得する
- But `admin` ロールを持たないユーザーが管理 API を呼ぶ
- Then ロールが空のユーザー "alice" が認証済みである
- And ユーザー "alice" がユーザー一覧を取得する
- And エラー "AccessDeniedError"

## Rule: REQ-IDMANAGEMENT-025 管理 API クライアントはプリンシパルの種類と操作の粒度でだけ User / Group / Agent を操作できる

Primary actor: `ManagementApiClient`

### Example: EX-IDMANAGEMENT-025-01 通常経路

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- Then `users:read` は User の参照に加えて、User を変更しない CSV エクスポートの開始、参照、ダウンロード、取り消しを許可する
- Then `groups:read` は Group の参照に加えて、Group を変更しない動的グループ規則のプレビューと CSV エクスポートを許可する
- Then `groups:write` は Group の変更に加えて、CSV インポートのプレビューと適用を許可する
- Then `agents:write` は Agent のキルと削除の両方を許可する

### Example: EX-IDMANAGEMENT-025-02 `users:read` だけで User の変更または CSV インポートを要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- But `users:read` だけで User の変更または CSV インポートを要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-IDMANAGEMENT-025-03 `groups:read` だけで Group の CSV インポートまたはその適用を要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- But `groups:read` だけで Group の CSV インポートまたはその適用を要求する
- Then 操作は AccessDeniedError で拒否され、`Group` は 1 件も作成、更新、削除されない

### Example: EX-IDMANAGEMENT-025-04 `users:*` だけで Group または Agent の操作を要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- But `users:*` だけで Group または Agent の操作を要求する
- Then 操作は AccessDeniedError で拒否される

### Example: EX-IDMANAGEMENT-025-05 `agents:read` だけで Agent のキルまたは削除を要求する

- Given クライアントは対象テナントの有効な API アクセストークンを提示している
- And トークンの発行者は対象操作に必要なロールを今も持っている
- When クライアントが User、Group、または Agent の操作をリクエストする
- But `agents:read` だけで Agent のキルまたは削除を要求する
- Then 操作は AccessDeniedError で拒否される

## Rule: REQ-IDMANAGEMENT-032 `system_admin` は制御面テナントの User と Group にだけ割り当てられる

Primary actor: `TenantAdministrator`

### Example: EX-IDMANAGEMENT-032-01 通常経路

- Given 制御面テナント `default` にロール=["admin"] のユーザー "operator" とロールが空のユーザー "root" が存在する
- When 管理者 "operator" がロール=["system_admin"] のグループ "system-operators" を作成する
- Then "GroupCreated" が発行され、グループのロールに "system_admin" が含まれる
- When 管理者 "operator" がユーザー "root" をグループ "system-operators" に所属させる
- Then "GroupMemberAdded" が発行される
- When 管理者 "operator" がユーザー "root" の所属グループを取得する
- Then 実効ロールに "system_admin" が含まれ、`group_roles` は "system_admin" を含み、`direct_roles` は空である

### Example: EX-IDMANAGEMENT-032-02 制御面テナントの User へ直接割り当てる

- Given 制御面テナント `default` にロール=["admin"] のユーザー "operator" とロールが空のユーザー "root" が存在する
- When 管理者 "operator" がユーザー "root" のロールを ["system_admin"] へ更新する
- Then 更新は成功し、ユーザー "root" の `roles` は "system_admin" を含む

### Example: EX-IDMANAGEMENT-032-03 制御面テナント以外のテナントの User へ直接割り当てる

- Given 制御面テナントではないテナント "acme" にロール=["admin"] のユーザー "operator" とロールが空のユーザー "alice" が存在する
- When 管理者 "operator" がユーザー "alice" のロールを ["system_admin"] へ、表示名を同じ要求で更新する
- Then 更新は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And ユーザー "alice" の `roles`、表示名、`updated_at` はいずれも変更されない

### Example: EX-IDMANAGEMENT-032-04 制御面テナント以外のテナントの Group へ割り当てる

- Given 制御面テナントではないテナント "acme" にロール=["admin"] のユーザー "operator" が存在する
- When 管理者 "operator" がロール=["system_admin"] のグループ "escalation" を作成する
- Then 作成は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And グループ "escalation" は作成されず、"GroupCreated" も発行されない

### Example: EX-IDMANAGEMENT-032-05 制御面テナントの Agent へ割り当てる

- Given 制御面テナント `default` にロール=["admin"] のユーザー "operator" と在籍中の Agent "batch" が存在する
- When 管理者 "operator" が Agent "batch" のロールを ["system_admin"] へ更新する
- Then 更新は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And Agent "batch" の `roles` と `updated_at` は変更されず、"AgentUpdated" も発行されない

### Example: EX-IDMANAGEMENT-032-06 制御面テナント以外のテナントの User CSV の行が `system_admin` を含む

- Given 制御面テナントではないテナント "acme" にロール=["support"] のユーザー "alice" が存在する
- When 管理者が `roles` 列に "system_admin" を持つ行を含む User CSV を事前検証へ投入する
- Then 対象行は `roles` 列を指す安定したエラーコード `invalid_roles` で `rejected` となり、ユーザー "alice" は変更されない

### Example: EX-IDMANAGEMENT-032-07 制御面テナント以外のテナントの Group CSV の行が `system_admin` を含む

- Given 制御面テナントではないテナント "acme" にロール=["catalog:read"] のグループ "engineering" が存在する
- When 管理者が `roles` 列に "system_admin" を持つ行を含む Group CSV を事前検証へ投入する
- Then 対象行は `roles` 列を指す安定したエラーコード `invalid_roles` で `rejected` となり、グループ "engineering" は変更されない

### Example: EX-IDMANAGEMENT-032-08 既に `system_admin` を持つ対象へ同じ値を再送する

- Given 制御面テナントではないテナント "acme" にロール=["system_admin"] のユーザー "legacy" が既に存在する
- When 管理者が `roles` 列に "system_admin" を持つ "legacy" の行を含む User CSV を事前検証へ投入する
- Then 対象行は `unchanged` となり、ユーザー "legacy" の `roles` は "system_admin" のまま残る

### Example: EX-IDMANAGEMENT-032-09 制御面テナント以外のテナントで User を作成する

- Given 制御面テナントではないテナント "acme" にロール=["admin"] のユーザー "operator" が存在する
- When 管理者 "operator" がロール=["system_admin"] のユーザー "mallory" を作成する
- Then 作成は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And ユーザー "mallory" は作成されず、"UserCreated" も発行されない

### Example: EX-IDMANAGEMENT-032-10 制御面テナント以外のテナントの Group を更新する

- Given 制御面テナントではないテナント "acme" にロール=["catalog:read"] のグループ "engineering" が存在する
- When 管理者がグループ "engineering" のロールを ["system_admin"] へ、説明を同じ要求で更新する
- Then 更新は `InvalidRequestError` で拒否され、エラー "invalid_role" を返す
- And グループ "engineering" の `roles`、説明、`updated_at` はいずれも変更されない
