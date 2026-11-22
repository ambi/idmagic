# AccessControl のページ

この文書は、AccessControl が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[AccessControl のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## 認可

| ページ | URL | 目的 |
| --- | --- | --- |
| [Agent のアクセス割り当て](admin-agent-access.md) | `/admin/authorization/access` | Agent とリソースサーバーの関係を管理する |
| [アクセス割り当てを追加](admin-agent-access-create.md) | `/admin/authorization/access/new` | Agent とリソースの関係を登録する |
| [認可モデル](admin-authorization-model.md) | `/admin/authorization/model` | 公開中または指定バージョンの型と関係を確認する |
| [認可モデルを作成](admin-authorization-model-create.md) | `/admin/authorization/model/new` | 最初の認可モデルを確認して公開する |
| [認可モデルを編集](admin-authorization-model-edit.md) | `/admin/authorization/model/edit` | 型と関係の変更を差分で確認して公開する |
| [関係タプル](admin-authorization-tuples.md) | `/admin/authorization/relation-tuples` | 主体とオブジェクトの関係を検索して削除する |
| [関係タプルを追加](admin-authorization-tuple-create.md) | `/admin/authorization/relation-tuples/new` | 主体、関係とオブジェクトを指定して登録する |
| [アクセス判定の試行](admin-authorization-check.md) | `/admin/authorization/check` | 指定した主体と委任の文脈でアクセスを試す |
