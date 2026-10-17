# System のデータ

この文書は、System の Aggregate を扱う。

## 集約

このモジュールは Aggregate を定義しない。
起動と経路の組み立てだけを担い、記録の正はすべて各モジュールに残る。

| 概念 | 内容 |
| --- | --- |
| 経路の種類 | ブラウザー向けの認証 API `/api/auth/*`、管理 API `/api/admin/*`、セルフサービス API `/api/account/*`、各標準が定めるパスの OAuth と OIDC のプロトコルのエンドポイント |
| 優先度クラス | `interactive_auth`、`management`、`management_bulk`、`infrastructure` |
| `FeatureRegistry` | 起動処理が静的に組み立てる、実行時に選べる機能の唯一の一覧 |
| 表示言語 | `ja` と `en`。フォールバックは `en` |

- **判断**：管理とアカウントのポータルの固定の `client_id` を含む、内部で生成する ID のカラムは、`TEXT` ではなく `UUID` 型とする。
