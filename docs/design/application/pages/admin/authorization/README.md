# 認可

この文書は、認可のページ一覧を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [管理コンソール](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [Agent のアクセス割り当て](../../../../../modules/authorization/pages/admin-agent-access.md) | `/admin/authorization/access` | Agent とリソースサーバーの関係を管理する |
| [アクセス割り当てを追加](../../../../../modules/authorization/pages/admin-agent-access-create.md) | `/admin/authorization/access/new` | Agent とリソースの関係を登録する |
| [認可モデル](../../../../../modules/authorization/pages/admin-authorization-model.md) | `/admin/authorization/model` | 公開中または指定バージョンの型と関係を確認する |
| [認可モデルを作成](../../../../../modules/authorization/pages/admin-authorization-model-create.md) | `/admin/authorization/model/new` | 最初の認可モデルを確認して公開する |
| [認可モデルを編集](../../../../../modules/authorization/pages/admin-authorization-model-edit.md) | `/admin/authorization/model/edit` | 型と関係の変更を差分で確認して公開する |
| [関係タプル](../../../../../modules/authorization/pages/admin-authorization-tuples.md) | `/admin/authorization/relation-tuples` | 主体とオブジェクトの関係を検索して削除する |
| [関係タプルを追加](../../../../../modules/authorization/pages/admin-authorization-tuple-create.md) | `/admin/authorization/relation-tuples/new` | 主体、関係とオブジェクトを指定して登録する |
| [アクセス判定の試行](../../../../../modules/authorization/pages/admin-authorization-check.md) | `/admin/authorization/check` | 指定した主体と委任の文脈でアクセスを試す |
| [アプリ間アクセスの規則](../../../../../modules/oauth2/pages/admin-cross-app-access.md) | `/admin/cross-app-access` | アプリ間の発行と受け入れの規則を管理する |
| [アプリ間の発行規則を追加](../../../../../modules/oauth2/pages/admin-cross-app-access-new-issue.md) | `/admin/cross-app-access/new-issue` | 対象アプリと発行できる範囲を登録する |
| [アプリ間の受け入れ規則を追加](../../../../../modules/oauth2/pages/admin-cross-app-access-new-redeem.md) | `/admin/cross-app-access/new-redeem` | 受け入れる発行者と利用範囲を登録する |
| [ロール一覧](../../../../../modules/oauth2/pages/admin-roles.md) | `/admin/roles` | 定義済みのロールと権限、付与人数を確認する |
| [ロール詳細](../../../../../modules/oauth2/pages/admin-role-detail.md) | `/admin/roles/$name` | 一つのロールの権限と付与先を確認する |
| [認可詳細の型](../../../../../modules/oauth2/pages/admin-authz-detail-types.md) | `/admin/authorization-detail-types` | 認可詳細のスキーマと利用するアプリを管理する |
| [認可詳細の型を作成](../../../../../modules/oauth2/pages/admin-authz-detail-type-create.md) | `/admin/authorization-detail-types/new` | 型の識別子と JSON Schema を登録する |
| [認可詳細の型を編集](../../../../../modules/oauth2/pages/admin-authz-detail-type-edit.md) | `/admin/authorization-detail-types/$type/edit` | 既存の型の説明とスキーマを変更する |
