# 参照 PEP

外部のリソースサーバーが `POST /resource-access/evaluate` で操作を評価する最小の実装である。
標準ライブラリだけを使い、IdMagic の内部型を import しない。

## 接続

1. 管理者が、機密クライアントを作成し、MCP のリソースサーバーの登録に `client_id` と `access_rules` を設定する。
2. リソースサーバーは、業務処理を `Enforcer.Protect` で包む。
3. `decision` が `true` の応答のときだけ業務処理を実行する。

応答は再利用しない。
通信の失敗と 200 以外の応答は、不許可と同じく業務処理を実行しない。
評価の後に確定した業務操作は取り消せない。

契約は[リソースアクセスの評価](../../docs/modules/oauth2/features/resource-access.md)が定める。
