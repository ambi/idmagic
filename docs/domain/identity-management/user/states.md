# ユーザーの状態遷移

## UserLifecycle

User Aggregate のライフサイクル。`Active` は通常稼働、`Disabled` は復元可能な無効化である。削除のデフォルトは予約であり、`PendingDeletion` に入って猶予期間内は `Active` へ戻せる。`Deleted` は Tombstone 化して匿名化をカスケードした終端状態で、復元できない。

`Deleted` へ入る経路は 2 つあり、どちらも `purge` の指定を要求する点で共通する。管理者が明示的に `purge=true` を指定すればどの状態からでも即時に `Deleted` へ入り、`PendingDeletion` のレコードはそれに加えて、猶予期間（業界で一般的な 7〜30 日に合わせたデフォルト 30 日）の経過でも `Deleted` へ入る。指定のない削除要求は必ず `PendingDeletion` を経由し、すでに `PendingDeletion` の User に対する再要求は冪等な no-op になる。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 通常稼働。認証を許可するのはこの状態だけである |
| Disabled | — | 復元可能な無効化 |
| PendingDeletion | — | 削除予約。猶予期間内は復元できる |
| Deleted | terminal | Tombstone 化して匿名化した。復元できない |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | UserDisabled | — | Disabled |  |
| Disabled | UserEnabled | — | Active |  |
| Active | UserSoftDeleted | — | PendingDeletion |  |
| Disabled | UserSoftDeleted | — | PendingDeletion |  |
| PendingDeletion | UserRestored | — | Active |  |
| PendingDeletion | UserDeleted | input.purge == true \|\| duration_since(status_changed_at) >= duration('2592000s') | Deleted | UserDeleted |
| Active | UserDeleted | input.purge == true | Deleted | UserDeleted |
| Disabled | UserDeleted | input.purge == true | Deleted | UserDeleted |
