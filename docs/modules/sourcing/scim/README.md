# SCIM による取り込み

## 概要

この文書は、SCIM 2.0 のサーバーとして、Okta、Google Cloud Identity、Entra ID などの外部の IdP から User と Group の同期を受ける仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `/Users` と `/Groups` の検索、作成、置換、部分更新、削除と、`/ServiceProviderConfig`、`/ResourceTypes`、`/Schemas` の Discovery |
| 行為者 | `ScimClient`（テナント単位の Bearer トークンを提示する外部のエージェント） |
| 扱わないもの | トークンの発行は `ApiTokens` が、User の無効化と削除の後の扱いは `IdManagement` が扱う |

## モデル

各テナントは `/realms/{realm_id}/scim/v2` を持つ。
サーバーは `/Users` と `/Groups`（GET、POST、GET/{id}、PUT/{id}、PATCH/{id}、DELETE/{id}）、`/ServiceProviderConfig`、`/ResourceTypes`、`/Schemas` を実装する。

属性は、User と Group の Aggregate へ直接対応付ける。

| SCIM | IdMagic |
| --- | --- |
| `Users.id` | `User.sub` |
| `Users.userName` | `User.preferred_username` |
| `Users.name.formatted` / `displayName` | `User.name` |
| `Users.emails` | `User.email`（primary、work、通信上の順序の優先順位で 1 件へ投影） |
| `Users.active` | `UserLifecycle.status == Active` |
| `Groups.id` | `Group.id` |
| `Groups.displayName` | `Group.name` |
| `Groups.members` | `GroupMember` のメンバーシップ |
| enterprise extension `employeeNumber` | `User.Attributes["employee_number"]` |
| enterprise extension `department` | `User.Attributes["department"]` |
| enterprise extension `manager.value` | `User.Attributes["manager_sub"]`（内部 `User.sub`。SCIM id 経由で解決） |

SCIM の `emails` は複数の値を持てるが、IdMagic の正規の User は、認証と通知に使う単一のメールアドレスだけを持つ。
User のレスポンスは、正規のメールアドレスがある場合だけ、単一の `type=work, primary=true` の要素へ正規化して返す。
入力の配列の全体を損失なく往復できることは保証しない。

スキーマの Discovery では、実装した範囲だけを広告する。
Group のメンバーシップは、直接所属する User だけを表し、Group の間の入れ子は保存しない。
Enterprise 拡張は `employeeNumber`、`department`、`manager` の三つの属性だけに対応し、`costCenter`、`division`、`organization` は対象外とする。
PATCH の `path` は、`employeeNumber` などの単純な名前と、Enterprise 拡張の URN で修飾した完全なパスの両方を受け付ける。

- **判断**：SCIM の多値のプロフィールのデータは、プロトコル専用の副次的なストアに持たず、IdMagic の正規の Aggregate へ投影する。ワイヤ表現を保存すると、正規のユーザー像が二つできてしまうからである。
- **判断**：未対応の複合属性（`phoneNumbers`、`addresses`）と入れ子の Group は、黙って捨てずに明示的に拒否する。気付かれないデータの損失と、認可の上で意味を持たない Group のグラフを生まないためである。

## 操作

### SCIM クライアントによるコレクションの検索

#### REQ-SOURCING-001 SCIM クライアントは Users と Groups のコレクションを検索できる

- SCIM クライアントが `/Users` または `/Groups` を検索したとき、Sourcing は、`filter` に一致するリソースの総数を `totalResults` に、`startIndex` から `count` 件のリソースを `Resources` に、返した件数を `itemsPerPage` に載せた SCIM の ListResponse を返す。
- `count` を省略した検索を受けたとき、Sourcing は、`ServiceProviderConfig` が広告する上限の 100 件を `count` として使う。
- 100 を超える `count` を受けたとき、Sourcing は、100 件まで返す。
- `count` に 0 を受けたとき、Sourcing は、リソースを返さず `totalResults` だけを返す。
- 1 未満の `startIndex` を受けたとき、Sourcing は、1 として扱う。
- Sourcing は、User の `userName`、`active`、`name.formatted`、`name.givenName`、`name.familyName`、`emails.value`、`id`、`meta.created`、`meta.lastModified` と、Group の `displayName`、`id`、`meta.created`、`meta.lastModified` を、属性ごとに許した演算子と `pr`、`and`、`or` でだけ絞り込む。
- 許していない属性か演算子を使う `filter`、または構文が不正な `filter` を受けた場合、Sourcing は、400 と scimType `invalidFilter` の SCIM のエラーで拒否する。
- 整数として読めない `startIndex` か `count`、または負の `count` を受けた場合、Sourcing は、400 と scimType `invalidValue` の SCIM のエラーで拒否する。
- 失効したか、期限を過ぎたか、別のテナントで発行したトークンを受けた場合、Sourcing は、401 の SCIM のエラーで拒否し、リソースを作成も変更もしない。
- 操作に必要な `scim:users:*` または `scim:groups:*` のスコープを持たないトークンを受けた場合、Sourcing は、403 の SCIM のエラーで拒否し、`WWW-Authenticate` に必要なスコープを示す。
- **例**：EX-SOURCING-001-01、EX-SOURCING-001-02、EX-SOURCING-001-03、EX-SOURCING-001-04

