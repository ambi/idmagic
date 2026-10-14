---
status: completed
archived_from: 3b54259a1368839fba1c5c0a8586ca71d831fffb
---

# 管理ポータルのアクセストークンで共通アカウントコンテキストを取得できるようにする

`/api/auth/account` を管理 portal・account portal・自己管理 API client の共通 bootstrap
endpoint として明文化し、`idmagic.admin`、`idmagic.account`、`account:read` のいずれかで
取得できるよう修正した。scope 判定は `hasRequiredAccountScope` に分離し、共通 endpoint
だけの例外を認証フロー本体から隔離した。

全文は `git show 3b54259a1368839fba1c5c0a8586ca71d831fffb:<このファイルのパス>` で読む。
