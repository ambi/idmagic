---
status: completed
---

# 外部のリソースサーバーが機械認証でエージェントの操作を認可できるようにする

登録したリソースサーバーが提示トークンで自身の操作を評価する `POST /resource-access/evaluate` を追加した（REQ-OAUTH2-062、REQ-OAUTH2-063）。
`McpResourceServer` に評価用の `client_id` と `access_rules` を追加し、管理 API と管理 UI から設定できるようにした（REQ-OAUTH2-064）。
追加した TypeSpec は `EvaluateResourceAccess`、`ResourceAccessRule`、評価の要求と応答、`ResourceAccessEvaluated`、`TemporarilyUnavailableError` であり、`McpResourceServer` の三つのモデルと `OAuthErrorCode` を変えた。
REQ-AUTHORIZATION-004、005、010 と REQ-OAUTH2-054 の文は変えず、管理 API は対話のセッションに限ったままである。

利用者の確認を経て、資源サーバーの認証は `client_id` の束縛、評価できる範囲は登録した規則だけ、許可の再利用はなし、送信者制約の検証は IdMagic 側と決めた。
入口と編成は OAuth2 に置き、関係の判定は既存の依存の向きに沿う `accesscontrol/ports.RelationChecker` を通す。
AuthZEN の Evaluation API は `subject` を必須とするので、適合を主張しない。

参照 PEP は標準ライブラリだけの `examples/resource-server-pep` にあり、製品の組み立てと接続して業務作用の有無を観測した。
確認した範囲は、停止、関係の取り消し、権限のない資源、評価不能、別の audience、別のテナント、未束縛と `Disabled` の登録、DPoP の欠落、別の鍵、再利用、範囲外の `htu` である。
RED は実装の後に入口と検査を一時的に外して観測したので、テストが実装より先行した順序ではない。

残るリスクは次のとおりである。
評価と業務処理の間の時間差は残り、確定した外部の業務操作は取り消さない。
mTLS の拇印はリソースサーバーの申告を信じる。
評価は業務操作ごとの通信を増やすが、容量の仮定には入れておらず、負荷は実測していない。
`/resource-access/evaluate` には流量の制限を掛けていない。
整合トークンの指定は対象外である。
