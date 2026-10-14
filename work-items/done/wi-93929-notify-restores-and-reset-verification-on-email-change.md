---
status: completed
archived_from: fe8791bde7bd3f2f61a451907a91ee765260e711
---

# User の復元を下流へ通知し、管理者がメールアドレスを変えたら確認済みの状態を戻す

`mise run spec-diff` は main に対する規範仕様の差分として REQ-IDMANAGEMENT-045、REQ-IDMANAGEMENT-049、REQ-PROVISIONING-006 の変更を報告した。
管理者が User を復元すると、IdManagement は再有効化（`user_enabled`）として下流のプロビジョニングへ通知する。下流で無効化した User は有効に戻り、削除した User は既存の実行経路が作り直す。
Provisioning は User の再有効化を受けると、接続の状態によらず、すべての接続でその User の猶予期間つきの削除の予約を取り消す。
管理者が `email_verified` を指定せずにメールアドレスを変えると `email_verified` を `false` にし、値が変われば `changed_fields` に `email_verified` を載せる。同じ要求で指定した `email_verified` はその値を保存する。管理画面はすでにこの規則どおりに値を送っており、変更していない。
例 EX-IDMANAGEMENT-049-03 と EX-IDMANAGEMENT-045-03 の期待結果を改め、EX-PROVISIONING-006-03 を加えた。リリースノートを追加した。

全文は `git show fe8791bde7bd3f2f61a451907a91ee765260e711:<このファイルのパス>` で読む。
