---
status: completed
---

# WS-Trust の UsernameToken のパスワード認証を Authentication に委ねる

REQ-WSFEDERATION-004 に、UsernameToken の照合がログインの API と同じ試行の制限、失敗の数え方と集約、認証の失敗のイベントを使うこと、パスワードのポリシーを満たさない資格情報と IP の制限を 401 に含めること、パスワードの変更を求められている User を 403 で拒否することを書いた。
REQ-AUTHENTICATION-007 には、仕様に書かれていなかったログインの API のポリシーの確認を書いた。
二つの判断（WS-Trust でもポリシーを確認する、パスワードの変更を求められている User に発行しない）は利用者に確認した。

まず、ログインの API のパスワード認証を echo に依存しない `PasswordAuthentication` に切り出し、ログインの API をそれに通した。
ログインの API の既存のテストは変えずに成功し、移したロジックの単体テストは同じ表明のまま usecase のパッケージへ移した。
次に、WS-Trust の能動的な STS を `PasswordAuthenticator` へ通し、写しの照合を消した。
受け入れテストで、WS-Trust の失敗が `AuthenticationFailed` を出さないことと、`update_password` を持つ User に発行することを RED にしてから直した。

期限切れのときに `update_password` を付ける案は、User を丸ごと保存する書き手を増やすので採らず、拒否だけにした。
ミューテーションで見つかった、失敗を記録する鍵の種類と IP の制限の分岐の観測の不足は、単体テストを足して塞いだ。
成功の記録のストアが失敗する経路は、切り出す前と同じくテストがない。
WS-Trust のログインの結果の指標とエンドポイント単位のレート制限は対象外に残した。

全文は `git log -1 -- work-items/active/wi-88033-authenticate-ws-trust-passwords-through-authentication.md` が示すコミットの親、またはそのコミットで読む。
