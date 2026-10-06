# WI-92312: Purge expired users by a job and make purge resumable

作業項目は `wi-92312-purge-expired-users-by-a-job-and-make-purge-resumable` である。

WI-92312 は、猶予期間を過ぎた削除予約の User を、管理者のユーザー一覧の取得ではなく、Batch の保持期限の削除（`idmagic-batch retention-sweep`）が完全削除するようにする。
あわせて、匿名化の後に途中で失敗した User の完全削除を、再実行で完了できるようにする。

これまでは、管理者がユーザー一覧を取得したときにだけ、猶予期間を過ぎた User を完全削除していた。
一覧を取得しないテナントでは、期限を過ぎた User の個人情報が残り続けた。
変更後は、`retention-sweep` がすべてのテナントで猶予期間を過ぎた User を完全削除し、ユーザー一覧の取得は User を完全削除しない。
デフォルトの配置は `retention-sweep` を毎時実行するので、猶予期間を過ぎた User が残るのは最長で約 1 時間である。
`retention-sweep` を CronJob で動かしていない配置では、期限を過ぎた User が完全削除されなくなるので、定期実行を用意する必要がある。

また、匿名化の後に関連する記録の削除、使用量の減算、`UserDeleted` の発行のどれかが失敗した完全削除は、同じ User の完全削除をもう一度要求するか、次の `retention-sweep` で、終えていない手順から再開する。
再開した完全削除の `UserDeleted` には、最初の要求の操作者と理由を記録する。
規範上の条件は[ユーザー](../../domain/identity-management/user/README.md)の REQ-IDMANAGEMENT-044 と、[ユーザーのライフサイクルの操作](../../domain/identity-management/user/lifecycle.md)の REQ-IDMANAGEMENT-050 が定める。

- 猶予期間を過ぎた削除予約の User を、`retention-sweep` が完全削除する。
- ユーザー一覧の取得は、User を完全削除しない。
- 匿名化の後に失敗した完全削除を、再実行が残りの手順から完了する。
