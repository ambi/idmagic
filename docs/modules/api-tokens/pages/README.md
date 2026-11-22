# ApiTokens のページ

この文書は、ApiTokens が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[ApiTokens のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## API トークン

| ページ | URL | 目的 |
| --- | --- | --- |
| [管理 API トークン](admin-api-tokens.md) | `/admin/api-tokens` | 管理用トークンの範囲、有効期限と失効状態を確認する |
| [管理 API トークンを発行](admin-api-token-create.md) | `/admin/api-tokens/new` | 用途と権限を指定してトークンを発行する |
| [本人の API トークン](account-api-tokens.md) | `/account/api-tokens` | 本人が発行したトークンと権限を確認する |
| [本人の API トークンを発行](account-api-token-create.md) | `/account/api-tokens/new` | 許可された権限の範囲でトークンを発行する |
