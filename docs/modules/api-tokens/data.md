# ApiTokens のデータ

この文書は、ApiTokens の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `ApiToken` | `jti`、発行者、説明、スコープの集合、audience、DPoP の束縛、発行時刻、有効期限、失効時刻 | `Tenant` を `tenant_id` で、発行者の User を `user_id` で参照する |

`ApiTokenScope` は `<resource>:<action>` の形で、`read` は参照、`write` は変更の操作を許可する。
`scim:` で始まるスコープは、SCIM 2.0 のプロビジョニング API のリソースと操作を表す。
スコープの語彙は TypeSpec の `ApiTokenScope` が定める。

- **判断**：発行時に一度だけ返す JWT は、どこにも保存しない。永続化するのは `jti` とライフサイクルのメタデータだけである。したがって、発行済みのトークンの一覧から本文を取り出す経路はなく、紛失への手段は再発行だけになる。
