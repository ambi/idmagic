# Sourcing の標準仕様

## System for Cross-domain Identity Management Core Schema

RFC 7643 — https://www.rfc-editor.org/rfc/rfc7643.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7643-SERVICE-PROVIDER-CONFIG | required | MUST | ServiceProviderConfig は authenticationSchemes を含み Bearer トークン方式を広告する。 |
| RFC7643-CORE-RESOURCES | partial | MUST | User と Group リソースを SCIM Core Schema に従って表現する。 |
| RFC7643-ENTERPRISE-EXTENSION | partial | SHOULD | User リソースは Enterprise 拡張（`urn:ietf:params:scim:schemas:extension:enterprise:2.0:User`）の employeeNumber、department、manager に、Discovery と CRUD / PATCH で対応する。 |

## System for Cross-domain Identity Management Protocol

RFC 7644 — https://www.rfc-editor.org/rfc/rfc7644.html

| ID | Adoption | Strength | Statement |
|---|---|---|---|
| RFC7644-RESOURCE-OPERATIONS | required | MUST | User と Group リソースに作成、参照、置換、削除の操作を提供する。 |
| RFC7644-DELETE-SEMANTICS | partial | MUST | 削除した User の id に対する以後の操作は 404 を返し、コレクションの照会結果、Group の `members`、Enterprise 拡張の `manager` のいずれにも現れない。内部の User レコードは soft delete として残す。削除済みの User は `userName` の一意性判定には残るため、同じ `userName` での再作成は 409 の `uniqueness` になる。 |
| RFC7644-PATCH | partial | SHOULD | User と Group リソースの部分更新を PATCH 操作で提供する。 |
| RFC7644-PATCH-PATH-OPTIONAL | required | MUST | User と Group の PATCH で `path` を省いた `add` と `replace` を受け入れ、`value` のオブジェクトの各メンバーを、その名前を `path` にした操作として、指定したときと同じ許可の一覧と値の検査で適用する。Enterprise 拡張の URN をキーにしたオブジェクトは、各メンバーを URN で修飾した `path` の操作として適用する。`value` が空でないオブジェクトでなければ `invalidValue`、`path` を省いた `remove` は `noTarget` で拒否する。 |
| RFC7644-BEARER-AUTHORIZATION | required | MUST | SCIM プロトコルのエンドポイントは、テナント単位の Bearer トークンで認証・認可する。トークンには ApiTokens Context が発行する API アクセストークンを使用し、SCIM 操作には `scim:users:read` / `scim:users:write` / `scim:groups:read` / `scim:groups:write` のうち該当するスコープを要求する。Discovery エンドポイントは `scim:*` のいずれかで参照できる。 |
| RFC7644-ERROR-RESPONSE | required | MUST | プロトコル上の失敗は、HTTP ステータスと detail を持つ SCIM エラーレスポンスで返す。 |
| RFC7644-FILTERING | partial | SHOULD | コレクション一覧のエンドポイントは、`filter` クエリパラメーターによる絞り込みを提供する。比較値の文字列は RFC 7159 の JSON の文字列として解釈する —— 制御文字をそのまま含む文字列と、`\u` の後が 16 進の 4 桁でない文字列は `invalidFilter` で拒否し、`\u` で書いたサロゲートペアは一つの文字として比べる。`id` の比較は大文字と小文字を区別し (RFC 7643 §3.1 の caseExact)、ほかの文字列の属性は区別しない。ABNF より寛容に、字句の間の空白の有無と量 (タブを含む) と、`true` と `false` の大文字と小文字は問わない —— 受理する入力を ABNF に絞ると、これまで受理していたクライアントの要求が 400 になるためである。 |
