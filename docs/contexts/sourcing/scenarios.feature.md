# Feature: Sourcing Scenarios

## Rule: REQ-SOURCING-001 SCIM クライアントは Users と Groups のコレクションを検索できる

Primary actor: `ScimClient`

### Example: EX-SOURCING-001-01 通常経路

- Given SCIM クライアントがテナントに対する有効なプロビジョニングトークンを持つ
- When SCIM クライアントが `filter`、`startIndex`、`count` を指定して `/Users` と `/Groups` を GET する
- Then 各レスポンスは、`filter` 適用後の `totalResults`、該当ページの `Resources`、`itemsPerPage` を持つ SCIM ListResponse を返す

### Example: EX-SOURCING-001-02 `filter` が許可されていない属性または演算子を使っている、あるいは構文が不正である

- Given SCIM クライアントがテナントに対する有効なプロビジョニングトークンを持つ
- When SCIM クライアントが `filter`、`startIndex`、`count` を指定して `/Users` と `/Groups` を GET する
- But `filter` が許可されていない属性または演算子を使っている、あるいは構文が不正である
- Then `invalidFilter` の SCIM プロトコルエラーで拒否される

### Example: EX-SOURCING-001-03 `startIndex` または `count` を整数として解釈できない、あるいは `count` が負数である

- Given SCIM クライアントがテナントに対する有効なプロビジョニングトークンを持つ
- When SCIM クライアントが `filter`、`startIndex`、`count` を指定して `/Users` と `/Groups` を GET する
- But `startIndex` または `count` を整数として解釈できない、あるいは `count` が負数である
- Then `invalidValue` の SCIM プロトコルエラーで拒否される

### Example: EX-SOURCING-001-04 プロビジョニングトークンのテナントがリクエスト先のテナントと一致しない

- Given SCIM クライアントがテナントに対する有効なプロビジョニングトークンを持つ
- When SCIM クライアントが `filter`、`startIndex`、`count` を指定して `/Users` と `/Groups` を GET する
- But プロビジョニングトークンのテナントがリクエスト先のテナントと一致しない
- Then SCIM プロトコルエラーで拒否される

## Rule: REQ-SOURCING-002 外部 IdP から SCIM でユーザーのライフサイクルを同期できる

Primary actor: `ScimBearerClient`

### Example: EX-SOURCING-002-01 通常経路

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser を呼び出す
- Then 内部 User が作成され、ステータスが `Active` になる
- When SCIM クライアントが PatchScimUser で `active=false` を指定する
- Then 内部 User が Disabled になる
- When SCIM クライアントが DeleteScimUser を呼び出す
- Then 内部 User が PendingDeletion に遷移する

### Example: EX-SOURCING-002-02 Bearer トークンが失効済み、期限切れ、または別テナントのトークンである

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser を呼び出す
- But Bearer トークンが失効済み、期限切れ、または別テナントのトークンである
- Then SCIM プロトコルエラーを返し、User を作成しない

### Example: EX-SOURCING-002-03 PATCH のリクエストボディが RFC 7644 の操作要件を満たさない

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser を呼び出す
- Then 内部 User が作成され、ステータスが `Active` になる
- When SCIM クライアントが PatchScimUser で `active=false` を指定する
- But PATCH のリクエストボディが RFC 7644 の操作要件を満たさない
- Then `invalidValue` の ScimProtocolError を返し、User を変更しない

### Example: EX-SOURCING-002-04 指定 ID が存在しない

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser を呼び出す
- Then 内部 User が作成され、ステータスが `Active` になる
- When SCIM クライアントが PatchScimUser で `active=false` を指定する
- Then 内部 User が Disabled になる
- When SCIM クライアントが DeleteScimUser を呼び出す
- But 指定 ID が存在しない
- Then 404 の ScimProtocolError を返す

### Example: EX-SOURCING-002-05 削除した User は以後の SCIM 操作と照会結果から消える

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser を呼び出す
- Then 内部 User が作成され、ステータスが `Active` になる
- When SCIM クライアントが DeleteScimUser を呼び出す
- Then 内部 User が PendingDeletion に遷移する
- When SCIM クライアントが同じ id で GetScimUser を呼び出す
- Then 404 の ScimProtocolError を返す
- And ListScimUsers の `Resources` と `totalResults` にその User は現れない

## Rule: REQ-SOURCING-003 外部 IdP は SCIM リソースを PUT で完全に置換できる

