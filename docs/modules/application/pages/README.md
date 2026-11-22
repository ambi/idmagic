# Application のページ

この文書は、Application が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[Application のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## セキュリティ

| ページ | URL | 目的 |
| --- | --- | --- |
| [サインインポリシー](admin-sign-in-policy.md) | `/admin/sign-in-policy` | テナントのデフォルトの規則とアプリごとの上書きを確認する |

## アプリケーション

| ページ | URL | 目的 |
| --- | --- | --- |
| [アプリケーション一覧](admin-applications.md) | `/admin/applications` | 登録アプリケーションを選び、設定と割り当てを確認する |
| [アプリケーション詳細](admin-application-detail.md) | `/admin/applications/$applicationId` | プロトコル、利用者への割り当てと資格情報の状態を確認する |
| [アプリケーションを作成](admin-application-create.md) | `/admin/applications/new` | 接続方式に応じたアプリケーションを登録する |
| [アプリケーションを編集](admin-application-edit.md) | `/admin/applications/$applicationId/edit` | アプリケーションの表示情報、接続・認証設定、資格情報、割り当てとカテゴリを管理する |
| [本人のアプリケーションランチャー](account-apps.md) | `/account/apps` | 利用可能なアプリケーションを起動し表示順を変更する |