### SCIM クライアントによる User の同期

#### REQ-SOURCING-002 外部 IdP から SCIM でユーザーのライフサイクルを同期できる

- SCIM クライアントが User を作成したとき、Sourcing は、`Active` の User を作り、201 と SCIM の User を返す。
- SCIM クライアントが `active=false` を指定して User を更新したとき、Sourcing は、IdManagement の無効化を通して User を `Disabled` にする。
- SCIM クライアントが `active=true` を指定して `Disabled` の User を更新したとき、Sourcing は、IdManagement の再有効化を通して User を `Active` に戻す。
- SCIM クライアントが User を削除したとき、Sourcing は、IdManagement の削除の予約を通して User を `PendingDeletion` にし、204 を返す。
- User が `PendingDeletion` の間、Sourcing は、その User を取得と検索の結果と `totalResults` に含めず、その `id` への操作を 404 の SCIM のエラーで拒否する。
- 同じテナントのほかの User と同じ `userName` を指定された場合、Sourcing は、409 と scimType `uniqueness` の SCIM のエラーで拒否し、User を作成も変更もしない。
- 存在しないか別のテナントの `id` を指定された場合、Sourcing は、404 の SCIM のエラーで拒否する。
- JSON として読めない本文を受けた場合、Sourcing は、400 と scimType `invalidSyntax` の SCIM のエラーで拒否する。
- **例**：EX-SOURCING-002-01、EX-SOURCING-002-02、EX-SOURCING-002-03、EX-SOURCING-002-04、EX-SOURCING-002-05

#### REQ-SOURCING-006 SCIM の複数メールアドレスを正規メールアドレスへ決定的に投影する

- SCIM クライアントが複数の `emails` を送ったとき、Sourcing は、`primary=true` の要素、`type` が大文字と小文字を区別せず `work` の最初の要素、通信上の最初の要素の順に一つを選び、その `value` だけを `User.email` に保存する。
- User がメールアドレスを持つ間、SCIM の User を返すとき、Sourcing は、`emails` に `type=work`、`primary=true` の要素を一つだけ返す。
- User がメールアドレスを持たない間、SCIM の User を返すとき、Sourcing は、`emails` を返さない。
- SCIM クライアントがスキーマを取得したとき、Sourcing は、一件のメールアドレスへの投影と User のメンバーだけを広告し、`phoneNumbers`、`addresses`、Group のメンバーを広告しない。
- `primary=true` の要素が複数ある、要素がオブジェクトでない、`value` が空か文字列でない、`type` が文字列でない、または `primary` が真偽値でない `emails` を受けた場合、Sourcing は、400 と scimType `invalidValue` の SCIM のエラーで拒否し、User を変えない。
- 作成または置換で `phoneNumbers` か `addresses` を受けた場合、Sourcing は、400 と scimType `invalidValue` の SCIM のエラーで拒否し、User を変えない。
- PATCH の `path` に `phoneNumbers` か `addresses` を指定された場合、Sourcing は、400 と scimType `invalidPath` の SCIM のエラーで拒否し、User を変えない。
- **例**：EX-SOURCING-006-01、EX-SOURCING-006-02、EX-SOURCING-006-03、EX-SOURCING-006-04

#### REQ-SOURCING-007 SCIM Enterprise 拡張の User 組織属性に対応する

- SCIM クライアントが Enterprise 拡張の `employeeNumber`、`department`、`manager` を送ったとき、Sourcing は、それぞれ User の属性 `employee_number`、`department`、`manager_sub`（`manager` が指す SCIM の User の内部の `User.sub`）に保存する。
- User が Enterprise 拡張の属性を一つ以上持つ間、SCIM の User を返すとき、Sourcing は、`schemas` に Enterprise 拡張の URN を含め、その属性のオブジェクトを返す。
- User が Enterprise 拡張の属性を持たない間、SCIM の User を返すとき、Sourcing は、`schemas` に Enterprise 拡張の URN を含めない。
- SCIM クライアントがスキーマまたはリソースの型を取得したとき、Sourcing は、Enterprise 拡張のスキーマを `/Schemas` に、`/ResourceTypes` の User の `schemaExtensions` に広告する。
- PATCH の `path` に単純な名前（`employeeNumber`、`department`、`manager`）か、Enterprise 拡張の URN で修飾した完全なパスを受けたとき、Sourcing は、どちらも同じ属性として扱う。
- 文字列でない `employeeNumber` か `department` を受けた場合、Sourcing は、400 と scimType `invalidValue` の SCIM のエラーで拒否し、User を変えない。
- 空の `manager` か、同じテナントに存在しない SCIM の User を指す `manager` を受けた場合、Sourcing は、400 と scimType `invalidValue` の SCIM のエラーで拒否し、User を変えない。
- **例**：EX-SOURCING-007-01、EX-SOURCING-007-02、EX-SOURCING-007-03

