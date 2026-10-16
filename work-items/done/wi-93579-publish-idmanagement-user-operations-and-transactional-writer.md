---
status: completed
---

# IdManagement のユーザー操作とトランザクションに参加する保存を公開する

IdManagement が `userports.UserWriter` と `pgx.Tx` から作る `userpostgres.UserWriterInTx` を公開した。
パスワードの再設定と IdGovernance のワークフローの記録は、構築関数を必須の引数として受け取り、組み立て地点が渡す。
アカウントのセキュリティ設定の取得は、公開済みの `UserRepository` の読み取りを合成する `LoadSelfUser` へ移し、IdManagement に新しい公開操作を作らなかった（D3）。
`private-import` の負債は 9 件から 4 件になった。

仕様差分は `GetAccountSecurity` の TypeSpec 宣言だけである。
変更前から返していた 404 `user_not_found` が宣言されていなかったので、`GetAccountSecurityError404` を加えて契約を応答に合わせた。

パスワードの再設定には、User の保存の後で書き込みが失敗したときに User の変更も巻き戻るテストを加えた。
両方の経路で、User を別のトランザクションで確定させる変異を手で入れ、テストが検出することを確かめた。

全文は `git log -1 -- work-items/active/wi-93579-publish-idmanagement-user-operations-and-transactional-writer.md` が示すコミットの親、またはそのコミットで読む。
