# TokenVault のページ

この文書は、TokenVault が主担当となるページの設計を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[TokenVault のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## 外部サービス接続

| ページ | URL | 目的 | 入力項目数 |
| --- | --- | --- | --- |
| [本人の外部サービス接続](account-connections.md) | `/account/connections` | 外部サービスとの接続と再認可、解除を管理する | 0 |