### SCIM クライアントによるリソースの置換

#### REQ-SOURCING-003 外部 IdP は SCIM リソースを PUT で完全に置換できる

- SCIM クライアントが User または Group を PUT で置換したとき、Sourcing は、本文にない属性をデフォルト値（文字列は空、`active` は `true`）に戻し、200 と `id`、`meta.resourceType`、`meta.created`、`meta.lastModified`、`meta.location` を含むリソースを返す。
- 既存の値と異なる `id` を含む PUT を受けたとき、Sourcing は、本文の `id` を無視し、サーバーが割り当てた `id` を保つ。
- User の `userName` か Group の `displayName` のない PUT を受けた場合、Sourcing は、400 と scimType `invalidValue` の SCIM のエラーで拒否し、リソースを変えない。
- **例**：EX-SOURCING-003-01、EX-SOURCING-003-02、EX-SOURCING-003-03

### SCIM クライアントによるリソースの部分更新

#### REQ-SOURCING-004 外部 IdP による未対応の PATCH パスや読み取り専用属性への書き込みを拒否する

- SCIM クライアントが対応するパス（User の `userName`、`name`、`active`、`emails`、Enterprise 拡張の属性、Group の `displayName`、`members`）への `add`、`replace`、`remove` を送ったとき、Sourcing は、その属性だけを更新し、ほかの属性を変えず、200 と更新後のリソースを返す。
- 対応する属性の一覧にないか、存在しない属性を指す `path` を受けた場合、Sourcing は、400 と scimType `invalidPath` の SCIM のエラーで拒否し、リソースを変えない。
- 読み取り専用の `id`、`meta`、`schemas` を指す `path` を受けた場合、Sourcing は、400 と scimType `mutability` の SCIM のエラーで拒否し、リソースを変えない。
- `op` が `add`、`replace`、`remove` のどれでもないか、RFC 7644 の PATCH の要件を満たさない本文を受けた場合、Sourcing は、400 と scimType `invalidValue` の SCIM のエラーで拒否し、リソースを変えない。
- **例**：EX-SOURCING-004-01、EX-SOURCING-004-02、EX-SOURCING-004-03、EX-SOURCING-004-04

### SCIM クライアントによる Group の同期

#### REQ-SOURCING-005 外部 IdP から SCIM でグループとメンバーシップを同期できる

- SCIM クライアントが Group を作成したとき、Sourcing は、Group を作り、201 と SCIM の Group を返す。
- SCIM クライアントが Group のメンバーを追加または削除したとき、Sourcing は、直接所属する User のメンバーシップを同期し、User の実効ロールに Group のロールを反映する。
- SCIM の Group を返すとき、Sourcing は、各メンバーに `type=User` を付ける。
- SCIM クライアントが Group を削除したとき、Sourcing は、Group を削除し、204 を返す。
- 同じテナントのほかの Group と同じ `displayName` を指定された場合、Sourcing は、409 と scimType `uniqueness` の SCIM のエラーで拒否し、Group を作成も変更もしない。
- 別のテナントの User を指すメンバーを受けた場合、Sourcing は、400 と scimType `invalidValue` の SCIM のエラーで拒否し、メンバーシップを作らない。
- `type` が `User` 以外のメンバーを受けた場合、Sourcing は、400 と scimType `invalidValue` の SCIM のエラーで拒否し、Group を変えない。
- **例**：EX-SOURCING-005-01、EX-SOURCING-005-02、EX-SOURCING-005-03

## セキュリティ上の考慮

SCIM のエンドポイントは、ユーザーのセッションではなく、`ApiTokens` が発行するテナント単位の API アクセストークンで認証し、操作ごとに `scim:users:*` と `scim:groups:*` のスコープを求める。
Discovery のエンドポイントは、`scim:` で始まるいずれかのスコープで参照できる。
全体で一つのトークンを共有する方式は、テナントの間の分離を破るので採らない。

トークンはテナントを束縛する。
要求したレルムがトークンのテナントと一致しなければ、リソースが存在するかを問わず拒否する。
取り込みの途中で解決する参照（Group のメンバー、Enterprise 拡張の `manager`）も同じテナントの中に限り、別のテナントの識別子を指す参照は `invalidValue` として拒否して保存しない。

`DELETE /Users/{id}` は即座の完全な削除にせず、削除を予約する。
理由は、[SCIM の User の削除を論理削除へ統合する](../design/decisions.md#scim-の-user-の削除を論理削除へ統合する)。
