---
status: completed
archived_from: 8fde8e6d80edf2c191cea8b3913e305b829be78d
---

# 宣言済みの Provisioning ライフサイクルイベントを状態遷移とともに発行する

`mise run spec-diff` は main に対する規範の差分が無いと報告した。規範は変えず、既に宣言されていた振る舞いへ実装を合わせた。
管理 API の接続の登録、資格情報の更新、隔離の解除は、保存の後に `ProvisioningConnectionRegistered`、`ProvisioningCredentialRotated`、`ProvisioningConnectionQuarantineCleared` を発行する。
ディスパッチャーはジョブの関連付けで配信を `in_flight` にし、関連付けに成功した worker だけが `ProvisioningDeliveryStarted` を発行する。
`ExecuteDelivery` は `succeeded` の保存後に遷移イベント（`UserProvisioned`、`UserDeprovisioned`、`GroupPushed`）を返し、ジョブハンドラーが連続失敗数の解除の後に発行する。終端状態の配信は再実行されても下流へ送らない。
最後の試行の失敗は `dead_letter` の保存後に `UserProvisioningFailed` を、隔離へ移った失敗は `ConnectionQuarantined` を一度だけ発行する。
`deactivate` 配信は `active=false` を送り、下流に無い User は作らない。必須属性の解決失敗は `ports.ErrRequiredAttributeUnresolved` として伝わり、最初の試行で `dead_letter` になる。これは連続失敗に数えない。
イベントの JSON 項目名を TypeSpec に合わせた。発行ポートは `Module.DispatcherDeps` と `Module.JobHandlerDeps` の引数にして、worker は `NewEmitFunc` を渡す。
`EX-PROVISIONING-003-01`、`004-01`、`005-01`、`010-01`、`014-01`、`017-01`、`018-01` を被覆負債から外した。`EX-PROVISIONING-011-01` は閾値判定が未実装のため、[[wi-61629-quarantine-connections-on-the-accidental-deletion-guard]] を起票して負債の担当を付け替えた。

全文は `git show 8fde8e6d80edf2c191cea8b3913e305b829be78d:<このファイルのパス>` で読む。
