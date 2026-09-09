# WI-523: 内省の `token_type` を提示形式として読み替える

作業項目は `wi-523-introspection-token-type-is-not-the-rfc6749-token-type` である。

`/introspect` の応答の `token_type` を読んでいたリソースサーバーは、読み替えが必要である。この値はトークンの区分名 (`access_token` / `refresh_token`) ではなくなり、RFC 6749 §5.1 の提示形式 (`Bearer` または `DPoP`) になった。

`token_type == "access_token"` で分岐していた実装は、その条件が成立しなくなる。内省したトークンがアクセストークンかどうかを知りたい場合は、リクエストに `token_type_hint` を付けて内省するか、`active` と必要なメタデータだけを読む。提示形式で分岐したい場合は、新しい値をそのまま読む。

リフレッシュトークンの内省の応答からは `token_type` が消えた。このキーの存在を前提にした読み取りは、キーが無い場合を扱えるようにする。

データ移行と設定変更は要らない。`/token` の応答の `token_type` は変わっていないため、発行の経路に手を入れる必要は無い。

互換性境界は [RFC7662-INTROSPECT](../../contexts/oauth2/standards.md#oauth-20-token-introspection) が定める。
