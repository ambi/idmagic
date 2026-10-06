# ユーザーの設計

この文書は、[ユーザー](README.md)の機能のうち、コードから読み取れない仕組みと、壊れたときの直し方を書く。
Context 全体の設計は [IdManagement の設計](../design/README.md)が扱う。

## データ

| データ | 仕組み |
| --- | --- |
| 認証の可否 | 認証を許可するのは `status` が `Active` の場合だけである。`Active` に解決されるゼロ値を含め、ほかの値はすべて拒否する |
| 住所の属性 | 平坦なキー（`address_formatted`、`address_locality` など）で保存し、UserInfo と ID Token を作るときだけ入れ子の `address` クレームへ組み直す。`AttributeValue` を文字列、数値、真偽値、日付、文字列の配列の単純な直和型に保つためである |

## 信頼性

完全削除は、Aggregate を破棄せずにその場で Tombstone へ書き換え、次の順に進む。

1. 所有する Agent を無効化する。
2. 再識別と再認証に使える項目を一度に消した Tombstone を保存する。Tombstone の `lifecycle` には、残りの手順（`pending_purge`）と、`UserDeleted` に記録する操作者と理由を入れる。
3. 削除した User から以後たどれてはならない記録を、それを所有する Context のポートを通して一つずつ消す。ポートは呼び出しごとに確定し、ポートをまたぐトランザクションはない。
4. テナントの User の使用量を一つ減らし、残りの手順を手順 5 に進めて Tombstone を保存する。
5. `UserDeleted` を発行し、下流のプロビジョニングへ通知し、`pending_purge` を消して Tombstone を保存する。

手順 3 は何度行っても同じ結果になる。
冪等でない手順 4 と 5 は、終えるたびに `pending_purge` を進めるので、再実行は終えていない手順から再開する。
再実行するのは、同じ User の完全削除の要求と、Batch の `retention-sweep` である。
`retention-sweep` は、各テナントで猶予期間を過ぎた `PendingDeletion` の User と、`pending_purge` を持つ Tombstone を探して完全削除し、一人の失敗で残りを止めない。

| 症状 | 原因 | 直し方 |
| --- | --- | --- |
| 完全削除がエラーを返した後も、User の関連する記録の一部が残る | 手順 2 の後に、手順 3 以降のどれかが失敗した | 同じ User を完全削除し直すか、次の `retention-sweep` を待つ。どちらも残りの手順から再開する |
| テナントの User の使用量が実際の件数より一つ少ない | 手順 4 の使用量の減算は成功し、直後の Tombstone の保存が失敗した後に再実行した | 再実行が減算をもう一度行った。使用量を実際の件数に合わせて手作業で直す |
| `UserDeleted` が二度記録されている | `UserDeleted` に続く反応（所有者の削除による失効エポックの前進）か、手順 5 の Tombstone の保存が失敗した後に再実行した | 再実行が `UserDeleted` をもう一度発行した。失効エポックの前進は何度行っても同じ結果になるので、直す必要はない |
| 猶予期間を過ぎた `PendingDeletion` の User が残っている | `retention-sweep` が動いていないか、その User の完全削除が失敗し続けている | CronJob の実行とログの `user purge failed` を確かめる |
