---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 未認証で /admin/* を開いたとき login に redirect し、戻り先を保持する

`/admin/*` の初期ロードで account API が 401 を返した場合、現在の path と
query を `return_to` として `/login` へ移動する。ログイン API と TOTP API
は SCL に既定済みの任意 `return_to` を受け取り、認証完了後に元の管理画面
へ戻す。戻り先は現在の realm の `/admin` 配下だけを許可する。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
