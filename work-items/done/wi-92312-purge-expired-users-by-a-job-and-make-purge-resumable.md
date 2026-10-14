---
status: completed
archived_from: b6994d15df9723631cc12b722ed7115bc6547f72
---

# 猶予期間を過ぎた User を定期ジョブで完全削除し、途中で失敗した完全削除を再実行で完了できるようにする

`mise run spec-diff -- main` の結果、REQ-IDMANAGEMENT-044 と REQ-IDMANAGEMENT-050 の要件文と、UserLifecycle の状態遷移が変わった。
REQ-IDMANAGEMENT-044 は、猶予期間を過ぎた削除予約の User を、ユーザー一覧の取得ではなく保持期限の削除（Batch の `retention-sweep`）がすべてのテナントで完全削除すると定める。保持期限の削除は匿名化の後に失敗した完全削除を再開し、一人の失敗で残りを止めずに失敗を返す。ユーザー一覧の取得は User を完全削除しない。
REQ-IDMANAGEMENT-050 は、匿名化の後に失敗した完全削除の再実行が、終えていない手順だけを行い、使用量を一度だけ減らし、最初の要求の操作者と理由で `UserDeleted` を発行すると定める。
UserLifecycle の状態遷移表は列「一覧の取得」を「保持期限の削除」に改め、遷移の表のガードを猶予期間ちょうどで完全削除しない `>` に直した。
実装は、Tombstone の `lifecycle` に残りの手順と操作者と理由（`pending_purge`）を記録し、冪等でない使用量の減算と `UserDeleted` の発行を終えるたびに進める。候補は新しい `UserRepository.ListPurgeCandidates` と部分インデックス `users_tenant_purge_candidates_idx` で引く。

全文は `git show b6994d15df9723631cc12b722ed7115bc6547f72:<このファイルのパス>` で読む。
