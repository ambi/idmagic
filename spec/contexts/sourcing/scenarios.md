# Sourcing Scenarios

### REQ-SOURCING-001: SCIM クライアントは Users と Groups のコレクションを検索できる
- ACTOR ScimClient
- GIVEN SCIM クライアントがテナントに対する有効なプロビジョニングトークンを持つ
- WHEN SCIM クライアントが `filter`、`startIndex`、`count` を指定して `/Users` と `/Groups` を GET する
  - ALT `filter` が許可されていない属性または演算子を使っている、あるいは構文が不正である → `invalidFilter` の SCIM プロトコルエラーで拒否される
  - ALT `startIndex` または `count` を整数として解釈できない、あるいは `count` が負数である → `invalidValue` の SCIM プロトコルエラーで拒否される
  - ALT プロビジョニングトークンのテナントがリクエスト先のテナントと一致しない → SCIM プロトコルエラーで拒否される
- THEN 各レスポンスは、`filter` 適用後の `totalResults`、該当ページの `Resources`、`itemsPerPage` を持つ SCIM ListResponse を返す

### REQ-SOURCING-002: 外部 IdP から SCIM でユーザーのライフサイクルを同期できる
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIM クライアントが CreateScimUser を呼び出す
  - ALT Bearer トークンが失効済み、期限切れ、または別テナントのトークンである → SCIM プロトコルエラーを返し、User を作成しない
- THEN 内部 User が作成され、ステータスが `Active` になる
- WHEN SCIM クライアントが PatchScimUser で `active=false` を指定する
  - ALT PATCH のリクエスト本文が RFC 7644 の操作要件を満たさない → `invalidValue` の ScimProtocolError を返し、User を変更しない
- THEN 内部 User が Disabled になる
- WHEN SCIM クライアントが DeleteScimUser を呼び出す
  - ALT 指定 ID が存在しない → 404 の ScimProtocolError を返す
- THEN 内部 User が PendingDeletion に遷移する

### REQ-SOURCING-003: 外部 IdP は SCIM リソースを PUT で完全に置換できる
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- GIVEN 対象 User が存在し、`name.givenName` と `active=false` を持つ
- WHEN SCIM クライアントが、`userName` だけを含むリクエスト本文で UpdateScimUser を呼び出す
  - ALT PUT のリクエスト本文に必須属性（User の `userName`、Group の `displayName`）がない → `invalidValue` の ScimProtocolError を返し、リソースを変更しない
  - ALT PUT のリクエスト本文に既存値と異なる `id` がある → 指定された `id` は無視し、サーバーが割り当てた既存の ID を維持する
- THEN `name.givenName` は空文字に、`active` は `true` にリセットされる
- THEN レスポンスは `id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を含む

### REQ-SOURCING-004: 外部 IdP による未対応の PATCH パスや読み取り専用属性への書き込みを拒否する
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- GIVEN 対象 User または Group が存在する
- WHEN SCIM クライアントが RFC7644-PATCH で対応するパス（User: `userName` / `name` / `active` / `emails`、Group: `displayName` / `members`）に対する replace 操作を、PatchScimUser または PatchScimGroup で送る
  - ALT `path` が対応属性の許可リスト外である、または存在しない属性を指す → `invalidPath` の ScimProtocolError を返し、リソースを変更しない
  - ALT `path` が読み取り専用の `id`、`meta`、`schemas` のいずれかを指す → `mutability` の ScimProtocolError を返し、リソースを変更しない
  - ALT `op` が `add` / `replace` / `remove` のいずれでもない → `invalidValue` の ScimProtocolError を返し、リソースを変更しない
- THEN 対象属性だけが更新され、他の属性は変化しない

### REQ-SOURCING-005: 外部 IdP から SCIM でグループとメンバーシップを同期できる
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIM クライアントが CreateScimGroup を呼び出す
- THEN グループが作成される
- WHEN SCIM クライアントが PatchScimGroup でメンバー追加を指定する
  - ALT 追加対象の User が別テナントに属する → ScimProtocolError を返し、メンバーシップを作成しない
  - ALT メンバーの `type` が `User` 以外である → `invalidValue` の ScimProtocolError を返し、Group を変更しない
- THEN GroupMembership が同期され User の有効ロールが更新される
- THEN Group レスポンスの各メンバーは `type=User` を持つ

### REQ-SOURCING-006: SCIM の複数メールアドレスを正規メールアドレスへ決定的に投影する
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で複数の `emails` を送る
  - ALT `primary=true` の要素が複数ある、要素がオブジェクトでない、`value` が空または文字列でない、`type` が文字列でない、`primary` が真偽値でない → `invalidValue` の ScimProtocolError を返し、User を変更しない
  - ALT `primary` がない → `type` が大文字と小文字を区別せず `work` と一致する最初の要素を選ぶ
  - ALT `primary` も `work` もない → 通信上で最初の要素を選ぶ
- THEN `primary=true` の要素が 1 件あれば、その `value` だけを `User.email` に保存する
- THEN User レスポンスは、保存済みメールアドレスがある場合だけ、`type=work`、`primary=true` の要素を 1 つ返す
- WHEN SCIM クライアントが CreateScimUser または UpdateScimUser で `phoneNumbers` または `addresses` を送る
  - THEN `invalidValue` の ScimProtocolError を返し、User を変更しない
- WHEN SCIM クライアントが PatchScimUser の `path` に `phoneNumbers` または `addresses` を指定する
  - THEN `invalidPath` の ScimProtocolError を返し、User を変更しない
- WHEN SCIM クライアントが GetScimSchemas を呼び出す
- THEN サーバーは、対応する 1 件のメールアドレスへの投影と User メンバーだけを広告し、`phoneNumbers`、`addresses`、Group メンバーは広告しない

### REQ-SOURCING-007: SCIM Enterprise 拡張の User 組織属性に対応する
- ACTOR ScimBearerClient
- GIVEN 有効な SCIM アクセストークンが発行されている
- WHEN SCIM クライアントが CreateScimUser、UpdateScimUser、または PatchScimUser で Enterprise 拡張の `employeeNumber`、`department`、`manager` を送る
  - ALT `employeeNumber` または `department` が文字列でない → `invalidValue` の ScimProtocolError を返し、User を変更しない
  - ALT `manager` の値（`value` オブジェクトまたは文字列）が空である、あるいは同じテナント内に存在しない SCIM User を指す → `invalidValue` の ScimProtocolError を返し、User を変更しない
- THEN 対応する値は `idmanagement.User.Attributes` の `employee_number`、`department`、`manager_sub` に永続化される（`manager` には解決した内部の `User.sub` を保存する）
- THEN User レスポンスは、いずれかの Enterprise 拡張属性を保持する場合だけ `schemas` に Enterprise 拡張 URN を含み、対応する属性オブジェクトを返す
- WHEN SCIM クライアントが GetScimSchemas または GetScimResourceTypes を呼び出す
- THEN サーバーは Enterprise 拡張スキーマを `/Schemas` に、`/ResourceTypes` の User エントリーの `schemaExtensions` に広告する
