# System のデータ

この文書は、System の Aggregate を扱う。

## 集約

このモジュールは Aggregate を定義しない。
業務状態の正は各モジュールに残る。
共通基盤の受付記録は業務の Aggregate と分け、[HTTP API の共通の振る舞い](features/http-api.md#api-の利用者への応答)と[データベース設計](../../design/data/database.md#再生成可能な認証状態と流量制御)が冪等要求の識別と保存を扱う。

| 概念 | 内容 |
| --- | --- |
| 経路の種類 | ブラウザー向けの認証 API `/api/auth/*`、管理 API `/api/admin/*`、セルフサービス API `/api/account/*`、各標準が定めるパスの OAuth と OIDC のプロトコルのエンドポイント |
| 優先度クラス | `interactive_auth`、`management`、`management_bulk`、`infrastructure` |
| `FeatureRegistry` | 起動処理が静的に組み立てる、実行時に選べる機能の唯一の一覧 |

- **判断**：管理とアカウントのポータルの固定の `client_id` を含む、内部で生成する ID のカラムは、`TEXT` ではなく `UUID` 型とする。
