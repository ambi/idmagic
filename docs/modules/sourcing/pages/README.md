# Sourcing のページ

この文書は、Sourcing が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Sourcing のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## ディレクトリ連携

| ページ | URL | 目的 |
| --- | --- | --- |
| [ディレクトリ接続](admin-directory-connectors.md) | `/admin/directory-connectors` | ディレクトリとの接続状態を確認する |
| [ディレクトリ接続を作成](admin-directory-connector-create.md) | `/admin/directory-connectors/new` | 接続を登録し、登録トークンを取得する |
| [ディレクトリ接続の詳細](admin-directory-connector-detail.md) | `/admin/directory-connectors/$connectorId` | 接続の登録状態と取り込み結果を確認する |
| [ディレクトリの委任認証](admin-directory-delegated-authentication.md) | `/admin/directory-connectors/$connectorId/delegated-authentication` | 認証委任の設定と疎通を確認する |
