# ユーザーのライフサイクルの操作

この章は、[ユーザー](README.md)の操作のうち、`User` の状態を変える無効化と再有効化、削除の予約、復元、完全削除を扱う。
状態ごとの結果は、機能仕様の [UserLifecycle](README.md#userlifecycle) の状態遷移表が定める。

## 無効化と再有効化

### REQ-IDMANAGEMENT-010 User の無効化は `Disabled` に、再有効化は `Active` にして、それぞれイベントを発行する

- 管理者が `Active` の User を無効化したとき、IdManagement は、User を `Disabled` にし、`UserDisabled` を発行し、その User の記憶済みの端末をすべて失効させる。
- 管理者が `Disabled` の User を再有効化したとき、IdManagement は、User を `Active` に戻し、`UserEnabled` を発行する。
- 管理者が User を無効化したとき、IdManagement は、[エージェント](../agent/README.md)の REQ-IDMANAGEMENT-081 のとおり、その User が所有する `Active` の Agent を無効化する。
- **判断**：無効化は削除とは別の、元に戻せる停止である。停止した User のアカウントと履歴は、復帰のために残す。

### REQ-IDMANAGEMENT-046 User の無効化と再有効化は、すでにその状態なら何もせず、管理者自身の無効化と削除予約中の User を拒否する

- User が `Disabled` の間、管理者が User を無効化したとき、IdManagement は、成功を返し、`status_changed_at` と `updated_at` を進めず、イベントを発行しない。
- User が `Active` の間、管理者が User を再有効化したとき、IdManagement は、成功を返し、`status_changed_at` と `updated_at` を進めず、イベントを発行しない。
- `admin` または `system_admin` を持つ管理者が自分自身を無効化しようとした場合、IdManagement は、422 と `self_disable_forbidden` で拒否し、User を変えない。
- 管理者が自分自身を再有効化したとき、IdManagement は、その要求を拒否しない。
- User が `PendingDeletion` の間、無効化または再有効化を要求されたとき、IdManagement は、409 と `user_pending_deletion` で拒否し、User を変えず、イベントを発行しない。
- **判断**：削除の予約を取り消す経路を、猶予期間を確かめて `UserRestored` を発行する復元だけにする。無効化と再有効化を許すと、状態遷移表にない遷移で猶予期間の判定を迂回できる。

## 削除の予約

### REQ-IDMANAGEMENT-011 User の削除の予約は `PendingDeletion` にして `UserSoftDeleted` を発行し、予約済みの User には何もしない

- 管理者が `PendingDeletion` でない User の削除を予約したとき、IdManagement は、User を `PendingDeletion` にし、`UserSoftDeleted` を発行し、その User が所有する `Active` の Agent を無効化する。
- 管理者がすでに `PendingDeletion` の User の削除を予約した場合、IdManagement は、成功を返し、イベントを発行しない。
- **判断**：削除のデフォルトを予約にして、誤操作を猶予期間の間に取り消せるようにする。

### REQ-IDMANAGEMENT-048 User の削除の予約は記録を残し、理由とともに下流へ削除として通知する

- 管理者が User の削除を予約したとき、IdManagement は、その User の個人情報、同意、リフレッシュトークン、セッションを残す。
- 管理者が理由を付けて User の削除を予約したとき、IdManagement は、その理由を `UserSoftDeleted` に記録する。
- 管理者が User の削除を予約したとき、IdManagement は、下流のプロビジョニングへ User の削除として通知する。

### REQ-IDMANAGEMENT-013 特権を持つ管理者自身を対象にする削除の予約、復元、完全削除は拒否する

- `admin` または `system_admin` を持つ管理者が自分自身を対象に削除の予約、復元、完全削除を要求した場合、IdManagement は、`self_delete_forbidden` で拒否し、User を変えない。
- 管理者自身が `PendingDeletion` の間、自分自身の削除の予約を要求されたとき、IdManagement は、`self_delete_forbidden` で拒否する。
- **判断**：管理者が自分の特権アカウントを消す経路は、どの対話フローにも要らない。
- **例**：EX-IDMANAGEMENT-013-03

### REQ-IDMANAGEMENT-012 削除を予約したユーザーはログインを拒否される (superseded by REQ-PLATFORM-002)

引き金は IdManagement の削除予約、観測はログインの拒否であり、どちらの Context も単独では起こせない。
削除の予約と復元が到達経路の開閉と対応することを、REQ-PLATFORM-002 が保証として述べる。

## 復元

### REQ-IDMANAGEMENT-049 削除を予約した User の復元は、猶予期間の終わりの時刻まで `Active` に戻す

- 管理者が `PendingDeletion` の User を復元したとき、IdManagement は、User を `Active` に戻し、`UserRestored` を発行する。
- `PendingDeletion` になった時刻に猶予期間の 30 日を加えた時刻ちょうどまでの間、管理者が User を復元したとき、IdManagement は、User を `Active` に戻す。
- `PendingDeletion` になった時刻を記録していない User では、管理者が User を復元したとき、IdManagement は、時刻によらず User を `Active` に戻す。
- 管理 API の応答で `PendingDeletion` の User を返すとき、IdManagement は、`PendingDeletion` になった時刻に猶予期間を加えた時刻を `purge_after` として返す。
- 管理者が User を復元したとき、IdManagement は、下流のプロビジョニングへ通知しない。
- 猶予期間を過ぎた User の復元を要求された場合、IdManagement は、409 と `restore_grace_expired` で拒否する。
- `PendingDeletion` でない User の復元を要求された場合、IdManagement は、409 と `not_pending_deletion` で拒否する。
- **例**：EX-IDMANAGEMENT-049-01、EX-IDMANAGEMENT-049-02、EX-IDMANAGEMENT-049-03

## 完全削除

### REQ-IDMANAGEMENT-050 User の完全削除は User を匿名化して `Deleted` にし、削除済みの User には何もしない

- 管理者が `Active`、`Disabled`、`PendingDeletion` の User を `purge=true` で削除したとき、IdManagement は、User を `Deleted` にし、`UserDeleted` を発行し、その User が所有する `Active` の Agent を無効化する。
- User を完全削除したとき、IdManagement は、ユーザー名を `deleted:<sub>` に置き換え、名前、メールアドレス、ロール、属性、必須操作を消し、`email_verified` と `mfa_enrolled` を `false` にし、どのパスワードでも認証できない状態にする。
- User を完全削除したとき、IdManagement は、その User の同意、リフレッシュトークン、セッション、パスワードの履歴、TOTP、記憶済みの端末、WebAuthn の資格情報、復旧コード、デバイス認可、承認要求を消す。
- User を完全削除したとき、IdManagement は、テナントの User の使用量を一つ減らす。
- 匿名化の後に完全削除が失敗した User の完全削除を再び要求されたとき、IdManagement は、終えていない手順だけを行い、使用量を一度だけ減らし、最初の要求の操作者と理由で `UserDeleted` を発行する。
- 完全削除を終えた `Deleted` の User の完全削除を要求された場合、IdManagement は、成功を返し、`UserDeleted` を発行しない。
- **判断**：[User の削除を物理削除ではなく Tombstone で行う](../design/decisions.md#user-の削除を物理削除ではなく-tombstone-で行う)。
- **例**：EX-IDMANAGEMENT-050-01、EX-IDMANAGEMENT-050-02
