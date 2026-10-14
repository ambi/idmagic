---
status: completed
archived_from: 2c4d62bd910232b33df0daa84f70a8e6f20e732b
---

# 委譲深さの上限をテナントポリシーにし、トークンが自律実行か代理実行かを判別可能にする

`just spec-diff` が示す意味差分は `REQ-OAUTH2-048`、`REQ-OAUTH2-049`、`REQ-TENANCY-021` と `DelegationMode` 宣言の追加である。Token Exchange の委譲深さはテナントがシステム既定 3 以下へ厳しくでき、解決失敗では拒否される。自律実行、利用者代理、直接実行は 1 つの純関数から introspection と `TokenExchanged` 監査イベントへ出力される。設定は memory / PostgreSQL / テナント API を往復し、管理 UI では実効値と継承状態を表示して上書きと解除を行える。

全文は `git show 2c4d62bd910232b33df0daa84f70a8e6f20e732b:<このファイルのパス>` で読む。
