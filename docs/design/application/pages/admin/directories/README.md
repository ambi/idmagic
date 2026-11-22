# ディレクトリ連携

この文書は、ディレクトリ連携のページ一覧を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [管理コンソール](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [ディレクトリ接続](../../../../../modules/sourcing/pages/admin-directory-connectors.md) | `/admin/directory-connectors` | ディレクトリとの接続状態を確認する |
| [ディレクトリ接続を作成](../../../../../modules/sourcing/pages/admin-directory-connector-create.md) | `/admin/directory-connectors/new` | 接続を登録し、登録トークンを取得する |
| [ディレクトリ接続の詳細](../../../../../modules/sourcing/pages/admin-directory-connector-detail.md) | `/admin/directory-connectors/$connectorId` | 接続の登録状態と取り込み結果を確認する |
| [ディレクトリの委任認証](../../../../../modules/sourcing/pages/admin-directory-delegated-authentication.md) | `/admin/directory-connectors/$connectorId/delegated-authentication` | 認証委任の設定と疎通を確認する |
