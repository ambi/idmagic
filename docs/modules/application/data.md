# Application のデータ

この文書は、Application とプロトコル設定の関係を、どう保存して保証するかを扱う。
テーブルの定義はスキーマファイルが定める。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `Application` | 名前、種類（`weblink`、`federated`、`service`）、起動の URL、状態、アイコン、カテゴリ、プロトコル設定への参照（`ApplicationProtocol`） | `Tenant` を `tenant_id` で参照する。プロトコル設定を最大一つ参照する |
| `ApplicationAssignment` | 主体（`user` または `group`）、`visibility` | `Application` と、User または Group を参照する |
| `ApplicationCategory` | テナントの管理者が定義するカテゴリ | `Tenant` を `tenant_id` で参照する |
| `ApplicationOrdering` | `(tenant_id, user_sub)` ごとの `application_id` の並び | User を参照する |

サインイン条件の集約と保存は [SignInPolicy のデータ](../sign-in-policy/data.md)が扱う。
- **判断**：ポータルでの手動の並び順とカテゴリは、IdManagement の User の Aggregate ではなく、`Application` の表示に関する概念なので、このモジュールに置く。

## Application とプロトコル設定の関係

各プロトコルのテーブル（`oauth2_clients`、`saml_service_providers`、`wsfed_relying_parties`）は、`NULL` を許す一意な `application_id` を持つ。
`application_id` が `NULL` でない場合は、テナントと固定のプロトコルの判別子も含む複合外部キーで Application を参照する。
二つのプロトコル設定が同じ Application を参照すること、テーブルをまたいで重複して参照すること、テナントや種別が食い違うことを、データベース自身が拒否する。

OAuth2 のプロトコルのテーブルは `oauth2_clients` とする。
SAML と WS-Fed のテーブルと同じく、プロトコルに固有の標準の用語を使う。

## 作成の原子性

カタログへの作成では、Application のレコードの作成と、プロトコル設定のレコードへの `application_id` の設定を一つのトランザクションで確定する。
Application が開始したトランザクションに、OAuth2、Saml、WsFederation の所有者が公開する結び付け操作を参加させる。
組み立て地点が各所有者の操作を渡し、Application は他のモジュールのテーブルへ直接書かない。
後半が失敗しても、カタログにだけ表示される孤立した Application は残らない。
