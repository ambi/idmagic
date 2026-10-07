# WI-515: API リファレンスと生成資料の説明文を日本語にする

作業項目は `wi-515-localize-existing-markdown-and-typespec-prose` である。

TypeSpec の説明文を日本語にしたので、OpenAPI の `description` と仕様サイトの API リファレンスが日本語になった。
対象は、`IssueApiToken` のような操作から、`ProblemDetails` や `OAuthError` のようなエラーのボディまで、説明文を持つすべての宣言である。
OpenAPI から生成するクライアントの型のコメントも日本語になる。

説明文のほかは変わらない。
スキーマ、経路、ステータスコード、`operationId`、タグ、Problem Details の `type`（`urn:idmagic:error:*`）は翻訳前と同じである。
API エラーの `detail`、SCIM と SharedSignals のエラーの説明、ロールと権限の説明のように、レスポンスとして返す文章は引き続き英語である。

[設定リファレンス](../../../CONFIGURATION.md)と[経路の優先度リファレンス](../../../ROUTE_PRIORITY.md)も日本語になった。
環境変数名、値の型、クラス名は変わらない。
条件付きで必須の項目は、`SEED_PROFILE != ""` のようなキーの式で条件を示す。
