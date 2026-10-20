---
status: completed
---

# Authentication は User の変える項目だけを更新する

規範仕様の差分はない。
Authentication が User を丸ごと保存していた書き手は、起票時の 3 箇所ではなく、ログインの記録、パスワードの変更と再設定、TOTP の登録、第二要素の増減の同期の 5 箇所あった。
受け入れテストで、ログインとパスワードの変更が、保存の直前に行われた無効化を書き戻すことを RED にしてから直した。

変える値が無効化の状態と同じ JSONB の列 `lifecycle` にあり、期限切れの判定も最新の値から行う必要があるので、列だけを書く更新ではなく、行を `FOR UPDATE` で固定して読み直した最新の User へ変更を当てる `UserRepository.Update` と `UserWriter.Update` を IdManagement に設けた。
5 箇所の書き手はすべてこれを使い、パスワードの再設定はトークンの使用済み化と同じトランザクションで `UserWriter.Update` を使う。
パスワードの変更と再設定は、新しいパスワードを当てる同じ関数を共有する。
データ設計の同時の更新の規則にこの形を加え、適用状況と IdManagement のデータとリスクの文書から Authentication の保存を除いた。

`Update` の意味はメモリと PostgreSQL の共有の契約テストで、再設定の確定が無効化を書き戻さないことはパスワードの永続化の契約テストで確かめた。
TOTP の登録と解除のテストは修正の後に書いたので、2 つの書き手を修正前の版に戻して失敗することを確かめた。
管理者の編集による丸ごとの保存が Authentication の記録を書き戻す逆向きの競合は、wi-605 の版による検出に残る。

全文は `git log -1 -- work-items/active/wi-98381-update-authentication-state-without-overwriting-the-user.md` が示すコミットの親、またはそのコミットで読む。