Primary actor: `ScimBearerClient`

### Example: EX-SOURCING-003-01 通常経路

- Given 有効な SCIM アクセストークンが発行されている
- And 対象 User が存在し、`name.givenName` と `active=false` を持つ
- When SCIM クライアントが、`userName` だけを含むリクエストボディで UpdateScimUser を呼び出す
- Then `name.givenName` は空文字に、`active` は `true` にリセットされる
- Then レスポンスは `id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を含む

### Example: EX-SOURCING-003-02 PUT のリクエストボディに必須属性（User の `userName`、Group の `displayName`）がない

- Given 有効な SCIM アクセストークンが発行されている
- And 対象 User が存在し、`name.givenName` と `active=false` を持つ
- When SCIM クライアントが、`userName` だけを含むリクエストボディで UpdateScimUser を呼び出す
- But PUT のリクエストボディに必須属性（User の `userName`、Group の `displayName`）がない
- Then `invalidValue` の ScimProtocolError を返し、リソースを変更しない

### Example: EX-SOURCING-003-03 PUT のリクエストボディに既存値と異なる `id` がある

- Given 有効な SCIM アクセストークンが発行されている
- And 対象 User が存在し、`name.givenName` と `active=false` を持つ
- When SCIM クライアントが、`userName` だけを含むリクエストボディで UpdateScimUser を呼び出す
- But PUT のリクエストボディに既存値と異なる `id` がある
- Then 指定された `id` は無視し、サーバーが割り当てた既存の ID を維持する

## Rule: REQ-SOURCING-004 外部 IdP による未対応の PATCH パスや読み取り専用属性への書き込みを拒否する

Primary actor: `ScimBearerClient`

### Example: EX-SOURCING-004-01 通常経路

- Given 有効な SCIM アクセストークンが発行されている
- And 対象 User または Group が存在する
- When SCIM クライアントが RFC7644-PATCH で対応するパス（User: `userName` / `name` / `active` / `emails`、Group: `displayName` / `members`）に対する replace 操作を、PatchScimUser または PatchScimGroup で送る
- Then 対象属性だけが更新され、他の属性は変化しない

### Example: EX-SOURCING-004-02 `path` が対応属性の許可リスト外である、または存在しない属性を指す

- Given 有効な SCIM アクセストークンが発行されている
- And 対象 User または Group が存在する
- When SCIM クライアントが RFC7644-PATCH で対応するパス（User: `userName` / `name` / `active` / `emails`、Group: `displayName` / `members`）に対する replace 操作を、PatchScimUser または PatchScimGroup で送る
- But `path` が対応属性の許可リスト外である、または存在しない属性を指す
- Then `invalidPath` の ScimProtocolError を返し、リソースを変更しない

### Example: EX-SOURCING-004-03 `path` が読み取り専用の `id`、`meta`、`schemas` のいずれかを指す

- Given 有効な SCIM アクセストークンが発行されている
- And 対象 User または Group が存在する
- When SCIM クライアントが RFC7644-PATCH で対応するパス（User: `userName` / `name` / `active` / `emails`、Group: `displayName` / `members`）に対する replace 操作を、PatchScimUser または PatchScimGroup で送る
- But `path` が読み取り専用の `id`、`meta`、`schemas` のいずれかを指す
- Then `mutability` の ScimProtocolError を返し、リソースを変更しない

### Example: EX-SOURCING-004-04 `op` が `add` / `replace` / `remove` のいずれでもない

- Given 有効な SCIM アクセストークンが発行されている
- And 対象 User または Group が存在する
- When SCIM クライアントが RFC7644-PATCH で対応するパス（User: `userName` / `name` / `active` / `emails`、Group: `displayName` / `members`）に対する replace 操作を、PatchScimUser または PatchScimGroup で送る
- But `op` が `add` / `replace` / `remove` のいずれでもない
- Then `invalidValue` の ScimProtocolError を返し、リソースを変更しない

## Rule: REQ-SOURCING-005 外部 IdP から SCIM でグループとメンバーシップを同期できる

Primary actor: `ScimBearerClient`

### Example: EX-SOURCING-005-01 通常経路

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimGroup を呼び出す
- Then グループが作成される
- When SCIM クライアントが PatchScimGroup でメンバー追加を指定する
- Then GroupMembership が同期され User の有効ロールが更新される
- Then Group レスポンスの各メンバーは `type=User` を持つ

### Example: EX-SOURCING-005-02 追加対象の User が別テナントに属する

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimGroup を呼び出す
- Then グループが作成される
- When SCIM クライアントが PatchScimGroup でメンバー追加を指定する
- But 追加対象の User が別テナントに属する
- Then ScimProtocolError を返し、メンバーシップを作成しない

### Example: EX-SOURCING-005-03 メンバーの `type` が `User` 以外である

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimGroup を呼び出す
- Then グループが作成される
- When SCIM クライアントが PatchScimGroup でメンバー追加を指定する
- But メンバーの `type` が `User` 以外である
- Then `invalidValue` の ScimProtocolError を返し、Group を変更しない

## Rule: REQ-SOURCING-006 SCIM の複数メールアドレスを正規メールアドレスへ決定的に投影する

  - THEN `invalidValue` の ScimProtocolError を返し、User を変更しない
  - THEN `invalidPath` の ScimProtocolError を返し、User を変更しない

Primary actor: `ScimBearerClient`

### Example: EX-SOURCING-006-01 通常経路

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で複数の `emails` を送る
- Then `primary=true` の要素が 1 件あれば、その `value` だけを `User.email` に保存する
- Then User レスポンスは、保存済みメールアドレスがある場合だけ、`type=work`、`primary=true` の要素を 1 つ返す
- When SCIM クライアントが CreateScimUser または UpdateScimUser で `phoneNumbers` または `addresses` を送る
- When SCIM クライアントが PatchScimUser の `path` に `phoneNumbers` または `addresses` を指定する
- When SCIM クライアントが GetScimSchemas を呼び出す
- Then サーバーは、対応する 1 件のメールアドレスへの投影と User メンバーだけを広告し、`phoneNumbers`、`addresses`、Group メンバーは広告しない

### Example: EX-SOURCING-006-02 `primary=true` の要素が複数ある、要素がオブジェクトでない、`value` が空または文字列でない、`type` が文字列でない、`primary` が真偽値でない

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で複数の `emails` を送る
- But `primary=true` の要素が複数ある、要素がオブジェクトでない、`value` が空または文字列でない、`type` が文字列でない、`primary` が真偽値でない
- Then `invalidValue` の ScimProtocolError を返し、User を変更しない

### Example: EX-SOURCING-006-03 `primary` がない

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で複数の `emails` を送る
- But `primary` がない
- Then `type` が大文字と小文字を区別せず `work` と一致する最初の要素を選ぶ

### Example: EX-SOURCING-006-04 `primary` も `work` もない

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で複数の `emails` を送る
- But `primary` も `work` もない
- Then 通信上で最初の要素を選ぶ

## Rule: REQ-SOURCING-007 SCIM Enterprise 拡張の User 組織属性に対応する

Primary actor: `ScimBearerClient`

### Example: EX-SOURCING-007-01 通常経路

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で Enterprise 拡張の `employeeNumber`、`department`、`manager` を送る
- Then 対応する値は `idmanagement.User.Attributes` の `employee_number`、`department`、`manager_sub` に永続化される（`manager` には解決した内部の `User.sub` を保存する）
- Then User レスポンスは、いずれかの Enterprise 拡張属性を保持する場合だけ `schemas` に Enterprise 拡張 URN を含み、対応する属性オブジェクトを返す
- When SCIM クライアントが GetScimSchemas または GetScimResourceTypes を呼び出す
- Then サーバーは Enterprise 拡張スキーマを `/Schemas` に、`/ResourceTypes` の User エントリーの `schemaExtensions` に広告する

### Example: EX-SOURCING-007-02 `employeeNumber` または `department` が文字列でない

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で Enterprise 拡張の `employeeNumber`、`department`、`manager` を送る
- But `employeeNumber` または `department` が文字列でない
- Then `invalidValue` の ScimProtocolError を返し、User を変更しない

### Example: EX-SOURCING-007-03 `manager` の値（`value` オブジェクトまたは文字列）が空である、あるいは同じテナント内に存在しない SCIM User を指す

- Given 有効な SCIM アクセストークンが発行されている
- When SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で Enterprise 拡張の `employeeNumber`、`department`、`manager` を送る
- But `manager` の値（`value` オブジェクトまたは文字列）が空である、あるいは同じテナント内に存在しない SCIM User を指す
- Then `invalidValue` の ScimProtocolError を返し、User を変更しない
