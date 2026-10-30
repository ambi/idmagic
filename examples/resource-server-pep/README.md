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

## 承認した操作

規則が `requires_approved_operation` を持つ操作では、`Operation.Key` に業務要求ごとの鍵を、`Operation.Parameters` に実際に使う引数の JSON オブジェクトを渡す。
業務処理は `OperationLedger` のように鍵ごとの結果を自身の記録として確定する。
成功と不明の結果は再実行せず、確実に失敗したときだけ同じ鍵で再実行してよい。
結果は応答の `approval_request_id` と `operation_digest` とともに記録する。
この記録がない資源では、IdMagic の承認だけで操作の一回性は成り立たない。

契約は[リソースアクセスの評価](../../docs/modules/oauth2/features/resource-access.md)が定める。
