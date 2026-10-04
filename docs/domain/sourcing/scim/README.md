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

### SCIM クライアントによる User の同期

#### REQ-SOURCING-002 外部 IdP から SCIM でユーザーのライフサイクルを同期できる

#### REQ-SOURCING-006 SCIM の複数メールアドレスを正規メールアドレスへ決定的に投影する

  - THEN `invalidValue` の ScimProtocolError を返し、User を変更しない
  - THEN `invalidPath` の ScimProtocolError を返し、User を変更しない

#### REQ-SOURCING-007 SCIM Enterprise 拡張の User 組織属性に対応する

### SCIM クライアントによるリソースの置換

#### REQ-SOURCING-003 外部 IdP は SCIM リソースを PUT で完全に置換できる

### SCIM クライアントによるリソースの部分更新

#### REQ-SOURCING-004 外部 IdP による未対応の PATCH パスや読み取り専用属性への書き込みを拒否する

### SCIM クライアントによる Group の同期

#### REQ-SOURCING-005 外部 IdP から SCIM でグループとメンバーシップを同期できる

## セキュリティ上の考慮

SCIM のエンドポイントは、ユーザーのセッションではなく、`ApiTokens` が発行するテナント単位の API アクセストークンで認証し、操作ごとに `scim:users:*` と `scim:groups:*` のスコープを求める。
Discovery のエンドポイントは、`scim:` で始まるいずれかのスコープで参照できる。
全体で一つのトークンを共有する方式は、テナントの間の分離を破るので採らない。

トークンはテナントを束縛する。
要求したレルムがトークンのテナントと一致しなければ、リソースが存在するかを問わず拒否する。
取り込みの途中で解決する参照（Group のメンバー、Enterprise 拡張の `manager`）も同じテナントの中に限り、別のテナントの識別子を指す参照は `invalidValue` として拒否して保存しない。

`DELETE /Users/{id}` は即座の完全な削除にせず、削除を予約する。
理由は、[SCIM の User の削除を論理削除へ統合する](../design/decisions.md#scim-の-user-の削除を論理削除へ統合する)。
