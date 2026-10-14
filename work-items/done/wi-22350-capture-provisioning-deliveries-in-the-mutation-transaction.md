---
status: cancelled
archived_from: 994dc060a4b52bf84cd4a58eb808397f8ec353ee
---

# Provisioning の配信行を、発火元の変更と同じトランザクションで作る

取り消す。管理 API の経路で、User と割り当ての変更と配信行を 1 つのトランザクションの作用範囲で確定させる実装を終え、`mise run verify` も通したが、コミットせずに破棄した。
調査の過程で、LifecycleWorkflow による有効化と無効化、SCIM の取り込み、CSV インポート、利用者自身のプロフィール変更が、User を書き換えても配信行を一切作らないことが分かった。書き込み時の原子性はこの漏れに効かない。そのうえ、全保存先を作用範囲へ参加させる `TransactionScopedDB` と、memory の書き戻しの仕組みという横断的な仕組みが必要になり、得られる保証に見合わなかった。
配信の保証は、あるべき状態と下流へ反映済みの状態を突き合わせる照合で行う方針に改め、[[wi-92540-guarantee-provisioning-deliveries-by-reconciliation]] が `REQ-PLATFORM-003` の改訂と照合の実装を引き継ぐ。同じ調査で見つかった削除カスケードの再実行不能とクォータの加算の取りこぼしは、[[wi-83071-make-user-deletion-rerunnable-and-recount-quota-usage]] が扱う。
製品、仕様、文書には変更を加えずに終了する。上の設計の節は、採用しなかった方式の記録として残す。

全文は `git show 994dc060a4b52bf84cd4a58eb808397f8ec353ee:<このファイルのパス>` で読む。
