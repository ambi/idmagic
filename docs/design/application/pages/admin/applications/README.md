# アプリケーション

この文書は、アプリケーションのページ一覧を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [管理コンソール](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [アプリケーション一覧](../../../../../modules/application/pages/admin-applications.md) | `/admin/applications` | 登録アプリケーションを選び、設定と割り当てを確認する |
| [アプリケーション詳細](../../../../../modules/application/pages/admin-application-detail.md) | `/admin/applications/$applicationId` | プロトコル、利用者への割り当てと資格情報の状態を確認する |
| [アプリケーションを作成](../../../../../modules/application/pages/admin-application-create.md) | `/admin/applications/new` | 接続方式に応じたアプリケーションを登録する |
| [アプリケーションを編集](../../../../../modules/application/pages/admin-application-edit.md) | `/admin/applications/$applicationId/edit` | アプリケーションの表示情報、接続・認証設定、資格情報、割り当てとカテゴリを管理する |
