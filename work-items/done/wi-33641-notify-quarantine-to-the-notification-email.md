---
status: completed
archived_from: e20fd77a3842296b0fc169ee3c117276dc2696f0
---

# 接続の隔離を `notification_email` へメールで通知する

`mise run spec-diff` は main に対する規範の差分なしと報告した。`REQ-PROVISIONING-011` の文面は変えず、`EX-PROVISIONING-011-01` が既に宣言していた `notification_email` への通知を実装した。
公開契約では、TypeSpec の `NotificationTemplateKey` に `provisioning_connection_quarantined` を追加した（`mise run check-api-compat` は非互換なしと判定した）。
連続失敗と誤削除ガードのどちらで接続を隔離しても、隔離を保存して `ConnectionQuarantined` を発行した後、`notification_email` へアプリケーション ID、隔離の理由、隔離の時刻を載せたメールを一通送る。宛先が未設定か空白だけなら送らない。
ja と en の組込み既定、差し込み変数の許可集合、`notification_templates` の CHECK、管理画面の型とラベルを揃え、`EX-PROVISIONING-011-01` を被覆の負債台帳から外した。

全文は `git show e20fd77a3842296b0fc169ee3c117276dc2696f0:<このファイルのパス>` で読む。
