# Feature: 管理 API の認可の例

## Rule: REQ-IDMANAGEMENT-014 管理 API のアクセスはロールに応じて制御される

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
