# ユーザーのライフサイクルの操作

この章は、[ユーザー](README.md)の操作のうち、`User` の状態を変える無効化と再有効化、削除の予約、復元、完全削除を扱う。
状態と遷移は、機能仕様の [UserLifecycle](README.md#userlifecycle) が定める。

## 無効化と再有効化

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 対象の User と、無効にするか有効にするか |
| 成功時の作用 | 無効化は `Disabled` にし、`UserDisabled` を発行し、記憶済みの端末をすべて失効させ、所有する `Active` の Agent を無効化する（REQ-IDMANAGEMENT-081）。再有効化は `Active` に戻し、`UserEnabled` を発行する |
| 拒否 | `PendingDeletion` の User の無効化と再有効化（409 `user_pending_deletion`）。`admin` または `system_admin` を持つ管理者が自分自身を無効化する操作（422 `self_disable_forbidden`）。どの拒否も User を変えない |
| 冪等性 | すでにその状態なら、成功を返し、時刻を進めず、イベントを発行しない |

### REQ-IDMANAGEMENT-010 管理者は無効化したユーザーを再有効化できる

- `Disabled` の User の再有効化は、User を `Active` に戻し、`UserEnabled` を発行する。
- **判断**：無効化は削除とは別の、元に戻せる停止である。停止した User のアカウントと履歴は、復帰のために残す。
- **担保手段**：`usecases.SetUserDisabled`
- **例**：EX-IDMANAGEMENT-010-01

### REQ-IDMANAGEMENT-046 User の無効化と再有効化は、すでにその状態なら何もしない

- `Disabled` の User の無効化と、`Active` の User の再有効化は成功を返し、`status_changed_at` と `updated_at` を進めず、イベントを発行しない。
- `admin` または `system_admin` を持つ管理者が自分自身を無効化する操作は、422 と `self_disable_forbidden` で拒否し、User を変えない。自分自身の再有効化は拒否しない。
- 無効化は、その User の記憶済みの端末をすべて失効させる。
- `PendingDeletion` の User の無効化と再有効化は、409 と `user_pending_deletion` で拒否し、User を変えず、イベントを発行しない。
- **判断**：削除の予約を取り消す経路を、猶予期間を確かめて `UserRestored` を発行する復元だけにする。無効化と再有効化を許すと、状態遷移表にない遷移で猶予期間の判定を迂回できる。
- **担保手段**：`usecases.SetUserDisabled`

## 削除の予約

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 対象の User と、任意の理由 |
| 成功時の作用 | `PendingDeletion` にし、理由を載せた `UserSoftDeleted` を発行し、下流のプロビジョニングへ User の削除として通知し、所有する `Active` の Agent を無効化する（REQ-IDMANAGEMENT-081）。個人情報、同意、リフレッシュトークン、セッションは残す |
| 拒否 | `admin` または `system_admin` を持つ管理者が自分自身を対象にする操作（`self_delete_forbidden`） |
| 冪等性 | すでに `PendingDeletion` なら、成功を返しイベントを発行しない |

### REQ-IDMANAGEMENT-011 管理者はユーザーの削除を予約し、猶予期間内に復元できる

- `Active` の User の削除は、User を `PendingDeletion` にし、`UserSoftDeleted` を発行する。
- `PendingDeletion` の User の復元は、User を `Active` に戻し、`UserRestored` を発行する。
- **判断**：削除のデフォルトを予約にして、誤操作を猶予期間の間に取り消せるようにする。
- **担保手段**：`usecases.SoftDeleteUser`、`usecases.RestoreUser`
- **例**：EX-IDMANAGEMENT-011-01

### REQ-IDMANAGEMENT-048 User の削除の予約

- 削除を予約した User の個人情報、同意、リフレッシュトークン、セッションは残す。
- すでに `PendingDeletion` の User の削除の予約は成功を返し、イベントを発行しない。
- `admin` または `system_admin` を持つ管理者が自分自身の削除を予約する操作は、すでに `PendingDeletion` であっても `self_delete_forbidden` で拒否する。
- 要求に含めた理由を `UserSoftDeleted` に記録する。
- 削除の予約を、下流のプロビジョニングへ User の削除として通知する。
- **担保手段**：`usecases.SoftDeleteUser`

### REQ-IDMANAGEMENT-012 削除を予約したユーザーはログインを拒否される (superseded by REQ-PLATFORM-002)

引き金は IdManagement の削除予約、観測はログインの拒否であり、どちらの Context も単独では起こせない。
削除の予約と復元が到達経路の開閉と対応することを、REQ-PLATFORM-002 が保証として述べる。

## 復元

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者 |
| 入力 | 対象の User |
| 成功時の作用 | `Active` に戻し、`UserRestored` を発行する（REQ-IDMANAGEMENT-011）。下流のプロビジョニングへは通知しない |
| 拒否 | 猶予期間を過ぎた User（409 `restore_grace_expired`）、`PendingDeletion` でない User（409 `not_pending_deletion`）、`admin` または `system_admin` を持つ管理者が自分自身を対象にする操作（`self_delete_forbidden`） |

### REQ-IDMANAGEMENT-049 削除を予約した User は、猶予期間の終わりの時刻まで復元できる

- 猶予期間は 30 日であり、`PendingDeletion` になった時刻に猶予期間を加えた時刻ちょうどまで復元できる。
- その時刻を過ぎた復元は、409 と `restore_grace_expired` で拒否する。
- `PendingDeletion` でない User の復元は、409 と `not_pending_deletion` で拒否する。
- `PendingDeletion` になった時刻を持たない User は、いつでも復元できる。
- 管理 API の応答の `purge_after` は、`PendingDeletion` になった時刻に猶予期間を加えた時刻である。
- 復元は、下流のプロビジョニングへ通知しない。
- **担保手段**：`usecases.RestoreUser`
- **要判断**：削除の予約は下流へ User の削除として通知するが、復元は通知しない。下流の宛先では、復元した User が削除されたまま残る。復元も通知するかを決める。

## 完全削除

| 項目 | 内容 |
| --- | --- |
| 行為者 | 管理者。猶予期間を過ぎた削除予約の User は、一覧の取得のときに `system` が完全削除する（REQ-IDMANAGEMENT-044） |
| 入力 | 対象の User と、`purge=true` |
| 成功時の作用 | Tombstone へ置き換えて `Deleted` にし、関連する記録を消し、テナントの User の使用量を一つ減らし、`UserDeleted` を発行し、所有する `Active` の Agent を無効化する（REQ-IDMANAGEMENT-081） |
| 拒否 | `admin` または `system_admin` を持つ管理者が自分自身を対象にする操作（`self_delete_forbidden`） |
| 冪等性 | すでに `Deleted` なら、成功を返し `UserDeleted` を発行しない |

### REQ-IDMANAGEMENT-050 User の完全削除は匿名化であり、削除済みの User には何もしない

- 完全削除は、`Active`、`Disabled`、`PendingDeletion` のどの User にもできる。
- 完全削除は、ユーザー名を `deleted:<sub>` に置き換え、名前、メールアドレス、ロール、属性、必須操作を消し、`email_verified` と `mfa_enrolled` を `false` にし、どのパスワードでも認証できない状態にする。
- 完全削除は、その User の同意、リフレッシュトークン、セッション、パスワードの履歴、TOTP、記憶済みの端末、WebAuthn の資格情報、復旧コード、デバイス認可、承認要求を消す。
- 完全削除は、テナントの User の使用量を一つ減らす。
- すでに `Deleted` の User の完全削除は成功を返し、`UserDeleted` を発行しない。
- **判断**：[User の削除を物理削除ではなく Tombstone で行う](../../design/decisions.md#user-の削除を物理削除ではなく-tombstone-で行う)。
- **担保手段**：`usecases.DeleteUser`

### REQ-IDMANAGEMENT-013 管理者はユーザーを完全削除できる

- `PendingDeletion` の User の完全削除は、User を `Deleted` にし、`UserDeleted` を発行する。
- 操作者自身が対象であり、その対象が `admin` または `system_admin` を持つ場合は、削除の予約、復元、完全削除のいずれも `self_delete_forbidden` で拒否する。
- **判断**：管理者が自分の特権アカウントを消す経路は、どの対話フローにも要らない。
- **担保手段**：`usecases.DeleteUser`
- **例**：EX-IDMANAGEMENT-013-01、EX-IDMANAGEMENT-013-02
