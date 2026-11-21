# ユーザー管理

この文書は、ユーザー管理のページ一覧を示す。

> 表示確認用の途中プレビュー。全ページの棚卸しと検証は継続中である。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [管理コンソール](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [ユーザー一覧](../../../../../modules/identity-management/pages/admin-users.md) | `/admin/users` | ユーザーを検索し、対象の確認と管理の入口を示す |
| [ユーザー詳細](../../../../../modules/identity-management/pages/admin-user-detail.md) | `/admin/users/$id` | 現在のプロフィール、権限、所属と認証の状態を確認する |
| [ユーザーを追加](../../../../../modules/identity-management/pages/admin-user-create.md) | `/admin/users/new` | 初期資格情報とプロフィールを指定してユーザーを作成する |
| [ユーザーを編集](../../../../../modules/identity-management/pages/admin-user-edit.md) | `/admin/users/$id/edit` | プロフィール、ロールとカスタム属性を一回の保存で変更する |
| [ユーザーの CSV インポート](../../../../../modules/identity-management/pages/admin-user-import.md) | `/admin/users/import` | CSV をプレビューしてから確認を経て適用し、行ごとの結果を確認する |
