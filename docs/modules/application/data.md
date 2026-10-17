# Application のデータ

この文書は、Application とプロトコル設定の関係を、どう保存して保証するかを扱う。
テーブルの定義はスキーマファイルが定める。

## Application とプロトコル設定の関係

各プロトコルのテーブル（`oauth2_clients`、`saml_service_providers`、`wsfed_relying_parties`）は、`NULL` を許す一意な `application_id` を持つ。
`application_id` が `NULL` でない場合は、テナントと固定のプロトコルの判別子も含む複合外部キーで Application を参照する。
二つのプロトコル設定が同じ Application を参照すること、テーブルをまたいで重複して参照すること、テナントや種別が食い違うことを、データベース自身が拒否する。

OAuth2 のプロトコルのテーブルは `oauth2_clients` とする。
SAML と WS-Fed のテーブルと同じく、プロトコルに固有の標準の用語を使う。

## 作成の原子性

カタログへの作成では、Application のレコードの作成と、プロトコル設定のレコードへの `application_id` の設定を一つのトランザクションで確定する。
後半が失敗しても、カタログにだけ表示される孤立した Application は残らない。
