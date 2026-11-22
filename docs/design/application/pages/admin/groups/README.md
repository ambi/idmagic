# グループ管理

この文書は、グループ管理のページ一覧を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [管理コンソール](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [グループ一覧](../../../../../modules/identity-management/pages/admin-groups.md) | `/admin/groups` | グループを選び、ロールと所属人数を確認する |
| [グループ詳細](../../../../../modules/identity-management/pages/admin-group-detail.md) | `/admin/groups/$groupId` | グループの属性とメンバーを確認する |
| [グループを作成](../../../../../modules/identity-management/pages/admin-group-create.md) | `/admin/groups/new` | グループの基本情報とカスタム属性を登録する |
| [グループを編集](../../../../../modules/identity-management/pages/admin-group-edit.md) | `/admin/groups/$groupId/edit` | 属性、ロール、所属と動的ルールを管理する |
| [グループの CSV インポート](../../../../../modules/identity-management/pages/admin-group-import.md) | `/admin/groups/import` | CSV の内容を確認してグループを取り込む |
| [グループメンバーの CSV インポート](../../../../../modules/identity-management/pages/admin-group-member-import.md) | `/admin/groups/$groupId/members/import` | 指定グループの所属を CSV で追加または除外する |
| [グループの CSV エクスポート](../../../../../modules/identity-management/pages/admin-group-export.md) | `/admin/groups/exports` | グループ情報を CSV で取得する |
| [グループメンバーの CSV エクスポート](../../../../../modules/identity-management/pages/admin-group-members-export.md) | `/admin/groups/$groupId/members/exports` | 指定グループの所属を CSV で取得する |
